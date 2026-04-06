import json
import os
import random
import torch
import platform
from collections import defaultdict
from datasets import Dataset

# Patch Torch for Windows stability
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
        return None

    grouped = defaultdict(list)
    with open(log_path, "r", encoding="utf-8") as f:
        for line in f:
            data = json.loads(line.strip())
            key = (data.get("question", ""), data.get("db_id", ""), data.get("schema_info", ""))
            grouped[key].append(data)

    prompts, chosens, rejecteds = [], [], []
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
        chosen = [i for i in items if i.get("rating") == "2"]
        rejected = [i for i in items if i.get("rating") in ["0", "1"]]
        
        if chosen and rejected:
            for c in chosen:
                for r in rejected:
                    prompts.append(prompt_style.format(question=question, db_id=db_id, schema=schema))
                    chosens.append(c.get("predicted_ra", ""))
                    rejecteds.append(r.get("predicted_ra", ""))

    if not prompts:
        prompts = [prompt_style.format(question="test", db_id="test", schema="test")]
        chosens, rejecteds = ["γ test (table)"], ["SELECT * FROM table"]

    return Dataset.from_dict({"prompt": prompts, "chosen": chosens, "rejected": rejecteds})

def main():
    dataset = load_rl_dataset("feedback_log.jsonl")
    if not dataset: return
    
    model, tokenizer = FastLanguageModel.from_pretrained(
        model_name="unsloth/Llama-3.2-1B-bnb-4bit",
        max_seq_length=640,
        load_in_4bit=True,
        device_map="auto"
    )

    # TRULY CUSTOM MODEL: Freeze core transformer blocks (Layers 1-12)
    # This allows the model to retain its base SQL logic while specializing 
    # the higher-order reasoning layers on our custom relational algebra task.
    print("🧊 Freezing core transformer blocks (Layers 1-12)...")
    for name, param in model.named_parameters():
        if any(f"layers.{i}." in name for i in range(12)):
            param.requires_grad = False

    model = FastLanguageModel.get_peft_model(
        model,
        r=8,
        target_modules=["q_proj", "k_proj", "v_proj", "o_proj", "gate_proj", "up_proj", "down_proj"],
        lora_alpha=16,
        lora_dropout=0,
        bias="none",
        use_gradient_checkpointing="unsloth"
    )

    trainer = ORPOTrainer(
        model=model,
        args=ORPOConfig(
            output_dir="outputs_rlhf",
            per_device_train_batch_size=1,
            gradient_accumulation_steps=16,
            learning_rate=2e-5,
            max_steps=50,
            beta=0.1,
            optim="adamw_8bit",
            remove_unused_columns=False,
            max_prompt_length=512,
            max_length=640,
        ),
        train_dataset=dataset,
        tokenizer=tokenizer,
    )

    print("🚀 Starting Truly Custom RLHF (ORPO) Training...")
    trainer.train()
    model.save_pretrained("rlhf_model_lora")
    tokenizer.save_pretrained("rlhf_model_lora")

if __name__ == "__main__":
    main()
