
import torch
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel

LORA_PATH   = "final_model_lora"
OUTPUT_PATH = "final_model_merged"
BASE_MODEL = "unsloth/Llama-3.2-1B"

print(f"Loading base model in fp16: {BASE_MODEL}")
tokenizer = AutoTokenizer.from_pretrained(LORA_PATH)
model = AutoModelForCausalLM.from_pretrained(
    BASE_MODEL,
    torch_dtype=torch.float16,
    device_map="auto",
)

print(f"Attaching LoRA adapters from: {LORA_PATH}")
model = PeftModel.from_pretrained(model, LORA_PATH)

print("Merging adapters into base weights...")
model = model.merge_and_unload()

print(f"Saving merged model to: {OUTPUT_PATH}")
model.save_pretrained(OUTPUT_PATH)
tokenizer.save_pretrained(OUTPUT_PATH)

print(f"✅ Done! Merged model saved to '{OUTPUT_PATH}'.")