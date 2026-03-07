# 1. MUST BE FIRST: Patch Torch before Unsloth is imported
import torch
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

import os
import platform
from unsloth import FastLanguageModel
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments

# =========================================================
# 🛡️ WINDOWS STABILITY PATCHES
# =========================================================
os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
os.environ["TORCHINDUCTOR_WORKER_START"] = "spawn"

if platform.system() == "Windows":
    def no_compile(model=None, *args, **kwargs):
        if model is None: return lambda x: x
        return model
    torch.compile = no_compile

# =========================================================
# 🦥 LOAD MODEL
# =========================================================
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Llama-3.2-1B-bnb-4bit",
    max_seq_length = 2048,
    load_in_4bit = True,
    device_map = "auto",
)

model = FastLanguageModel.get_peft_model(
    model,
    r = 16,
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",
                      "gate_proj", "up_proj", "down_proj"],
    lora_alpha = 16,
    lora_dropout = 0,
    bias = "none",    
    use_gradient_checkpointing = "unsloth",
)

# =========================================================
# 📊 DATASET & PROMPT SETUP (Aligned with Go Agent)
# =========================================================
alpaca_prompt = """Below is an instruction that describes a task. Write a response that appropriately completes the request.

### Instruction:
{}

### Input:
{}

### Response:
{}"""

def formatting_prompts_func(examples):
    instructions = examples["instruction"]
    inputs       = examples["input"]
    outputs      = examples["output"]
    texts = []
    for instruction, input, output in zip(instructions, inputs, outputs):
        # The input field already contains "Question: ... \nDatabase: ..."
        text = alpaca_prompt.format(instruction, input, output) + tokenizer.eos_token
        texts.append(text)
    return texts

# Load your prepared RA dataset
dataset = load_dataset("json", data_files="data/train_ra.json", split="train")

# =========================================================
# 🚀 TRAINER (Fixed Logging Error)
# =========================================================
trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    formatting_func = formatting_prompts_func,
    max_seq_length = 2048,
    args = TrainingArguments(
        per_device_train_batch_size = 2,
        gradient_accumulation_steps = 4,
        warmup_steps = 5,
        max_steps = 300,
        learning_rate = 2e-4,
        fp16 = not torch.cuda.is_bf16_supported(),
        bf16 = torch.cuda.is_bf16_supported(),
        logging_steps = 1,
        # Set eval_strategy to "no" to prevent the Logging TypeError
        eval_strategy = "no", 
        save_strategy = "steps",
        save_steps = 50,
        optim = "adamw_8bit",
        weight_decay = 0.01,
        output_dir = "outputs",
        report_to = "none", # Stop external logging to improve stability
    ),
)

# Replace the end of python/train.py with this:
print("🚀 Starting Training...")
trainer.train()

# Save as LoRA instead of Merged to avoid the disk/RAM crash
model.save_pretrained("final_model_lora") 
tokenizer.save_pretrained("final_model_lora")
print("✅ Done! LoRA adapters saved in 'final_model_lora'.")