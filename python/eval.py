"""
eval.py — Model Evaluation & Comparison
----------------------------------------
Evaluates your trained model against Spider dev questions and compares
it against a baseline. Run after training is complete.

Usage:
    python python/eval.py                          # eval your model
    python python/eval.py --compare sqlcoder       # compare vs sqlcoder baseline
    python python/eval.py --n 100                  # eval on first 100 questions
"""

import sys
import os
import re
import json
import sqlite3
import requests
import argparse
from datetime import datetime

# ─── Config ──────────────────────────────────────────────────────────────────
SERVER_URL     = "http://localhost:8000/v1/completions"
DEV_DATA_PATH  = "data/train_spider.json"   # use dev_spider.json if you have it
TABLES_PATH    = "data/tables.json"
DB_BASE_PATH   = "data/database"

PROMPT_TEMPLATE = """### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
Question: {question}
Database: {db_id}
Schema: {schema}

### Response:
RA: """

# ─── Schema loader ───────────────────────────────────────────────────────────
def load_schemas(tables_path):
    with open(tables_path, "r", encoding="utf-8") as f:
        data = json.load(f)
    schemas = {}
    for db in data:
        db_id = db["db_id"]
        tables = db["table_names_original"]
        columns = db["column_names_original"]
        parts = []
        for i, table in enumerate(tables):
            cols = [c[1] for c in columns if c[0] == i]
            parts.append(f"Table {table}: ({', '.join(cols)})")
        schemas[db_id] = "\n".join(parts)
    return schemas

# ─── RA → SQL (Python port, same as train_rl.py) ─────────────────────────────
def strip_aliases(s):
    return re.sub(r'(?i)\bT\d+\.', '', s)

def _matching_paren(s, open_idx):
    depth = 0
    for i, c in enumerate(s[open_idx:]):
        if c == '(':   depth += 1
        elif c == ')':
            depth -= 1
            if depth == 0: return open_idx + i
    return -1

def _find_structural_paren(s, from_idx):
    last_close = s.rfind(')')
    if last_close == -1: return -1
    for i in range(from_idx, len(s)):
        if s[i] == '(' and _matching_paren(s, i) == last_close:
            return i
    return -1

def _split_cols(s):
    s = s.strip()
    if not s or s == '*': return ['*']
    parts, depth, start = [], 0, 0
    for i, c in enumerate(s):
        if c == '(':   depth += 1
        elif c == ')': depth -= 1
        elif c == ',' and depth == 0:
            parts.append(s[start:i].strip())
            start = i + 1
    parts.append(s[start:].strip())
    return parts

def _parse(ra):
    ra = ra.strip()
    op_idx, op_char = -1, None
    for i, c in enumerate(ra):
        if c in ('γ','σ','π'):
            op_idx, op_char = i, c
            break
    if op_idx == -1:
        ra = ra.strip('() ')
        if '⨝' in ra:
            ji = ra.index('⨝')
            left = ra[:ji].strip()
            rest = ra[ji+len('⨝'):].strip()
            cond, right = '', rest
            if rest.startswith('_') or rest.startswith('{'):
                cs, ce = rest.find('{'), rest.find('}')
                if cs != -1 and ce != -1:
                    cond  = rest[cs+1:ce].strip()
                    right = rest[ce+1:].strip()
            return ('join', left, right, cond)
        return ('table', ra.strip())
    sp = _find_structural_paren(ra, op_idx)
    if sp == -1: return ('table', ra.strip())
    lp = _matching_paren(ra, sp)
    if lp == -1: return ('table', ra.strip())
    header = re.sub(r'[^a-zA-Z0-9\s><=!*._(),⨝]', '', ra[op_idx+1:sp]).strip()
    body   = ra[sp+1:lp].strip()
    if op_char == 'γ': return ('agg',  _split_cols(header), _parse(body))
    if op_char == 'σ':
        cond = header if header and header != '*' else '1=1'
        return ('sel', strip_aliases(cond), _parse(body))
    if op_char == 'π': return ('proj', _split_cols(header), _parse(body))
    return ('table', ra.strip())

def _to_source(node):
    if node[0] == 'table': return node[1].replace('⨝', ' NATURAL JOIN ')
    if node[0] == 'join':
        l = node[1].replace('⨝', ' NATURAL JOIN ')
        r = node[2].replace('⨝', ' NATURAL JOIN ')
        return f"{l} JOIN {r} ON {strip_aliases(node[3])}" if node[3] else f"{l} NATURAL JOIN {r}"
    return f"({_to_sql(node)}) AS sub"

def _to_sql(node):
    if node[0] == 'table': return "SELECT * FROM " + node[1].replace('⨝', ' NATURAL JOIN ')
    if node[0] == 'join':  return "SELECT * FROM " + _to_source(node)
    if node[0] == 'sel':   return f"SELECT * FROM {_to_source(node[2])} WHERE {node[1]}"
    if node[0] == 'proj':  return f"SELECT {', '.join(node[1])} FROM {_to_source(node[2])}"
    if node[0] == 'agg':
        fns = ('count','sum','avg','max','min')
        agg   = [c for c in node[1] if any(f in c.lower() for f in fns)]
        group = [strip_aliases(c) for c in node[1] if not any(f in c.lower() for f in fns)]
        sel   = group + agg
        q = f"SELECT {', '.join(sel) or 'COUNT(*)'} FROM {_to_source(node[2])}"
        if group: q += f" GROUP BY {', '.join(group)}"
        return q
    return ""

def ra_to_sql(ra_string):
    ra_string = re.sub(r'^RA:\s*', '', ra_string.strip()).strip()
    if not ra_string: return None
    try:
        sql = _to_sql(_parse(ra_string))
        return sql if sql.strip().upper().startswith("SELECT") else None
    except Exception:
        return None

# ─── Execution ────────────────────────────────────────────────────────────────
def execute_sql(db_id, sql):
    if not sql: return None
    db_path = os.path.join(DB_BASE_PATH, db_id, f"{db_id}.sqlite")
    if not os.path.exists(db_path): return None
    try:
        conn = sqlite3.connect(db_path)
        rows = conn.cursor().execute(sql).fetchall()
        conn.close()
        return sorted([str(r) for r in rows])
    except Exception:
        return None

# ─── LLM call ────────────────────────────────────────────────────────────────
def call_model(question, db_id, schema):
    prompt = PROMPT_TEMPLATE.format(question=question, db_id=db_id, schema=schema)
    try:
        resp = requests.post(SERVER_URL, json={
            "model": "local",
            "prompt": prompt,
            "max_tokens": 128,
            "temperature": 0.0,
            "stop": "\n"
        }, timeout=30)
        data = resp.json()
        return data["choices"][0]["text"].strip()
    except Exception as e:
        return f"ERROR: {e}"

# ─── Metrics ─────────────────────────────────────────────────────────────────
class Metrics:
    def __init__(self, name):
        self.name        = name
        self.total       = 0
        self.exact_match = 0   # SQL result matches gold exactly
        self.exec_ok     = 0   # SQL executed without error
        self.parse_ok    = 0   # RA parsed to valid SQL
        self.ra_valid    = 0   # output contains RA operators
        self.errors      = []

    def record(self, ra, sql, result, gold_result, error=None):
        self.total += 1
        if any(op in (ra or '') for op in ('γ','σ','π','⨝')):
            self.ra_valid += 1
        if sql:
            self.parse_ok += 1
        if result is not None:
            self.exec_ok += 1
        if result is not None and gold_result is not None and result == gold_result:
            self.exact_match += 1
        if error:
            self.errors.append(error)

    def summary(self):
        t = self.total or 1
        return {
            "model":           self.name,
            "total":           self.total,
            "exact_match":     self.exact_match,
            "exact_match_pct": round(100 * self.exact_match / t, 1),
            "exec_accuracy":   round(100 * self.exec_ok / t, 1),
            "parse_rate":      round(100 * self.parse_ok / t, 1),
            "ra_valid_rate":   round(100 * self.ra_valid / t, 1),
        }

def print_metrics(m):
    s = m.summary()
    print(f"\n{'─'*50}")
    print(f"  Model         : {s['model']}")
    print(f"  Questions     : {s['total']}")
    print(f"  Exact match   : {s['exact_match']} / {s['total']}  ({s['exact_match_pct']}%)")
    print(f"  SQL executed  : {s['exec_accuracy']}%  (ran without error)")
    print(f"  SQL generated : {s['parse_rate']}%  (RA parsed to valid SQL)")
    print(f"  RA valid      : {s['ra_valid_rate']}%  (output had RA operators)")
    print(f"{'─'*50}")

# ─── Main evaluation loop ─────────────────────────────────────────────────────
def evaluate(model_name, questions, schemas, n):
    metrics = Metrics(model_name)
    print(f"\n🔍 Evaluating {model_name} on {n} questions...")
    print("  [Progress: . = correct  x = wrong  ! = SQL error  ? = parse fail]\n  ", end="", flush=True)

    for i, entry in enumerate(questions[:n]):
        question = entry["question"]
        db_id    = entry["db_id"]
        gold_sql = entry["query"]
        schema   = schemas.get(db_id, "")

        # Get model output
        ra_output = call_model(question, db_id, schema)

        # Parse and execute
        generated_sql = ra_to_sql(ra_output)
        result        = execute_sql(db_id, generated_sql)
        gold_result   = execute_sql(db_id, gold_sql)

        # Progress indicator
        if result is not None and gold_result is not None and result == gold_result:
            print(".", end="", flush=True)
        elif generated_sql is None:
            print("?", end="", flush=True)
        elif result is None:
            print("!", end="", flush=True)
        else:
            print("x", end="", flush=True)

        if (i + 1) % 50 == 0:
            print(f"\n  [{i+1}/{n}]", end="", flush=True)
            print("\n  ", end="", flush=True)

        metrics.record(ra_output, generated_sql, result, gold_result)

    print()
    return metrics

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--n",       type=int, default=200,   help="Number of questions to evaluate")
    parser.add_argument("--compare", type=str, default=None,  help="Compare against: sqlcoder, baseline")
    parser.add_argument("--out",     type=str, default=None,  help="Save results to JSON file")
    args = parser.parse_args()

    print("Loading data...")
    with open(DEV_DATA_PATH, "r", encoding="utf-8") as f:
        questions = json.load(f)
    schemas = load_schemas(TABLES_PATH)

    n = min(args.n, len(questions))

    # Evaluate your model
    your_metrics = evaluate("Your model (RA-SQL)", questions, schemas, n)
    print_metrics(your_metrics)

    # Compare against a baseline if requested
    if args.compare:
        print(f"\n⚠️  Baseline comparison mode: '{args.compare}'")
        print("   This calls the same server — swap the model in server.py to compare.")
        print("   Start server.py with a different model, then re-run with --compare.")
        print("   Example baseline: a model that just returns 'SELECT * FROM <table>'")

        # Simple heuristic baseline for comparison: always select * from first table
        baseline = Metrics("Baseline (SELECT * heuristic)")
        for entry in questions[:n]:
            db_id    = entry["db_id"]
            gold_sql = entry["query"]
            # Naive: guess SELECT * from first table in gold SQL
            tables = re.findall(r'\bFROM\s+(\w+)', gold_sql, re.IGNORECASE)
            naive_sql = f"SELECT * FROM {tables[0]}" if tables else None
            result     = execute_sql(db_id, naive_sql)
            gold_result = execute_sql(db_id, gold_sql)
            baseline.record("", naive_sql, result, gold_result)

        print_metrics(baseline)

        # Side-by-side comparison
        ys = your_metrics.summary()
        bs = baseline.summary()
        print(f"\n{'─'*50}")
        print(f"  {'Metric':<25} {'Your Model':>12} {'Baseline':>12}")
        print(f"  {'─'*49}")
        for key, label in [
            ("exact_match_pct", "Exact match %"),
            ("exec_accuracy",   "SQL execution %"),
            ("parse_rate",      "SQL parse rate %"),
            ("ra_valid_rate",   "RA valid rate %"),
        ]:
            y = ys[key]
            b = bs[key]
            diff = y - b
            arrow = "▲" if diff > 0 else ("▼" if diff < 0 else "=")
            print(f"  {label:<25} {y:>11}%  {b:>11}%  {arrow} {abs(diff):.1f}%")
        print(f"{'─'*50}")

    # Save results
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    out_path = args.out or f"eval_results_{timestamp}.json"
    with open(out_path, "w", encoding="utf-8") as f:
        results = {"your_model": your_metrics.summary()}
        if args.compare:
            results["baseline"] = baseline.summary()
        json.dump(results, f, indent=2)
    print(f"\n💾 Results saved to {out_path}")
    print(f"\n📊 Quick summary: {your_metrics.exact_match}/{n} exact matches "
          f"({your_metrics.summary()['exact_match_pct']}% accuracy)")

if __name__ == "__main__":
    main()