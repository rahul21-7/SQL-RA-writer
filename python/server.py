import torch

# Windows compatibility patch — must come before any other torch-dependent imports
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

import os
from flask import Flask, request, jsonify
from transformers import AutoModelForCausalLM, AutoTokenizer

app = Flask(__name__)

# ---------------------------------------------------------------------------
# CONFIGURATION
# Set MERGED_MODEL_PATH to your merged model directory.
# After training, merge LoRA adapters once with merge_adapters.py, then point
# this variable at the merged output. This avoids slow PeftModel loading.
# ---------------------------------------------------------------------------
MERGED_MODEL_PATH = os.environ.get("MERGED_MODEL_PATH", "final_model_merged")

print(f"Loading model from '{MERGED_MODEL_PATH}' ...")
tokenizer = AutoTokenizer.from_pretrained(MERGED_MODEL_PATH)
model = AutoModelForCausalLM.from_pretrained(
    MERGED_MODEL_PATH,
    device_map="auto",
    torch_dtype=torch.float16,
)
model.eval()
print("Model loaded.")


def generate_ra(prompt: str) -> str:
    inputs = tokenizer(prompt, return_tensors="pt").to(model.device)
    with torch.no_grad():
        outputs = model.generate(
            **inputs,
            max_new_tokens=128,
            temperature=0.1,
            do_sample=False,       # greedy — matches temperature=0 intent
            repetition_penalty=1.2,
            pad_token_id=tokenizer.eos_token_id,
        )
    # Only decode the newly generated tokens (skip the prompt)
    new_tokens = outputs[0][inputs["input_ids"].shape[1]:]
    decoded = tokenizer.decode(new_tokens, skip_special_tokens=True)
    return decoded.strip()


# ---------------------------------------------------------------------------
# OpenAI-compatible /v1/completions  (used by the Go agent via direct HTTP)
# ---------------------------------------------------------------------------
@app.route("/v1/completions", methods=["POST"])
def completions():
    data = request.json or {}
    prompt = data.get("prompt", "")
    ra_result = generate_ra(prompt)
    return jsonify({
        "id": "cmpl-local",
        "object": "text_completion",
        "choices": [{
            "text": ra_result,
            "index": 0,
            "finish_reason": "stop",
        }],
    })


# ---------------------------------------------------------------------------
# OpenAI-compatible /v1/chat/completions  (kept for other clients / testing)
# ---------------------------------------------------------------------------
@app.route("/v1/chat/completions", methods=["POST"])
def chat_completions():
    data = request.json or {}
    messages = data.get("messages", [])
    prompt = messages[-1]["content"] if messages else data.get("prompt", "")
    ra_result = generate_ra(prompt)
    return jsonify({
        "id": "chatcmpl-local",
        "object": "chat.completion",
        "choices": [{
            "message": {"role": "assistant", "content": ra_result},
            "index": 0,
            "finish_reason": "stop",
        }],
    })


if __name__ == "__main__":
    print("RA AI Server online at http://localhost:8000")
    app.run(host="0.0.0.0", port=8000, threaded=False)