FROM python:3.11-slim

WORKDIR /app

# Install system dependencies for sentencepiece and torch
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies (use transformers for proper NLLB support)
RUN pip install --no-cache-dir torch==2.5.1 --index-url https://download.pytorch.org/whl/cpu
RUN pip install --no-cache-dir \
    transformers==4.46.0 \
    sentencepiece==0.2.0 \
    fastapi==0.115.0 \
    uvicorn==0.31.0

# Copy server
COPY server.py .

# The model will be downloaded on first request via HuggingFace
ENV HF_HOME=/app/hf_cache

EXPOSE 5000

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "5000"]