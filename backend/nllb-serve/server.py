import os
import logging
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from transformers import AutoModelForSeq2SeqLM, AutoTokenizer

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="NLLB Translation Service")

model_name = os.environ.get("NLLB_MODEL", "facebook/nllb-200-distilled-600M")
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
    logger.info(f"Loading model {model_name} (this may take a moment on first run)...")
    tokenizer = AutoTokenizer.from_pretrained(model_name, src_lang="eng_Latn")
    model = AutoModelForSeq2SeqLM.from_pretrained(model_name)
    model.eval()
    logger.info("Model loaded successfully")

@app.get("/health")
async def health():
    return {"status": "healthy", "model_loaded": model is not None}

@app.post("/translate", response_model=TranslateResponse)
async def translate(req: TranslateRequest):
    if model is None or tokenizer is None:
        raise HTTPException(status_code=503, detail="Model still loading")

    try:
        tokenizer.src_lang = req.source

        inputs = tokenizer(
            req.text,
            return_tensors="pt",
            truncation=True,
            max_length=512
        )

        translated_tokens = model.generate(
            **inputs,
            forced_bos_token_id=tokenizer.convert_tokens_to_ids(f"__{req.target}__"),
            max_length=512,
            num_beams=1
        )

        translated = tokenizer.batch_decode(translated_tokens, skip_special_tokens=True)[0]
        return TranslateResponse(translatedText=translated)
    except Exception as e:
        logger.error(f"Translation error: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/")
async def root():
    return {"service": "nllb-translation", "model": model_name}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=5000)