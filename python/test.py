"""
test.py
-------
Interactive CLI to test the trained RA model locally.
Uses the SAME prompt template as train.py and agent.go.
"""
# 1. MUST BE FIRST: Patch Torch for Windows
import torch
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

import os
import platform
from unsloth import FastLanguageModel

# =========================================================
# Windows Stability Patches
# =========================================================
os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
if platform.system() == "Windows":
    def no_compile(model=None, *args, **kwargs):
        if model is None:
            return lambda x: x
        return model
    torch.compile = no_compile

# =========================================================
# Load the trained model
# =========================================================
MODEL_PATH = "final_model_merged"  # Use the merged model (run merge_adapters.py first)

model, tokenizer = FastLanguageModel.from_pretrained(
    model_name=MODEL_PATH,
    max_seq_length=2048,
    load_in_4bit=True,
    device_map="auto",
)
FastLanguageModel.for_inference(model)

# =========================================================
# Prompt template — MUST be identical to train.py prompt_style
# and agent.go promptTemplate
# =========================================================
PROMPT_TEMPLATE = """### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
Question: {question}
Database: {db_id}
Schema: {schema}

### Response:
RA: """


def run_inference(question: str, db_id: str, schema: str) -> str:
    prompt = PROMPT_TEMPLATE.format(
        question=question,
        db_id=db_id,
        schema=schema,
    )
    inputs = tokenizer([prompt], return_tensors="pt").to("cuda")
    outputs = model.generate(
        **inputs,
        max_new_tokens=128,
        use_cache=True,
        do_sample=False,           # greedy — matches temperature=0 intent
        repetition_penalty=1.2,
    )
    # Decode only the newly generated tokens
    new_tokens = outputs[0][inputs["input_ids"].shape[1]:]
    response = tokenizer.decode(new_tokens, skip_special_tokens=True)
    return response.strip()


def chat_loop():
    print("\n--- Relational Algebra Translator (type 'exit' to quit) ---")
    db_id = input("Enter Database Name (e.g., department_management): ").strip()
    schema = input("Enter Schema (or press Enter to skip): ").strip()

    while True:
        question = input("\n❓ Question: ").strip()
        if question.lower() == "exit":
            break
        ra = run_inference(question, db_id, schema)
        print(f"🤖 RA: {ra}")


if __name__ == "__main__":
    chat_loop()