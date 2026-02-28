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
# 🦥 LOAD MODEL (Optimized for 6GB VRAM)
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
# 📊 DATASET & PROMPT SETUP
# =========================================================
alpaca_prompt = """Below is an instruction that describes a task. Write a response that appropriately completes the request.

### Instruction:
{}

### Input:
{}

### Response:
{}"""

# Formatting function for the trainer
def formatting_prompts_func(examples):
    instructions = examples["instruction"]
    inputs       = examples["input"]
    outputs      = examples["output"]
    texts = []
    for instruction, input, output in zip(instructions, inputs, outputs):
        text = alpaca_prompt.format(instruction, input, output) + tokenizer.eos_token
        texts.append(text)
    return texts # Returns a list of strings

import torch
from unsloth import FastLanguageModel
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments, EarlyStoppingCallback

# ... [Keep your existing Model Loading and Formatting function code] ...

# =========================================================
# 📊 DATASET SPLIT (The "Cross-Validation" substitute)
# =========================================================
raw_dataset = load_dataset("json", data_files="data/train_ra.json", split="train")

# Split: 90% for training, 10% for validation
dataset_split = raw_dataset.train_test_split(test_size=0.1, seed=3407)
train_dataset = dataset_split["train"]
eval_dataset  = dataset_split["test"]

# =========================================================
# 🚀 TRAINER WITH EARLY STOPPING
# =========================================================
trainer = SFTTrainer(
    model = model,
    processing_class = tokenizer,
    train_dataset = train_dataset,
    eval_dataset = eval_dataset,      # Give the model the test set
    formatting_func = formatting_prompts_func,
    max_seq_length = 2048,
    
    # ADD THE CALLBACK HERE
    callbacks = [EarlyStoppingCallback(early_stopping_patience=3)],
    
    args = TrainingArguments(
        per_device_train_batch_size = 2,
        gradient_accumulation_steps = 4,
        warmup_steps = 5,
        max_steps = 150,              # Increased so Early Stopping has room to work
        learning_rate = 5e-5,         # Gentler learning rate
        bf16 = True,
        logging_steps = 5,
        
        # Evaluation Strategy
        eval_strategy = "steps",      # Evaluate every X steps
        eval_steps = 10,              # Check the test set every 10 steps
        save_strategy = "steps",
        save_steps = 10,
        load_best_model_at_end = True, # Crucial: saves the version with lowest error
        
        optim = "adamw_8bit",
        weight_decay = 0.1,           # Forces generalization
        output_dir = "outputs",
        remove_unused_columns = False,
    ),
)

print("🚀 Starting Training with Validation...")
trainer.train()

# =========================================================
# 💾 SAVE THE "BEST" VERSION
# =========================================================
model.save_pretrained_merged("final_model_merged", tokenizer, save_method = "merged_16bit")
print("✅ Done! The most accurate version has been saved.")