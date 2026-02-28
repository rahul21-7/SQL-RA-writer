import torch
import os
from flask import Flask, request, jsonify
from transformers import AutoModelForCausalLM, AutoTokenizer

app = Flask(__name__)

# Path to your merged model folder
model_path = "final_model_merged"

print("🤖 Loading Tokenizer and Model...")

# 1. Use AutoTokenizer to handle the Llama 3 BPE format correctly
tokenizer = AutoTokenizer.from_pretrained(model_path)

# 2. Use AutoModel for Causal LM
model = AutoModelForCausalLM.from_pretrained(
    model_path,
    torch_dtype=torch.float16,
    device_map="auto",
    low_cpu_mem_usage=True
)

@app.route('/v1/completions', methods=['POST'])
# Change this line in python/server.py:
@app.route('/v1/chat/completions', methods=['POST']) # Add /chat/ here
def completions():
    data = request.json
    
    # LangChainGo sends the prompt inside a "messages" list for Chat
    # We need to extract the content from the last message
    if "messages" in data:
        prompt = data["messages"][-1]["content"]
    else:
        prompt = data.get("prompt", "")
    
    inputs = tokenizer(prompt, return_tensors="pt").to("cuda")
    
    with torch.no_grad():
        outputs = model.generate(
            **inputs, 
            max_new_tokens=128, 
            temperature=0.1,
            repetition_penalty=1.5
        )
    
    decoded = tokenizer.decode(outputs[0], skip_special_tokens=True)
    
    try:
        ra_result = decoded.split("### Response (RA):")[-1].strip()
    except:
        ra_result = decoded.strip()

    # Return in Chat Completion format
    return jsonify({
        "choices": [{
            "message": {
                "role": "assistant",
                "content": ra_result
            },
            "text": ra_result # Keep this for backward compatibility
        }]
    })

if __name__ == '__main__':
    print("🚀 RA AI Server online at http://localhost:8000")
    # threaded=False helps avoid CUDA memory fragmentation on laptop GPUs
    app.run(host='0.0.0.0', port=8000, threaded=False)