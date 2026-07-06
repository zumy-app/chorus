import os
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import ctranslate2
import sentencepiece as spm

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="NLLB Translation Service")

model_dir = os.environ.get("MODEL_DIR", "/app/models/opus_nlp_models/nllb-200-distilled-600M-ct2-int8")
model = None
sp_source = None
sp_target = None

class TranslateRequest(BaseModel):
    text: str
    source: str = "eng_Latn"
    target: str = "spa_Latn"

class TranslateResponse(BaseModel):
    translatedText: str

@app.on_event("startup")
async def load_model():
    global model, sp_source, sp_target
    logger.info(f"Loading model from {model_dir}")
    model = ctranslate2.Translator(model_dir, device="cpu", intra_threads=4)
    sp_source = spm.SentencePieceProcessor()
    sp_source.Load(os.path.join(model_dir, "source.spm"))
    sp_target = spm.SentencePieceProcessor()
    sp_target.Load(os.path.join(model_dir, "target.spm"))
    logger.info("Model loaded successfully")

@app.get("/health")
async def health():
    return {"status": "healthy", "model_loaded": model is not None}

@app.post("/translate", response_model=TranslateResponse)
async def translate(req: TranslateRequest):
    if model is None or sp_source is None:
        raise HTTPException(status_code=503, detail="Model not loaded yet")

    try:
        # Prepend source language token
        source_text = f"__{req.source}__ {req.text}"
        target_prefix = f"__{req.target}__"

        # Tokenize with source tokenizer
        source_tokens = sp_source.EncodeAsPieces(source_text)
        source_batch = [source_tokens]

        # Translate
        results = model.translate_batch(
            source_batch,
            target_prefix=[target_prefix],
            max_batch_size=1,
            replace_unknowns=True,
            max_decoding_length=512
        )

        # Detokenize with target tokenizer
        translated_pieces = results[0].hypotheses[0]
        translated = sp_target.DecodePieces(translated_pieces)

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