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
# 🛡️ WINDOWS STABILITY PATCHES
# =========================================================
os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
if platform.system() == "Windows":
    def no_compile(model=None, *args, **kwargs):
        if model is None: return lambda x: x
        return model
    torch.compile = no_compile

# =========================================================
# 🚀 LOAD THE TRAINED RA MODEL
# =========================================================
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "final_model_merged", # Points to your newly trained folder
    max_seq_length = 2048,
    load_in_4bit = True,
    device_map = "auto",
)
FastLanguageModel.for_inference(model) # Enable 2x faster inference

# =========================================================
# 📝 PROMPT SETUP (Matches your JSON structure exactly)
# =========================================================
alpaca_prompt = """Below is an instruction that describes a task. Write a response that appropriately completes the request.

### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
Question: {}
Database: {}

### Response:
"""

def chat_loop():
    print("\n--- Relational Algebra Translator (Type 'exit' to quit) ---")
    db_name = input("Enter Database Name (e.g., department_management): ")
    
    while True:
        question = input("\n❓ Question: ")
        if question.lower() == 'exit':
            break
            
        inputs = tokenizer(
            [alpaca_prompt.format("Convert the SQL query to Relational Algebra.", f"Question: {question}\nDatabase: {db_name}", "")], 
            return_tensors = "pt"
        ).to("cuda")

        outputs = model.generate(
            **inputs, 
            max_new_tokens = 128, 
            use_cache = True,
            repetition_penalty = 1.2, # Prevents the model from saying the same word twice
            temperature = 0.1,        # Makes the model more "focused" and less creative
            top_p = 0.9
        )
        
        response = tokenizer.batch_decode(outputs)[0]
        
        if "### Response:" in response:
            ra = response.split("### Response:")[1].split("<|end_of_text|>")[0].strip()
            print(f"🤖 RA: {ra}")

# Call the loop
chat_loop()

#List the name and budget of all departments.