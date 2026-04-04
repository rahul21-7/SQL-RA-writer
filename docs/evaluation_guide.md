# SQL Model Evaluation Guide

To get official accuracy numbers for your models (Llama 3.2 vs. Qwen 2.5), you should use the `python/eval.py` script. This script runs the model against the standard Spider dataset and compares execution results to get a precise accuracy metric.

## Quick Benchmark Workflow

### Step 1: Evaluate the Base Model (Llama 3.2)
Ensure `python/train.py` (or your server config) is pointing to the Llama model, then run:
```bash
python python/eval.py --n 100 --out results_llama.json
```
*   `--n 100`: Evaluates on the first 100 questions.
*   `--out`: Saves the metrics to a JSON file.

### Step 2: Evaluate the Specialized Model (Qwen 2.5 Coder)
Update your configuration to use the Qwen model and run:
```bash
python python/eval.py --n 100 --out results_qwen.json
```

### Step 3: Compare Results
To see a head-to-head comparison table with delta (▲ / ▼) indicators:
```bash
python python/eval.py --compare results_llama.json
```
Wait, if you already have `results_qwen.json`, you can compare any two files:
```bash
python python/eval.py --compare results_llama.json --out results_comp.json
```

---

## Metrics Explained

- **Exact Match %**: How often the SQL executed results matched the "gold" SQL results perfectly.
- **SQL Execution %**: The percentage of queries that ran without any syntax or runtime errors.
- **SQL Parse Rate %**: How often the Relational Algebra was successfully converted into valid SQL syntax.
- **RA Valid Rate %**: Whether the model correctly used Relational Algebra operators (`γ`, `σ`, etc).

---

## FAQ

> [!NOTE]
> **Does the UI update automatically?**
> Yes, the "ACCURACY" stat in the web UI is a live metric based on your actual usage feedback in the dashboard. However, `eval.py` is the industry-standard way to benchmark the model on thousands of unseen questions.

> [!TIP]
> **Running on more questions**:
> To get a full scientific benchmark, run on all questions by omitting `--n` or setting it to a high number (e.g., 2000).
