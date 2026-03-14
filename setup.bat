@echo off
SETLOCAL EnableDelayedExpansion

echo ====================================================
echo   AI Project: Relational Algebra to SQL - Setup
echo ====================================================
echo.

:: ─── Check Python ────────────────────────────────────────────────────────────
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Python is not installed or not in PATH.
    echo         Install Python 3.10+ from https://python.org
    pause & exit /b 1
)
for /f "tokens=2 delims= " %%v in ('python --version 2^>^&1') do set PYVER=%%v
echo [OK] Python %PYVER%

:: ─── Check Go ────────────────────────────────────────────────────────────────
go version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH.
    echo         Install Go 1.22+ from https://go.dev/dl
    pause & exit /b 1
)
for /f "tokens=3 delims= " %%v in ('go version') do set GOVER=%%v
echo [OK] Go %GOVER%

:: ─── Virtual Environment ─────────────────────────────────────────────────────
echo.
if not exist "venv" (
    echo [1/5] Creating Python virtual environment...
    python -m venv venv
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to create venv.
        pause & exit /b 1
    )
) else (
    echo [1/5] Virtual environment already exists, skipping.
)

:: ─── Upgrade pip ─────────────────────────────────────────────────────────────
echo.
echo [2/5] Upgrading pip...
.\venv\Scripts\python.exe -m pip install --upgrade pip --quiet

:: ─── PyTorch + CUDA 12.1 ─────────────────────────────────────────────────────
:: Installed separately from requirements.txt because it needs the CUDA index URL.
:: CUDA 12.1 wheel source: https://download.pytorch.org/whl/cu121
:: To use a different CUDA version, change cu121 to e.g. cu118 or cu124 in the URL.
echo.
echo [3/5] Installing PyTorch 2.5.1 with CUDA 12.1...
echo       (Source: https://download.pytorch.org/whl/cu121)
echo       This may take several minutes...
.\venv\Scripts\python.exe -m pip install ^
    torch==2.5.1 ^
    torchvision==0.20.1 ^
    torchaudio==2.5.1 ^
    --index-url https://download.pytorch.org/whl/cu121
if %errorlevel% neq 0 (
    echo [ERROR] PyTorch installation failed.
    echo         Check https://pytorch.org/get-started/locally/ for other CUDA versions.
    pause & exit /b 1
)
echo [OK] PyTorch 2.5.1 + CUDA 12.1 installed.

:: ─── Python Dependencies ─────────────────────────────────────────────────────
echo.
if not exist "requirements.txt" (
    echo [ERROR] requirements.txt not found.
    echo         Run this script from the project root folder.
    pause & exit /b 1
)
echo [4/5] Installing Python dependencies from requirements.txt...
.\venv\Scripts\python.exe -m pip install -r requirements.txt
if %errorlevel% neq 0 (
    echo [ERROR] Failed to install requirements.txt dependencies.
    pause & exit /b 1
)
echo [OK] Python dependencies installed.

:: ─── Go Dependencies ─────────────────────────────────────────────────────────
echo.
echo [5/5] Downloading Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo [ERROR] go mod tidy failed. Check your internet connection.
    pause & exit /b 1
)
echo [OK] Go dependencies ready.

:: ─── Warnings ────────────────────────────────────────────────────────────────
echo.
if not exist "config.json" (
    echo [WARNING] config.json not found.
    echo           Create it in this folder with your PostgreSQL credentials:
    echo           {
    echo             "driver":  "postgres",
    echo             "dsn":     "host=localhost port=5432 user=postgres password=X dbname=myproject sslmode=disable",
    echo             "dialect": "postgres"
    echo           }
)
if not exist "data\tables.json" (
    echo [WARNING] data\tables.json not found.
    echo           Download the Spider dataset and place train_spider.json
    echo           and tables.json inside the data\ folder before training.
)

echo.
echo ====================================================
echo   SETUP COMPLETE!
echo.
echo   Next steps in order:
echo   1. Activate venv:          .\venv\Scripts\activate
echo   2. Prepare training data:  python python\prepare_data.py
echo   3. Train the model:        python python\train.py
echo   4. Merge LoRA adapters:    python python\merge_adapters.py
echo   5. Start the AI server:    python python\server.py
echo   6. Run the Go agent:       go run .
echo ====================================================
pause