import os
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import ctranslate2

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="NLLB Translation Service")

model_dir = os.environ.get("MODEL_DIR", "/app/models/nllb-200-distilled-600M-ct2-int8")
model = None
tokenizer = None

class TranslateRequest(BaseModel):
    text: str
    source: str = "eng_Latn"
    target: str = "spa_Latn"

class TranslateResponse(BaseModel):
    translatedText: str

@app.on_event("startup")
async def load_model():
    global model, tokenizer
    logger.info(f"Loading model from {model_dir}")
    model = ctranslate2.Translator(model_dir, device="cpu", intra_threads=4)
    tokenizer = ctranslate2.sentencepiece.BPEProcessor(model_dir)
    logger.info("Model loaded successfully")

@app.get("/health")
async def health():
    return {"status": "healthy", "model_loaded": model is not None}

@app.post("/translate", response_model=TranslateResponse)
async def translate(req: TranslateRequest):
    if model is None or tokenizer is None:
        raise HTTPException(status_code=503, detail="Model not loaded yet")

    try:
        # NLLB uses language tokens like __eng_Latn__ and __spa_Latn__
        source_with_lang = f"__{req.source}__ {req.text}"
        target_prefix = f"__{req.target}__"

        # Tokenize
        source_tokens = tokenizer.encode(source_with_lang)
        source_batch = [source_tokens]

        # Translate
        results = model.translate_batch(
            source_batch,
            target_prefix=[target_prefix],
            max_batch_size=1,
            replace_unknowns=True,
            max_decoding_length=512
        )

        # Detokenize
        translated_tokens = results[0].hypotheses[0]
        translated = tokenizer.decode(translated_tokens)

        # Remove the target language token prefix if it's still in the output
        if translated.startswith(f"__"):
            parts = translated.split("__", 2)
            if len(parts) >= 3:
                translated = parts[2].strip()

        return TranslateResponse(translatedText=translated)
    except Exception as e:
        logger.error(f"Translation error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/")
async def root():
    return {"service": "nllb-translation", "model": "nllb-200-distilled-600M"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5000)