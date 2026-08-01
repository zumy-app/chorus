#!/usr/bin/env python3
"""NLLB-200 CTranslate2 translation sidecar.

High-performance, CPU-optimized all-to-all translation service for Meta's
NLLB-200 model. Uses the CTranslate2 runtime with 8-bit integer (int8)
quantization for fast inference on standard CPU cores.

Endpoints (JSON):
    GET  /health    -> {"status": ..., "model_loaded": ...}
    POST /translate -> {"text": "...", "source": "en", "target": "es"}
                       -> {"translatedText": "..."}

The service accepts common 2-letter ISO codes (e.g. "en", "es", "zh") which
are mapped internally to the FLORES-200 BCP-47 codes the NLLB model requires
(e.g. "eng_Latn", "spa_Latn", "zho_Hans"). Fully-qualified FLORES codes are
also accepted and passed through unchanged.

Model lifecycle:
    On boot the script checks for an already-converted int8 CTranslate2 model
    on disk (NLLB_MODEL_DIR / nllb_200_distilled_int8). If it is missing it
    downloads `facebook/nllb-200-distilled-600M` and converts it with
    `TransformersConverter(..., quantization="int8")`. The converted model is
    cached in a persistent Docker volume so restarts are instantaneous.

    A pre-converted int8 checkpoint can be downloaded from the Hugging Face
    Hub instead by setting NLLB_CT2_REPO (falls back to full conversion).

Formatting robustness:
    Multilingual seq2seq models occasionally hallucinate punctuation and
    capitalization (lower-cased sentence starts, lost title case, stray
    BCP-47 language tokens, doubled whitespace). A strict pre/post-processing
    pipeline stabilizes these artifacts without touching the model.

CLI:
    python server.py                 # run the API (uvicorn, port 5001)
    python server.py --convert       # convert + exit (used at image build time)
"""

import argparse
import gc
import logging
import os
import re
import sys
import time
from contextlib import asynccontextmanager
from pathlib import Path

logger = logging.getLogger("nllb")
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s [%(name)s] %(message)s",
)

# =============================================================================
# Configuration (env overridable)
# =============================================================================
MODEL_DIR = os.environ.get("NLLB_MODEL_DIR", "/app/models/nllb_200_distilled_int8")
SOURCE_MODEL = os.environ.get("NLLB_SOURCE_MODEL", "facebook/nllb-200-distilled-600M")
PRECONVERTED_REPO = os.environ.get("NLLB_CT2_REPO", "")
DEFAULT_SOURCE = os.environ.get("NLLB_DEFAULT_SOURCE", "eng_Latn")
DEFAULT_TARGET = os.environ.get("NLLB_DEFAULT_TARGET", "spa_Latn")

INTER_THREADS = int(os.environ.get("NLLB_INTER_THREADS", "2"))
INTRA_THREADS = int(os.environ.get("NLLB_INTRA_THREADS", "2"))
COMPUTE_TYPE = os.environ.get("NLLB_COMPUTE_TYPE", "int8")
BEAM_SIZE = int(os.environ.get("NLLB_BEAM_SIZE", "1"))
MAX_LENGTH = int(os.environ.get("NLLB_MAX_LENGTH", "512"))
MAX_INPUT_LENGTH = int(os.environ.get("NLLB_MAX_INPUT_LENGTH", "512"))
LENGTH_PENALTY = float(os.environ.get("NLLB_LENGTH_PENALTY", "0.6"))

PORT = int(os.environ.get("PORT", "5001"))

# =============================================================================
# 2-letter ISO -> FLORES-200 BCP-47 language codes
# =============================================================================
LANG_MAP = {
    "af": "afr_Latn", "am": "amh_Ethi", "ar": "arb_Arab", "az": "azj_Latn",
    "be": "bel_Cyrl", "bg": "bul_Cyrl", "bn": "ben_Beng", "bs": "bos_Latn",
    "ca": "cat_Latn", "ceb": "ceb_Latn", "cs": "ces_Latn", "cy": "cym_Latn",
    "da": "dan_Latn", "de": "deu_Latn", "el": "ell_Grek", "en": "eng_Latn",
    "eo": "epo_Latn", "es": "spa_Latn", "et": "est_Latn", "fa": "pes_Arab",
    "fi": "fin_Latn", "fil": "fil_Latn", "fr": "fra_Latn", "ga": "gle_Latn",
    "gl": "glg_Latn", "gu": "guj_Gujr", "ha": "hau_Latn", "he": "heb_Hebr",
    "hi": "hin_Deva", "hr": "hrv_Latn", "ht": "hat_Latn", "hu": "hun_Latn",
    "hy": "hye_Armn", "id": "ind_Latn", "ig": "ibo_Latn", "is": "isl_Latn",
    "it": "ita_Latn", "ja": "jpn_Jpan", "jv": "jav_Latn", "ka": "kat_Geor",
    "kk": "kaz_Cyrl", "km": "khm_Khmr", "kn": "kan_Knda", "ko": "kor_Kore",
    "ky": "kir_Cyrl", "lo": "lao_Laoo", "lt": "lit_Latn", "lv": "lvs_Latn",
    "mg": "plt_Latn", "mk": "mkd_Cyrl", "ml": "mal_Mlym", "mn": "khk_Cyrl",
    "mr": "mar_Deva", "ms": "zsm_Latn", "mt": "mlt_Latn", "my": "mya_Mymr",
    "ne": "nep_Deva", "nl": "nld_Latn", "no": "nob_Latn", "pa": "pan_Guru",
    "pl": "pol_Latn", "ps": "pbt_Arab", "pt": "por_Latn", "ro": "ron_Latn",
    "ru": "rus_Cyrl", "rw": "kin_Latn", "si": "sin_Sinh", "sk": "slk_Latn",
    "sl": "slv_Latn", "so": "som_Latn", "sq": "als_Latn", "sr": "srp_Cyrl",
    "sv": "swe_Latn", "sw": "swh_Latn", "ta": "tam_Taml", "te": "tel_Telu",
    "tg": "tgk_Cyrl", "th": "tha_Thai", "tk": "tuk_Latn", "tr": "tur_Latn",
    "uk": "ukr_Cyrl", "ur": "urd_Arab", "uz": "uzn_Latn", "vi": "vie_Latn",
    "xh": "xho_Latn", "yo": "yor_Latn", "zh": "zho_Hans", "zu": "zul_Latn",
}

# Extra aliases beyond the strict 2-letter codes.
LANG_ALIASES = {
    "auto": None,            # resolved later via _detect_lang / DEFAULT_SOURCE
    "zh-hans": "zho_Hans",
    "zh-hant": "zho_Hant",
    "zh-cn": "zho_Hans",
    "zh-tw": "zho_Hant",
    "pt-br": "por_Latn",
    "pt-pt": "por_Latn",
    "no-nb": "nob_Latn",
    "no-nn": "nob_Latn",
    "tl": "fil_Latn",
    "in": "ind_Latn",
    "iw": "heb_Hebr",
    "jw": "jav_Latn",
    "mo": "ron_Latn",
    "nb": "nob_Latn",
    "nn": "nob_Latn",
    "sh": "srp_Latn",
}

# =============================================================================
# Defensive text stabilization (pre / post processing)
# =============================================================================
_BCP47_TOKEN_RE = re.compile(r"__[A-Za-z0-9]+(?:_[A-Za-z0-9]+)*__")
_PLAIN_LANG_LEAD_RE = re.compile(r"^([A-Za-z]{2,3})_[A-Za-z]{3,4}(\s|$)")
_MULTI_SPACE_RE = re.compile(r"[ \t]+")
_MULTI_NEWLINE_RE = re.compile(r"\n{3,}")


def _has_letters(s: str) -> bool:
    return any(ch.isalpha() for ch in s)


def capture_format(text: str) -> dict:
    """Snapshot the casing/whitespace shape of the input for later re-application."""
    words = [w for w in re.split(r"\s+", text) if w]
    profile = {
        "leading": len(text) - len(text.lstrip(" \t")),
        "first_cap": False,
        "screaming": False,
        "title": False,
    }
    if not words:
        return profile

    first = words[0]
    profile["first_cap"] = bool(first and first[0].isalpha() and first[0].isupper())

    alpha_words = [w for w in words if _has_letters(w)]
    n_alpha = len(alpha_words)
    if n_alpha == 0:
        return profile

    screaming = all(w.isupper() for w in alpha_words)
    capped = sum(1 for w in alpha_words if w[0].isupper())
    title = (not screaming) and n_alpha >= 3 and capped >= max(2, int(n_alpha * 0.6))
    profile["screaming"] = screaming
    profile["title"] = title
    return profile


def preprocess(text: str):
    """Normalize irregular whitespace/newlines and strip stray BCP-47 tokens."""
    if text is None:
        return "", {}
    stripped = text.strip()
    if not stripped or len(stripped) <= 1:
        # Too short to be meaningfully translated; pass through untouched.
        return text, {}

    profile = capture_format(text)
    cleaned = text.replace("\r\n", "\n").replace("\r", "\n")
    cleaned = _MULTI_SPACE_RE.sub(" ", cleaned)
    cleaned = _BCP47_TOKEN_RE.sub("", cleaned)
    cleaned = _MULTI_NEWLINE_RE.sub("\n\n", cleaned)
    cleaned = cleaned.strip()
    return cleaned, profile


def postprocess(raw: str, profile: dict) -> str:
    """Strip language prefixes, normalize spacing, and re-apply captured casing."""
    if not raw:
        return ""
    text = _BCP47_TOKEN_RE.sub("", raw)          # __spa_Latn__ style tags
    text = re.sub(r"[ \t]+", " ", text).strip()

    # Drop a leading raw BCP-47 tag (e.g. "spa_Latn El gato...").
    m = _PLAIN_LANG_LEAD_RE.match(text)
    if m and not text.startswith(("_", "http")):
        text = text[m.end():].lstrip()

    if not text:
        return text

    if profile.get("screaming"):
        text = text.upper()
        return " " * profile.get("leading", 0) + text

    lines = text.split("\n")
    for i, line in enumerate(lines):
        if not line.strip():
            continue
        m = re.match(r"^(\s*)([^\w]*)(\w+)(.*)$", line, re.S)
        if m:
            indent, punct, word, rest = m.groups()
            if profile.get("title"):
                word = word.capitalize()
                rest = re.sub(r"\s+(\w)", lambda mm: mm.group(0)[:-1] + mm.group(1).upper(), rest)
            elif profile.get("first_cap") and word and word[0].islower():
                word = word[0].upper() + word[1:]
            elif (not profile.get("first_cap") and word.isalpha()
                  and len(word) > 1 and word[0].isupper()):
                word = word[0].lower() + word[1:]
            lines[i] = indent + punct + word + rest
        break

    return " " * profile.get("leading", 0) + "\n".join(lines)


def _detect_lang(text: str) -> str:
    """Cheap script-based language detection for 'auto' source requests."""
    if re.search(r"[\u4e00-\u9fff]", text):
        return "zho_Hans"
    if re.search(r"[\u3040-\u30ff]", text):
        return "jpn_Jpan"
    if re.search(r"[\uac00-\ud7af]", text):
        return "kor_Kore"
    if re.search(r"[\u0e00-\u0e7f]", text):
        return "tha_Thai"
    if re.search(r"[\u0900-\u097f]", text):
        return "hin_Deva"
    if re.search(r"[\u0600-\u06ff]", text):
        return "arb_Arab"
    if re.search(r"[\u0400-\u04ff]", text):
        return "rus_Cyrl"
    if re.search(r"[\u05d0-\u05ea]", text):
        return "heb_Hebr"
    return DEFAULT_SOURCE


def _flores(code: str, default: str) -> str:
    """Resolve a 2-letter/alias/FLORES code to a FLORES-200 code."""
    if not code:
        return default
    c = code.strip()
    low = c.lower()

    if low in ("auto", ""):
        return default

    if low in LANG_ALIASES:
        val = LANG_ALIASES[low]
        return val if val else default

    if low in LANG_MAP:
        return LANG_MAP[low]

    # Already a fully-qualified FLORES-200 code? Pass it through.
    if re.fullmatch(r"[a-z]{2,3}_[A-Za-z]{3,4}", c):
        return c

    logger.warning("Unknown language code %r, defaulting to %s", code, default)
    return default


# =============================================================================
# Model management
# =============================================================================
def model_ready() -> bool:
    return (Path(MODEL_DIR) / "model.bin").exists()


def convert_model() -> None:
    """Download the HF NLLB checkpoint and convert it to CT2 int8 (once)."""
    from ctranslate2.converters import TransformersConverter

    out = Path(MODEL_DIR)
    out.mkdir(parents=True, exist_ok=True)
    logger.info("Converting %s -> %s (quantization=int8, device=cpu)", SOURCE_MODEL, out)
    started = time.time()
    try:
        converter = TransformersConverter(
            SOURCE_MODEL,
            copy_files=[
                "sentencepiece.bpe.model",
                "tokenizer_config.json",
                "special_tokens_map.json",
            ],
            load_as_float16=True,   # halves peak RAM during conversion
            low_cpu_mem_usage=True,
        )
        converter.convert(str(out), quantization="int8", force=True)
    except Exception:
        logger.exception("Model conversion failed")
        raise
    logger.info("Conversion complete in %.1fs", time.time() - started)
    gc.collect()


def ensure_model() -> None:
    """Make sure an int8 CT2 model exists on disk, converting if needed."""
    if model_ready():
        logger.info("Cached int8 model found at %s", MODEL_DIR)
        return

    # Optional fast path: fetch a pre-converted int8 checkpoint from the Hub.
    if PRECONVERTED_REPO:
        try:
            from huggingface_hub import snapshot_download

            logger.info("Downloading pre-converted model %s", PRECONVERTED_REPO)
            snapshot_download(repo_id=PRECONVERTED_REPO, local_dir=MODEL_DIR)
            if model_ready():
                logger.info("Pre-converted model downloaded to %s", MODEL_DIR)
                return
            logger.warning("Pre-converted download produced no model.bin; converting instead")
        except Exception as exc:  # noqa: BLE001
            logger.warning("Pre-converted download failed (%s); converting instead", exc)

    convert_model()


def load_model() -> None:
    """Load the SentencePiece tokenizer + CT2 int8 translator (CPU)."""
    global tokenizer, translator

    ensure_model()

    from transformers import AutoTokenizer

    try:
        tokenizer = AutoTokenizer.from_pretrained(
            MODEL_DIR, src_lang="eng_Latn", use_fast=False, local_files_only=True
        )
    except Exception:
        logger.warning("Local tokenizer load failed, falling back to %s", SOURCE_MODEL)
        tokenizer = AutoTokenizer.from_pretrained(
            SOURCE_MODEL, src_lang="eng_Latn", use_fast=False
        )

    import ctranslate2

    translator = ctranslate2.Translator(
        MODEL_DIR,
        device="cpu",
        compute_type=COMPUTE_TYPE,
        intra_threads=INTRA_THREADS,
        inter_threads=INTER_THREADS,
    )
    logger.info(
        "Translator ready: device=cpu compute_type=%s intra=%d inter=%d",
        COMPUTE_TYPE, INTRA_THREADS, INTER_THREADS,
    )


def _translate(text: str, src_flores: str, tgt_flores: str) -> str:
    """Core CTranslate2 NLLB translation for a single string."""
    src_id = tokenizer.lang_code_to_id.get(src_flores)
    if src_id is None:
        src_id = tokenizer.lang_code_to_id[DEFAULT_SOURCE]
    source_ids = tokenizer(text, add_special_tokens=False)["input_ids"]
    source_tokens = tokenizer.convert_ids_to_tokens([src_id]) + \
        tokenizer.convert_ids_to_tokens(source_ids)

    results = translator.translate_batch(
        [source_tokens],
        target_prefix=[[tgt_flores]],
        beam_size=BEAM_SIZE,
        max_batch_size=1,
        length_penalty=LENGTH_PENALTY,
        max_length=MAX_LENGTH,
        max_input_length=MAX_INPUT_LENGTH,
    )
    hypothesis = results[0].hypotheses[0]
    # Drop the target language prefix token emitted at the start of the output.
    if hypothesis and hypothesis[0] in (tgt_flores, "__%s__" % tgt_flores):
        hypothesis = hypothesis[1:]
    target_ids = tokenizer.convert_tokens_to_ids(hypothesis)
    return tokenizer.decode(target_ids, skip_special_tokens=True)


def translate_text(text: str, source: str, target: str) -> str:
    """Pre/translate/post pipeline entrypoint."""
    if not text:
        return ""

    stripped = text.strip()
    if len(stripped) <= 1:
        return text  # empty / single-character guard

    cleaned, profile = preprocess(text)
    if not cleaned:
        return text

    src = _flores(source, DEFAULT_SOURCE)
    if src == "auto":
        src = _detect_lang(cleaned)
    tgt = _flores(target, DEFAULT_TARGET)

    if src == tgt:
        return text  # no-op translation

    raw = _translate(cleaned, src, tgt)
    return postprocess(raw, profile)


# =============================================================================
# FastAPI application
# =============================================================================
tokenizer = None
translator = None


@asynccontextmanager
async def lifespan(_app):
    started = time.time()
    logger.info("Loading NLLB-200 model (first boot converts to int8, may take a while)...")
    load_model()
    logger.info("Model ready in %.1fs", time.time() - started)
    yield


from fastapi import FastAPI, HTTPException  # noqa: E402
from pydantic import BaseModel, Field  # noqa: E402


class TranslateRequest(BaseModel):
    text: str = Field(..., description="Text to translate")
    source: str = Field("en", description="Source language code (ISO 2-letter or FLORES)")
    target: str = Field("es", description="Target language code (ISO 2-letter or FLORES)")


class TranslateResponse(BaseModel):
    translatedText: str
    source: str
    target: str
    provider: str = "nllb-local"


class HealthResponse(BaseModel):
    status: str
    model_loaded: bool
    device: str = "cpu"
    quantization: str = COMPUTE_TYPE
    inter_threads: int = INTER_THREADS
    model_dir: str = MODEL_DIR
    languages: int = len(LANG_MAP)


app = FastAPI(title="NLLB-200 CTranslate2 Translation Sidecar", version="2.0.0", lifespan=lifespan)


@app.get("/", response_model=dict)
def root():
    return {
        "service": "nllb-local",
        "model": SOURCE_MODEL,
        "backend": "ctranslate2",
        "quantization": COMPUTE_TYPE,
        "device": "cpu",
        "languages": len(LANG_MAP),
    }


@app.get("/health", response_model=HealthResponse)
def health():
    return HealthResponse(
        status="ok" if translator is not None else "loading",
        model_loaded=translator is not None,
    )


@app.post("/translate", response_model=TranslateResponse)
def translate(req: TranslateRequest):
    if translator is None or tokenizer is None:
        raise HTTPException(status_code=503, detail="Model is still loading")
    try:
        out = translate_text(req.text, req.source, req.target)
        src = _flores(req.source, DEFAULT_SOURCE)
        tgt = _flores(req.target, DEFAULT_TARGET)
        return TranslateResponse(translatedText=out, source=src, target=tgt)
    except HTTPException:
        raise
    except Exception as exc:  # noqa: BLE001
        logger.exception("Translation failed")
        raise HTTPException(status_code=500, detail=str(exc))


# =============================================================================
# Entry point
# =============================================================================
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="NLLB-200 CTranslate2 sidecar")
    parser.add_argument(
        "--convert",
        action="store_true",
        help="Download + convert the model to int8 and exit (used at build time)",
    )
    parser.add_argument(
        "--host", default="0.0.0.0", help="Bind host (default 0.0.0.0)"
    )
    args = parser.parse_args()

    if args.convert:
        ensure_model()
        logger.info("Model ready at %s", MODEL_DIR)
        sys.exit(0)

    import uvicorn

    uvicorn.run(app, host=args.host, port=PORT, workers=1)
