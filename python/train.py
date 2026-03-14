import torch
# Patch Torch for Windows stability before Unsloth import
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

os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
os.environ["TORCHINDUCTOR_WORKER_START"] = "spawn"
os.environ["PYTORCH_CUDA_ALLOC_CONF"] = "expandable_segments:True"

if platform.system() == "Windows":
    def no_compile(model=None, *args, **kwargs):
        if model is None: return lambda x: x
        return model
    torch.compile = no_compile
model, tokenizer = FastLanguageModel.from_pretrained(
    model_name = "unsloth/Llama-3.2-1B-bnb-4bit",
    max_seq_length = 640,
    load_in_4bit = True,
    device_map = "auto",
)

# FIX 3: LoRA rank reduced from 16 → 8.
#   r=16 with 7 target modules = ~11M trainable params on a 1B model.
#   r=8 halves the adapter memory with negligible quality drop for this task.
#   lora_alpha kept at 16 (alpha/r = 2.0, a good ratio for fast learning).
model = FastLanguageModel.get_peft_model(
    model,
    r = 8,
    target_modules = ["q_proj", "k_proj", "v_proj", "o_proj",
                      "gate_proj", "up_proj", "down_proj"],
    lora_alpha = 16,
    lora_dropout = 0,
    bias = "none",
    use_gradient_checkpointing = "unsloth",  # Unsloth's checkpointing saves ~30% VRAM
)

# =========================================================
# 2. Prompt Template — identical to agent.go and test.py
# =========================================================
prompt_style = """### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
{}

### Response:
RA: {}"""

def formatting_prompts_func(examples):
    inputs  = examples["input"]
    outputs = examples["output"]
    texts = []
    for i, o in zip(inputs, outputs):
        text = prompt_style.format(i, o) + tokenizer.eos_token
        texts.append(text)
    return texts

# =========================================================
# 3. Load Dataset — filter examples that are too long
# The crash "Input IDs of shape [1, 521] > max 512" happens when
# a prompt is longer than max_seq_length. We tokenize every example
# upfront and drop anything over MAX_SEQ_LEN so the trainer never
# sees an oversized input. Typical keep rate is 95-99%.
# =========================================================
MAX_SEQ_LEN = 640

dataset = load_dataset("json", data_files="data/train_ra.json", split="train")

def is_within_length(example):
    text = prompt_style.format(example["input"], example["output"]) + tokenizer.eos_token
    token_count = len(tokenizer(text, add_special_tokens=True)["input_ids"])
    return token_count <= MAX_SEQ_LEN

before = len(dataset)
dataset = dataset.filter(is_within_length)
after  = len(dataset)
print(f"Dataset: kept {after}/{before} examples (dropped {before - after} over {MAX_SEQ_LEN} tokens)")

# =========================================================
# 4. Trainer Setup
# FIX 4: per_device_train_batch_size reduced from 4 → 1.
#   batch_size=4 with seq_len=2048 was the primary OOM cause.
#   We compensate with gradient_accumulation_steps=16 to keep
#   the effective batch size at 16 (same as before: 1 x 16 = 16).
#
# FIX 5: dataloader_pin_memory=False
#   Pin memory pre-allocates CPU RAM for faster GPU transfers,
#   but on Windows it can cause extra VRAM pressure. Disable it.
#
# FIX 6: dataloader_num_workers=0
#   Windows multiprocessing with PyTorch dataloaders is unstable
#   and can leak memory. 0 = use the main process only.
# =========================================================
trainer = SFTTrainer(
    model = model,
    tokenizer = tokenizer,
    train_dataset = dataset,
    formatting_func = formatting_prompts_func,
    max_seq_length = MAX_SEQ_LEN,   # Must match the value above
    args = TrainingArguments(
        per_device_train_batch_size = 1,       # FIX 4: was 4
        gradient_accumulation_steps = 16,      # FIX 4: was 4 — keeps effective batch = 16
        warmup_steps = 10,
        max_steps = 300,
        learning_rate = 2e-4,
        fp16 = not torch.cuda.is_bf16_supported(),
        bf16 = torch.cuda.is_bf16_supported(),
        logging_steps = 10,                    # Less console spam
        eval_strategy = "no",
        save_strategy = "no",
        optim = "adamw_8bit",
        weight_decay = 0.01,
        output_dir = "outputs",
        report_to = "none",
        dataloader_pin_memory = False,         # FIX 5
        dataloader_num_workers = 0,            # FIX 6
    ),
)

print("🚀 Training starting (6GB GPU mode)...")
trainer.train()
model.save_pretrained("final_model_lora")
tokenizer.save_pretrained("final_model_lora")
print("✅ Done! LoRA adapters saved in 'final_model_lora'.")