# Translator Engine

High-performance translation container using `llama.cpp` server with a 7B-parameter translation model (ALMA-7B) in GGUF format.

## Architecture

- **Isolation**: The LLM runtime runs in its own C++ Docker container to prevent large memory maps from disrupting the Go backend's runtime and garbage collector.
- **RAM**: Allocates ~4.2–4.5 GB (fits a Q3_K_M 7B model comfortably on a 6 GB total system).
- **API**: Exposes an OpenAI-compatible API (`/v1/chat/completions`) on port 5000.
- **Hardware Acceleration**: Utilizes CPU AVX2/AVX512 instruction sets.

## Model

Default model: **ALMA-7B** (Q3_K_M quantization, ~3.3 GB) from [TheBloke/ALMA-7B-GGUF](https://huggingface.co/TheBloke/ALMA-7B-GGUF).

Override via environment variables:
- `HF_REPO` — Hugging Face repo (default: `TheBloke/ALMA-7B-GGUF`)
- `HF_FILE` — GGUF filename (default: `alma-7b.Q3_K_M.gguf`)

### Alternative Models

| Model | HF Repo | Quant | Size | Notes |
|-------|---------|-------|------|-------|
| ALMA-7B | TheBloke/ALMA-7B-GGUF | Q3_K_M | 3.3 GB | Default; good translation quality |
| ALMA-7B | TheBloke/ALMA-7B-GGUF | Q4_K_M | 4.08 GB | Higher quality, more RAM |
| Synatra-7B-Translation | andreass123/Synatra-7B-v0.3-Translation-Q4_K_M-GGUF | Q4_K_M | 4.37 GB | Dedicated translation model |
| Madlad-400-7B | *(check HF for GGUF variants)* | — | — | Google's multilingual translation model |

## Server Flags

Configured via environment variables:
- `CTX_SIZE` — Context size in tokens (default: 2048)
- `THREADS` — CPU threads (default: 4)
- `TEMPERATURE` — Generation temperature (default: 0.1)
- `PORT` — Server port (default: 5000)
- `HOST` — Bind address (default: 0.0.0.0)

## Building

```bash
docker build -t chorus-translator-engine ./backend/translator-engine
```

## API Example

```bash
curl -X POST http://localhost:5000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "default",
    "messages": [
      {"role": "system", "content": "You are a professional translator. Translate accurately."},
      {"role": "user", "content": "Translate from English to Spanish: Hello world"}
    ],
    "temperature": 0.1
  }'
```

Response:
```json
{
  "choices": [{"message": {"content": "Hola mundo"}}]
}
```
