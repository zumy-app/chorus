import os
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import ctranslate2
import sentencepiece as spm

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="NLLB Translation Service")

model_dir = os.environ.get("MODEL_DIR", "/app/models/nllb-200-distilled-600M-ct2-int8")
model = None
sp = None

class TranslateRequest(BaseModel):
    text: str
    source: str = "eng_Latn"
    target: str = "spa_Latn"

class TranslateResponse(BaseModel):
    translatedText: str

@app.on_event("startup")
async def load_model():
    global model, sp
    logger.info(f"Loading model from {model_dir}")

    bpe_model_path = os.path.join(model_dir, "sentencepiece.bpe.model")
    if not os.path.exists(bpe_model_path):
        # Try fallback filename
        files = os.listdir(model_dir)
        logger.info(f"Files in model dir: {files}")
        spm_files = [f for f in files if f.endswith('.model') or f.endswith('.spm')]
        if spm_files:
            bpe_model_path = os.path.join(model_dir, spm_files[0])

    logger.info(f"Using tokenizer model: {bpe_model_path}")
    sp = spm.SentencePieceProcessor()
    sp.Load(bpe_model_path)

    model = ctranslate2.Translator(model_dir, device="cpu", intra_threads=4)
    logger.info("Model loaded successfully")

@app.get("/health")
async def health():
    return {"status": "healthy", "model_loaded": model is not None}

@app.post("/translate", response_model=TranslateResponse)
async def translate(req: TranslateRequest):
    if model is None or sp is None:
        raise HTTPException(status_code=503, detail="Model not loaded yet")

    try:
        # NLLB uses closed vocabulary with language tags
        source_text = f"__{req.source}__ {req.text}"
        target_prefix = [f"__{req.target}__"]

        # Tokenize
        source_tokens = sp.EncodeAsPieces(source_text)
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
        translated_pieces = results[0].hypotheses[0]
        translated = sp.DecodePieces(translated_pieces)

        # Strip target language token prefix if present
        if translated.startswith("__"):
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