#!/bin/bash
# =============================================================================
# Translator-Engine entrypoint
# Downloads the GGUF translation model on first run (cached in MODEL_DIR),
# then starts the llama.cpp server with an OpenAI-compatible API.
# =============================================================================
set -euo pipefail

# ---- Configuration (overridable via environment variables) ------------------
HF_REPO="${HF_REPO:-mradermacher/Synatra-7B-v0.3-Translation-GGUF}"
HF_FILE="${HF_FILE:-Synatra-7B-v0.3-Translation.Q4_K_S.gguf}"
MODEL_DIR="${MODEL_DIR:-/models}"
MODEL_PATH="${MODEL_DIR}/${HF_FILE}"

HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-5000}"
CTX_SIZE="${CTX_SIZE:-2048}"
THREADS="${THREADS:-4}"
N_GPU_LAYERS="${N_GPU_LAYERS:-0}"
TEMPERATURE="${TEMPERATURE:-0.1}"
MAX_TOKENS="${MAX_TOKENS:-512}"

# ---- Download model if not cached ------------------------------------------
if [ ! -f "${MODEL_PATH}" ]; then
    echo "======================================================================"
    echo " Model not found in cache. Downloading..."
    echo "   Repo : ${HF_REPO}"
    echo "   File : ${HF_FILE}"
    echo "   Dest : ${MODEL_PATH}"
    echo "======================================================================"

    mkdir -p "${MODEL_DIR}"

    # Prefer huggingface-cli if available (pip install huggingface-hub)
    if command -v huggingface-cli &>/dev/null; then
        huggingface-cli download "${HF_REPO}" "${HF_FILE}" \
            --local-dir "${MODEL_DIR}" \
            --resume-download
    else
        # Fallback: direct download via curl
        echo "huggingface-cli not found; falling back to curl download."
        DOWNLOAD_URL="https://huggingface.co/${HF_REPO}/resolve/main/${HF_FILE}"
        echo "  URL: ${DOWNLOAD_URL}"
        curl -#L --retry 3 --retry-delay 5 \
            -o "${MODEL_PATH}.tmp" \
            "${DOWNLOAD_URL}"
        mv "${MODEL_PATH}.tmp" "${MODEL_PATH}"
    fi

    # Verify the file was downloaded and looks like a GGUF
    if [ ! -f "${MODEL_PATH}" ]; then
        echo "ERROR: Model download failed — ${MODEL_PATH} not found."
        exit 1
    fi

    # Quick sanity: GGUF files start with "GGUF" (0x47 0x47 0x55 0x46)
    MAGIC=$(head -c 4 "${MODEL_PATH}" | od -A n -t x1 | tr -d ' \n')
    if [ "${MAGIC}" != "47475546" ]; then
        echo "WARNING: File does not have GGUF magic bytes (got '${MAGIC}')."
        echo "  The model may be corrupted or the wrong file."
    fi

    echo "======================================================================"
    echo " Model downloaded successfully."
    echo " Size: $(ls -lh "${MODEL_PATH}" | awk '{print $5}')"
    echo "======================================================================"
else
    echo "Model found in cache: ${MODEL_PATH}"
fi

# ---- Start llama.cpp server ------------------------------------------------
echo "======================================================================"
echo " Starting llama.cpp server (OpenAI-compatible API)..."
echo "   Model   : ${MODEL_PATH}"
echo "   Host    : ${HOST}:${PORT}"
echo "   Context : ${CTX_SIZE} tokens"
echo "   Threads : ${THREADS}"
echo "======================================================================"

exec llama-server \
    --model "${MODEL_PATH}" \
    --host "${HOST}" \
    --port "${PORT}" \
    --ctx-size "${CTX_SIZE}" \
    --threads "${THREADS}" \
    --n-gpu-layers "${N_GPU_LAYERS}" \
    --temp "${TEMPERATURE}" \
    --repeat-penalty 1.0 \
    --parallel 2 \
    --cont-batching
