# fastapi-bff

## Crear entorno
python -m venv .venv

## Activar Linux/macOS
source .venv/bin/activate

## Activar Windows PowerShell
.venv\Scripts\Activate.ps1

## Instalar dependencias
pip install -e .

## Ejecutar
uvicorn app.main:app --reload --host 0.0.0.0 --port 8000
