import json
import sqlglot
from sqlglot import exp

def translate_to_ra(sql_query):
    try:
        # Parse using SQLite dialect to match Spider dataset
        node = sqlglot.parse_one(sql_query, read="sqlite")
        
        # 1. UNIFORM JOINS: Replace comma-separated tables with explicit Join (⨝)
        # This allows the Go translator.go to identify Join nodes correctly
        tables = [t.name for t in node.find_all(exp.Table)]
        if len(tables) > 1:
            table_str = " ⨝ ".join(tables)
        else:
            table_str = tables[0] if tables else "UNKNOWN_TABLE"
        
        # 2. SELECTION (σ): Wrap the filter logic
        where_node = node.find(exp.Where)
        where_str = f"σ {where_node.this.sql()} " if where_node else ""
        
        # 3. PROJECTION (π) & AGGREGATE (γ)
        select_node = node.find(exp.Select)
        projections = [p.alias_or_name for p in select_node.expressions] if select_node else []
        proj_str = ", ".join(projections)
        
        # Determine operator
        is_agg = any(kw in sql_query.upper() for kw in ["COUNT", "SUM", "AVG", "MAX", "MIN"])
        op = "γ" if is_agg else "π"
        
        # 4. EMPTY AGGREGATE FIX: Ensure γ always has a target
        if is_agg and (not proj_str or proj_str == "*"):
            proj_str = "count(*)"
            
        # 5. UNIFORM PARENTHESES: Standardize as Op Cols ( Select ( Table ) )
        # This matches the robust regex in the updated algebra/parser.go
        return f"{op} {proj_str} ({where_str}({table_str}))"
        
    except Exception as e:
        return f"ERROR: {str(e)}"

def prepare_dataset(input_path, output_path):
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    ra_dataset = []
    for entry in data:
        ra_query = translate_to_ra(entry['query'])
        
        if "ERROR" in ra_query: 
            continue 

        ra_dataset.append({
            "instruction": "Convert the SQL query to Relational Algebra.",
            "input": f"Question: {entry['question']}\nDatabase: {entry['db_id']}",
            "output": ra_query
        })
    
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(ra_dataset, f, indent=4)

if __name__ == "__main__":
    # Generate the improved, uniform dataset
    prepare_dataset('data/train_spider.json', 'data/train_ra.json')
    print("✅ Uniform RA dataset generated with Joins and explicit Aggregates!")