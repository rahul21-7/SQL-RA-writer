# The SQL Forge: Project Architecture & Technologies Reference

An in-depth technical resource mapping the components of **The SQL Forge** relational algebra translation pipeline. Use this guide to understand how the frontend, backend, parser, and ML components integrate.

---

## 🛠️ Technology Stack Breakdown

This project bridges statistical AI logic (using LLMs) with deterministic database compilation (using parsed Relational Algebra). The technologies used are grouped below:

### 1. The Interactive Presentation Layer (Frontend)
*   **Web Mechanics**: HTML5 & Vanilla CSS. Built around a dark-mode cybernetic dashboard theme utilising custom CSS variables and the Google Fonts *Outfit* (headings) and *JetBrains Mono* (code).
*   **Graphics & Shader (Three.js)**: Runs a responsive, real-time WebGL rendering canvas (`static/js/forge.js`). It visualises a glowing green **"Data Core"** using revolving cylinders, floating particle point rings (`THREE.Points`), and orbiting text sprites of SQL keywords (e.g. `SELECT`, `JOIN`, `π`).
*   **Animations (GSAP)**: GreenSock Animation Platform handles smooth tweening transitions for the 3D-camera, input slide panels, load status indicators, and screen entrance effects to build a premium feel.

### 2. The Core Translator Engine (Go Backend)
*   **Go Standard Library HTTP Server**: Uses `net/http` to host the static file server and expose server routes:
    *   `POST /api/query`: Forwards natural language prompts to the AI engine, obtains predicted Relational Algebra (RA) strings, parses them, and translates them to standard SQL.
    *   `POST /api/execute`: Runs generated SQL on the configured database engine.
    *   `POST /api/feedback`: Appends human feedback ratings (Correct, Partial, Wrong) and comments to a JSONL log file.
    *   `POST /api/stats`: Serves metadata and aggregation values of human alignment.
    *   `GET /api/schema`: Runs SQL introspection queries to discover database layouts.
    *   `GET /api/benchmarks`: Serves saved accuracy evaluation metrics.
*   **Database Introspection (`internal/db/executor.go`)**: Automatically queries system schemas (e.g., `sqlite_master` or `information_schema.columns`) to discover current table metadata (table names, column types) dynamically to construct LLM schema prompts.
*   **Relational Algebra AST Compiler (`internal/algebra/`)**:
    *   `parser.go`: A recursive-descent compiler parsing RA operators like Projection ($\pi$), Selection ($\sigma$), Rename ($\rho$), Join ($\bowtie$), and Aggregation ($\gamma$).
    *   `translator.go`: Traverses the parsed Abstract Syntax Tree (AST) recursively and outputs standardized SQL queries adhering to standard DBMS dialects.

### 3. The AI & Reinforcement Learning Layer (Python Engine)
*   **Inference Server (`python/server.py`)**: Runs a lightweight Flask API on port 8000 exposing OpenAI-compatible completion endpoints. It loads model weights quantized to 4-bit precision to enable rapid inference on consumer-grade hardware.
*   **Model Optimization (Unsloth)**: Integrates *Unsloth* (built on PyTorch and Triton kernels). Unsloth speeds up LLM fine-tuning and inference (up to 2-5x faster) and uses 4-bit/8-bit QLoRA weight parameter configurations to reduce VRAM footprints.
*   **Supervised Fine-Tuning (SFT) (`python/train.py`)**: Train scripts that map inputs of format `Schema + Schema Details + NL Question` to correct `Relational Algebra` tokens.
*   **RLHF Human Alignment via ORPO (`python/train_rl.py`)**: Uses ORPO (Odds Ratio Preference Optimization) to align the model.
*   **Weight Merging (`python/merge_adapters.py`)**: Collapses LoRA weights (adapters) back into the base model's floating-point weights, outputting a single deployment-ready model folder (`final_model_merged`).

---

## 📂 File Structure Map

```text
SQL-RA-writer/
│
├── data/                               # Database & Dataset Assets
│   ├── database/                       # Extracted SQLite DBs (166 items from Spider)
│   ├── tables.json                     # Spider dataset schemas (columns, foreign keys)
│   └── train_spider.json               # Spider training questions and target SQL
│
├── docs/                               # Documentation & Guides
│   ├── architecture_guide.md           # [This File] Complete architecture schematic
│   └── evaluation_guide.md             # Guidelines for running benchmarks
│
├── internal/                           # Go Backend Implementation
│   ├── agent/                          # Prompt constructors and Python API connectors
│   ├── algebra/                        # Relational Algebra Parser & SQL Compiler
│   │   ├── parser.go                   # Recursive token scanner and compiler
│   │   └── translator.go                   # AST-to-SQL translator
│   ├── db/                             # Schema discovery and SQL execution engines
│   └── server/                         # Go HTTP web routers and endpoints
│
├── python/                             # AI Engine, Training, & Evaluation
│   ├── download_db.py                  # Downloader for Spider SQLite database folder
│   ├── eval.py                         # Offline accuracy benchmarking tool
│   ├── merge_adapters.py               # Collapses LoRA weights into base weights
│   ├── prepare_data.py                 # Converted NL queries to RA dataset targets
│   ├── server.py                       # Local Flask API server for inference hosting
│   ├── train.py                        # QLoRA Supervised Fine-Tuning (SFT) script
│   └── train_rl.py                     # ORPO Alignment script (combining SFT + RLHF)
│
├── static/                             # Frontend client files
│   ├── css/                            # Dashboard layout & styles
│   ├── js/                             # UI interactions, HTTP fetches & Three.js 3D core
│   └── index.html                      # Entrypoint client interface loading page
│
├── .gitignore                          # List of files excluded from Git tracking
├── config.json                         # Configuration file for active database driver
├── feedback_log.jsonl                  # Appended human thumbs-rating logs
├── go.mod / go.sum                     # Go module definitions
├── launch_forge.bat                    # Script to start Go servers instantly
├── main.go                             # Entrypoint for Go CLI ("web", etc.)
├── README.md                           # Main repository deployment guide
└── setup.bat                           # Script to configure virtualenv & packages
```

---

## 🧠 Core Machine Learning Concepts

### 1. LoRA & QLoRA (Low-Rank Adaptation)
Training full LLM models is highly compute-intensive. **LoRA** solves this by keeping the original model weights frozen and introducing thin, trainable rank-decomposition matrices ($A$ and $B$) into the self-attention blocks. 
*   **QLoRA** extends this logic by quantizing the base model's static parameters to 4-bit NormalFloat (NF4) while maintaining the tiny LoRA adapters in 16-bit float format. This allows 7B/8B parameter models to be trained and aligned on standard consumer GPUs (under 12GB VRAM).

### 2. RLHF via ORPO (Odds Ratio Preference Optimization)
Historically, aligning models using Reinforcement Learning from Human Feedback (RLHF) required a multi-step sequence: training a Supervised Fine-Tuning (SFT) model, mapping a Reward Model, and adjusting logits with PPO (Proximal Policy Optimization).
*   **ORPO** simplifies this into a single step. It adds an odds ratio penalty directly to the standard cross-entropy loss. While the SFT loss teaches the model *how to speak and parse*, the odds-ratio loss penalises the model for generating rejected outputs (like incorrect syntax or logic flagged in `feedback_log.jsonl`) compared to correct ones.

### 3. Layer Freezing
During ORPO alignment, the script (`python/train_rl.py`) freezes the first 12 layers of the target base model:
```python
# Freeze layers 1 to 12
for name, param in model.named_parameters():
    if any(f"layers.{i}." in name for i in range(12)):
        param.requires_grad = False
```
*   **Why?** In Deep Neural Networks, the early layers are responsible for capturing general token structures, grammar, and basic syntactic logic. Higher-level layers capture complex logical relationships (like Relational Algebra semantics and condition joining). 
*   Freezing early layers prevents catastrophic forgetting of base language skills and stabilizes convergence while editing parameters in higher layers.
