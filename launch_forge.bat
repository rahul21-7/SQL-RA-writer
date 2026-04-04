@echo off
echo 🔥 Forging the SQL Stream...
echo 🤖 Starting AI Inference Server (Python)...
start "SQL Forge - AI Server" cmd /k ".\venv\Scripts\python.exe python\server.py"

echo ⏳ Waiting for model to warm up...
timeout /t 8 /nobreak > nul

echo 🚀 Launching The SQL Forge Web Interface (Go)...
start "SQL Forge - Web Interface" cmd /k "go run main.go web"

echo ✅ Both systems are initializing. 
echo 🌐 Visit http://localhost:8080 to access The Forge.
pause
