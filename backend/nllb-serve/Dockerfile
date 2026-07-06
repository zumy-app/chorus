FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for sentencepiece
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
RUN pip install --no-cache-dir \
    ctranslate2==4.4.0 \
    fastapi==0.115.0 \
    uvicorn==0.31.0 \
    huggingface-hub==0.26.0 \
    sentencepiece==0.2.0

# Copy the server script
COPY server.py .

# Download the model at build time
RUN python -c "\
import huggingface_hub; \
from pathlib import Path; \
model_name = 'nllb-200-distilled-600M-ct2-int8'; \
repo_id = f'JustFrederik/{model_name}'; \
local_dir = Path(f'/app/models/{model_name}'); \
if not local_dir.exists(): \
    huggingface_hub.snapshot_download(repo_id=repo_id, local_dir=str(local_dir)); \
print('Model downloaded successfully'); \
"

ENV MODEL_DIR=/app/models/nllb-200-distilled-600M-ct2-int8

EXPOSE 5000

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "5000"]