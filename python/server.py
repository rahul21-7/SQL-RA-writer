import torch
# 1. Mandatory Patch for Windows compatibility
for i in range(1, 9):
    attr = f"int{i}"
    if not hasattr(torch, attr):
        setattr(torch, attr, torch.int8)

from flask import Flask, request, jsonify
from transformers import AutoModelForCausalLM, AutoTokenizer
from peft import PeftModel

app = Flask(__name__)

# Paths
lora_path = "final_model_lora" 
base_model = "unsloth/Llama-3.2-1B-bnb-4bit"

print("🤖 Loading Base Model and Adapters...")
tokenizer = AutoTokenizer.from_pretrained(lora_path)
model = AutoModelForCausalLM.from_pretrained(
    base_model,
    device_map="auto",
    dtype=torch.float16
)

# Attach your trained RA logic
model = PeftModel.from_pretrained(model, lora_path)
model.eval()

@app.route('/v1/chat/completions', methods=['POST'])
def completions():
    data = request.json
    prompt = data["messages"][-1]["content"] if "messages" in data else data.get("prompt", "")
    
    inputs = tokenizer(prompt, return_tensors="pt").to("cuda")
    with torch.no_grad():
        outputs = model.generate(**inputs, max_new_tokens=128, temperature=0.1)
    
    decoded = tokenizer.decode(outputs[0], skip_special_tokens=True)
    
    # Extraction logic for uniformity with train.py
    try:
        ra_result = decoded.split("### Response:")[-1].strip()
    except:
        ra_result = decoded.strip()

    return jsonify({"choices": [{"message": {"role": "assistant", "content": ra_result}}]})

if __name__ == '__main__':
    print("🚀 RA AI Server online at http://localhost:8000")
    app.run(host='0.0.0.0', port=8000, threaded=False)