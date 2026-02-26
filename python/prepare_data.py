import json
import sqlglot
from sqlglot import exp

def translate_to_ra(sql_query):
    try:
        # Parse the query using the SQLite dialect (closest to Spider)
        node = sqlglot.parse_one(sql_query, read="sqlite")
        
        # 1. Extract all table names found in the query
        tables = [t.name for t in node.find_all(exp.Table)]
        table_str = ", ".join(tables) if tables else "UNKNOWN_TABLE"
        
        # 2. Extract Where conditions (Selection - σ)
        where_node = node.find(exp.Where)
        where_str = f"σ_{{{where_node.this.sql()}}} " if where_node else ""
        
        # 3. Extract Projections/Aggregations (π or γ)
        # We look for the main SELECT expressions
        select_node = node.find(exp.Select)
        projections = [p.alias_or_name for p in select_node.expressions] if select_node else []
        proj_str = ", ".join(projections)
        
        # Determine if it's an aggregation (Gamma) or simple projection (Pi)
        is_agg = any(kw in sql_query.upper() for kw in ["COUNT", "SUM", "AVG", "MAX", "MIN"])
        op = "γ" if is_agg else "π"
        
        # Final assembly: Operator_{Columns} ( Selection ( Tables ) )
        return f"{op}_{{{proj_str}}} ({where_str}({table_str}))"
        
    except Exception as e:
        return f"ERROR: {str(e)}"

def prepare_dataset(input_path, output_path):
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    ra_dataset = []
    for entry in data:
        ra_query = translate_to_ra(entry['query'])
        
        # Skip errors to keep training data clean
        if "ERROR" in ra_query: continue 

        ra_dataset.append({
            "instruction": "Convert the SQL query to Relational Algebra.",
            "input": f"Question: {entry['question']}\nDatabase: {entry['db_id']}",
            "output": ra_query
        })
    
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(ra_dataset, f, indent=4)

if __name__ == "__main__":
    prepare_dataset('data/train_spider.json', 'data/train_ra.json')
    print("✅ RA dataset generated successfully with table names!")