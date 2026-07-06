FROM python:3.11-slim

# Install patchelf for fixing executable stack issue
RUN apt-get update && apt-get install -y --no-install-recommends \
    patchelf \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install Python dependencies
RUN pip install --no-cache-dir \
    ctranslate2==4.4.0 \
    fastapi==0.115.0 \
    uvicorn==0.31.0 \
    huggingface-hub==0.26.0 \
    sentencepiece==0.2.0

# Fix ctranslate2 native library executable stack issue
RUN find /usr/local -name "libctranslate2*.so*" -exec patchelf --clear-execstack {} \; 2>/dev/null; echo "Library stack cleared"

# Copy scripts
COPY server.py .
COPY download_model.py .

# Download the model at build time
RUN python download_model.py

ENV MODEL_DIR=/app/models/nllb-200-distilled-600M-ct2-int8

EXPOSE 5000

CMD ["uvicorn", "server:app", "--host", "0.0.0.0", "--port", "5000"]