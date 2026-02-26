import os
import torch
import platform

# =========================================================
# 🛡️ THE COMPLETE COMPATIBILITY SHIELD
# =========================================================

# 1. FIX: module 'torch' has no attribute 'int1', 'int2', etc.
# We loop through possible missing types and map them to int8
for i in range(1, 8):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

# 2. FIX: module 'torchao.quantization' has no attribute 'Float8WeightOnlyConfig'
try:
    import torchao.quantization
    if not hasattr(torchao.quantization, "Float8WeightOnlyConfig"):
        class Dummy: pass
        torchao.quantization.Float8WeightOnlyConfig = Dummy
except (ImportError, AttributeError):
    pass

# 3. FIX: module 'torch._inductor' has no attribute 'config'
try:
    import torch._inductor.config
except ImportError:
    pass

# 4. Windows Stability Environment Variables
os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
os.environ["TORCHINDUCTOR_WORKER_START"] = "spawn"

if platform.system() == "Windows":
    # This version accepts ANY arguments Unsloth throws at it
    def no_compile(model=None, *args, **kwargs):
        # If used as a decorator @torch.compile(...)
        if model is None:
            return lambda x: x
        # If used as a function torch.compile(model)
        return model
    
    torch.compile = no_compile

# =========================================================
# 🦥 START UNSLOTH
# =========================================================
from unsloth import FastLanguageModel
from datasets import load_dataset
from trl import SFTTrainer
from transformers import TrainingArguments

# Load Model
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/llama-3-8b-bnb-4bit",
    max_seq_length = 2048,
    load_in_4bit = True,
)

# Add LoRA
model = FastLanguageModel.get_peft_model(
    model,
    r = 16,
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",],
    lora_alpha = 16,
    bias = "none",    
    use_gradient_checkpointing = "unsloth",
)

# Load Dataset
dataset = load_dataset("yahma/alpaca-cleaned", split = "train[:1000]") # Small slice for testing

# Train
trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    dataset_text_field = "instruction", # Matches Alpaca format
    max_seq_length = 2048,
    args = TrainingArguments(
        per_device_train_batch_size = 2,
        gradient_accumulation_steps = 4,
        max_steps = 60,
        learning_rate = 2e-4,
        fp16 = True,
        logging_steps = 1,
        output_dir = "outputs",
    ),
)

print("🚀 Environment Stabilized. Starting Training...")
trainer.train()