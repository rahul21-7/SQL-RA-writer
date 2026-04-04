import json
import os
import random
import torch
import platform
from collections import defaultdict
from datasets import Dataset

# Patch Torch for Windows stability before Unsloth import
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

from unsloth import FastLanguageModel
from trl import ORPOTrainer, ORPOConfig

os.environ["UNSLOTH_USE_FLEX_ATTENTION"] = "0"
os.environ["TORCHINDUCTOR_WORKER_START"] = "spawn"
os.environ["PYTORCH_CUDA_ALLOC_CONF"] = "expandable_segments:True"

if platform.system() == "Windows":
    def no_compile(model=None, *args, **kwargs):
        if model is None: return lambda x: x
        return model
    torch.compile = no_compile

def load_rl_dataset(log_path):
    if not os.path.exists(log_path):
        print(f"File {log_path} not found.")
        return None

    # Group by question and schema to find chosen vs rejected pairs
    grouped = defaultdict(list)
    with open(log_path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            data = json.loads(line)
            key = (data.get("question", ""), data.get("db_id", ""), data.get("schema_info", ""))
            grouped[key].append(data)

    prompts = []
    chosens = []
    rejecteds = []

    prompt_style = """### Instruction:
Convert the SQL query to Relational Algebra.

### Input:
Question: {question}
Database: {db_id}
Schema: {schema}

### Response:
RA: """

    for key, items in grouped.items():
        question, db_id, schema = key
        
        # separate into chosen (2) and rejected (0, 1)
        chosen_items = [i for i in items if i.get("rating") == "2"]
        rejected_items = [i for i in items if i.get("rating") in ["0", "1"]]
        
        # Create pairs. If no direct pairing exists, skip. 
        # In a real environment, you'd curate this more explicitly.
        if chosen_items and rejected_items:
            for c in chosen_items:
                for r in rejected_items:
                    prompt = prompt_style.format(question=question, db_id=db_id, schema=schema)
                    prompts.append(prompt)
                    chosens.append(c.get("predicted_ra", ""))
                    rejecteds.append(r.get("predicted_ra", ""))

    if not prompts:
        # Fallback dummy data just to demonstrate the pipeline
        print("Warning: No valid chosen/rejected pairs found. Using dummy data to demonstrate pipeline.")
        prompts = [prompt_style.format(question="test", db_id="test", schema="test")]
        chosens = ["γ test (table)"]
        rejecteds = ["SELECT * FROM table"]

    return Dataset.from_dict({
        "prompt": prompts,
        "chosen": chosens,
        "rejected": rejecteds
    })

def main():
    print("Loading RL dataset...")
    dataset = load_rl_dataset("feedback_log.jsonl")
    if not dataset:
        return
    
    print(f"Loaded {len(dataset)} preference pairs.")

    print("Loading model for RLHF...")
    model, tokenizer = FastLanguageModel.from_pretrained(
        model_name="unsloth/Llama-3.2-1B-bnb-4bit", # Or your preferred base model
        max_seq_length=640,
        load_in_4bit=True,
        device_map="auto"
    )

    model = FastLanguageModel.get_peft_model(
        model,
        r=8,
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj",
                        "gate_proj", "up_proj", "down_proj"],
        lora_alpha=16,
        lora_dropout=0,
        bias="none",
        use_gradient_checkpointing="unsloth"
    )

    print("Setting up ORPO Trainer...")
    # ORPO uses an odds ratio penalty to favor chosen over rejected without a reference model
    orpo_config = ORPOConfig(
        output_dir="outputs_rlhf",
        per_device_train_batch_size=1,
        gradient_accumulation_steps=16,
        learning_rate=2e-5, # lower learning rate for alignment
        max_steps=50,
        beta=0.1, # ORPO beta parameter
        optim="adamw_8bit",
        remove_unused_columns=False,
        report_to="none",
        dataloader_pin_memory=False,
        dataloader_num_workers=0,
        max_prompt_length=512,
        max_length=640,
    )

    trainer = ORPOTrainer(
        model=model,
        args=orpo_config,
        train_dataset=dataset,
        tokenizer=tokenizer,
    )

    print("🚀 Starting RLHF (ORPO) Training...")
    trainer.train()

    save_dir = "rlhf_model_lora"
    model.save_pretrained(save_dir)
    tokenizer.save_pretrained(save_dir)
    print(f"✅ RLHF Complete! LoRA adapters saved in '{save_dir}'.")

if __name__ == "__main__":
    main()
