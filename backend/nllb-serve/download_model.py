import huggingface_hub
from pathlib import Path

model_name = "nllb-200-distilled-600M-ct2-int8"
repo_id = f"JustFrederik/{model_name}"
local_dir = Path(f"/app/models/{model_name}")

if not local_dir.exists():
    huggingface_hub.snapshot_download(repo_id=repo_id, local_dir=str(local_dir))

print("Model downloaded successfully")