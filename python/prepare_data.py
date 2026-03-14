import json
import sqlglot
from sqlglot import exp

def translate_to_ra(sql_query):
    try:
        node = sqlglot.parse_one(sql_query, read="sqlite")
        
        # 1. TABLE SOURCE: Handle Joins
        tables = [t.name for t in node.find_all(exp.Table)]
        unique_tables = list(dict.fromkeys(tables))
        # Use the ⨝ symbol that your Go parser expects
        table_str = " ⨝ ".join(unique_tables) if unique_tables else "UNKNOWN_TABLE"
        
        # 2. SELECTION (σ)
        where_node = node.find(exp.Where)
        where_str = f"σ {where_node.this.sql()} " if where_node else ""
        
        # 3. PROJECTION/AGGREGATE (π / γ)
        select_node = node.find(exp.Select)
        projections = []
        is_agg = False
        
        if select_node:
            for expr in select_node.expressions:
                proj_sql = expr.sql().lower()
                projections.append(proj_sql)
                if any(fn in proj_sql for fn in ["count", "sum", "avg", "max", "min"]):
                    is_agg = True
        
        proj_str = ", ".join(projections) if projections else "*"
        op = "γ" if is_agg else "π"
        
        # 4. UNIFORM NESTING
        if where_str:
            return f"{op} {proj_str} ({where_str}({table_str}))"
        else:
            return f"{op} {proj_str} ({table_str})"
            
    except Exception as e:
        return f"ERROR: {str(e)}"

def load_schemas(tables_path):
    """Extracts schema strings from Spider tables.json"""
    with open(tables_path, 'r', encoding='utf-8') as f:
        schema_data = json.load(f)
    
    schemas = {}
    for db in schema_data:
        db_id = db['db_id']
        table_names = db['table_names_original']
        column_names = db['column_names_original']
        
        formatted_tables = []
        for i, table_name in enumerate(table_names):
            # Get columns belonging to this table
            cols = [col[1] for col in column_names if col[0] == i]
            formatted_tables.append(f"{table_name} {{ {', '.join(cols)} }}")
        
        schemas[db_id] = " | ".join(formatted_tables)
    return schemas

def prepare_dataset(input_path, tables_path, output_path):
    # Load raw spider data
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # Load database schemas
    schemas = load_schemas(tables_path)
    
    ra_dataset = []
    for entry in data:
        db_id = entry['db_id']
        ra_query = translate_to_ra(entry['query'])
        
        if "ERROR" in ra_query: 
            continue 

        # MATCHES GO AGENT PROMPT STRUCTURE EXACTLY
        schema_info = schemas.get(db_id, "Schema not found")
        
        input_block = (
            f"Question: {entry['question']}\n"
            f"Database: {db_id}\n"
            f"Schema: {schema_info}"
        )

        ra_dataset.append({
            "instruction": "Convert the SQL query to Relational Algebra.",
            "input": input_block,
            "output": ra_query
        })
    
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(ra_dataset, f, indent=4)

if __name__ == "__main__":
    # Ensure you have 'tables.json' from the Spider dataset in your data folder
    prepare_dataset('data/train_spider.json', 'data/tables.json', 'data/train_ra.json')
    print("✅ Dataset generated with Schema info! Model will now learn to use column names.")