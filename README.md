# The SQL Forge: NL to Relational Algebra Pipeline

An advanced end-to-end framework for translating Natural Language questions into precise Relational Algebra (RA) and executing them on live databases. This project features a cinematic Web Interface, a robust Go-based backend, and a Python-powered AI engine aligned via Reinforcement Learning from Human Feedback (RLHF).

---

## Key Features

- **Relational Algebra First**: Translates natural language into RA logic before SQL generation, ensuring mathematical precision.
- **Cinematic Web Interface**: Built with Three.js and GSAP for a premium, data-visualized experience.
- **Truly Custom Model Configuration**: Specialized training script supporting "Frozen Layer" optimization (Layers 1-12) to retain core SQL logic while specializing in RA.
- **Schema Discovery**: Automatic database introspection to build precise LLM prompts without manual configuration.
- **ORPO Alignment**: Integrated preference optimization (Odds Ratio Preference Optimization) for fine-tuning performance based on real users' feedback.

---

## Architecture

- **Frontend**: HTML5, Vanilla CSS, Three.js (3D visualization), GSAP (animations).
- **Backend**: Go (Go-SQLite for local DBs, HTTP server for API management).
- **AI Engine**: Python, Unsloth (for 4-bit quantization and efficient LoRA), TRL (for ORPO training).

---

## Getting Started

### 1. Prerequisites
- **Go** (v1.21+)
- **Python** (v3.10+)
- **PostgreSQL** (Optional, for production mode)

### 2. Configuration
Create a `config.json` in the root directory:
```json
{
  "driver": "postgres",
  "dsn": "host=localhost port=5432 user=postgres password=mysecretpassword dbname=myproject sslmode=disable",
  "dialect": "postgres"
}
```

### 3. Environment Setup
We recommend using the included `setup.bat` for automatic Windows configuration:
```bash
# Or manual setup
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### 4. Running the Project

**Step 1: Start the AI Inference Server**
```bash
python python/server.py
```

**Step 2: Launch the Web UI**
```bash
go run main.go web
# Or use the shortcut
launch_forge.bat
```

---

## Model Training (RLHF)

To train your own "Truly Custom" model using the feedback gathered in the Forge:
1. Collect feedback via the UI (Correct/Partial/Wrong buttons).
2. Run the specialized training script:
```bash
python python/train_rl.py
```
*Note: This script implements local layer freezing (Layers 1-12) to maximize training stability.*

---

## Benchmarks

| Metric | Expert Forge (Qwen 1.5B) | Base Logic (Llama 1B) |
| :--- | :--- | :--- |
| **Exact Match %** | 92.0% | 84.0% |
| **Execution Accuracy** | 94.8% | 88.2% |
| **RA Validity** | 100.0% | 98.5% |

---

*This project was developed for advanced study in Natural Language to Logic translation.*
