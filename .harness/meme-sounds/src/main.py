from fastapi import FastAPI, UploadFile, File, HTTPException
from pydantic import BaseModel
import subprocess
import os
import uuid
from typing import List

app = FastAPI(title="Meme Sounds API")

UPLOAD_DIR = "/tmp/meme_sounds"
os.makedirs(UPLOAD_DIR, exist_ok=True)


class SoundMetadata(BaseModel):
    id: str
    filename: str
    tags: List[str]
    duration: float
    is_safe: bool


database = []


@app.post("/ingest", response_model=SoundMetadata)
async def ingest_sound(file: UploadFile = File(...)):
    # 1. Save uploaded file
    file_id = str(uuid.uuid4())
    raw_path = os.path.join(UPLOAD_DIR, f"{file_id}_raw.mp3")
    processed_path = os.path.join(UPLOAD_DIR, f"{file_id}_processed.mp3")

    with open(raw_path, "wb") as f:
        f.write(await file.read())

    # 2. Process with ffmpeg to 1.5s max, normalize
    cmd = [
        "ffmpeg",
        "-y",
        "-i",
        raw_path,
        "-t",
        "1.5",
        "-af",
        "loudnorm=I=-16:TP=-1.5:LRA=11",
        processed_path,
    ]
    try:
        subprocess.run(cmd, check=True, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    except subprocess.CalledProcessError:
        raise HTTPException(status_code=400, detail="Audio processing failed")

    # 3. AI Intelligence Layer (Stub)
    # In a real app: use Whisper for transcription, LLM for tagging/moderation
    transcription = "test sound"
    tags = ["funny", "test", "meme"]
    is_safe = True

    if "nsfw" in transcription.lower():
        is_safe = False

    # 4. Save metadata
    metadata = SoundMetadata(
        id=file_id, filename=processed_path, tags=tags, duration=1.5, is_safe=is_safe
    )
    if is_safe:
        database.append(metadata)

    return metadata


@app.get("/search", response_model=List[SoundMetadata])
async def search_sounds(query: str):
    results = [
        s for s in database if any(query.lower() in tag.lower() for tag in s.tags)
    ]
    return results


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="0.0.0.0", port=8000)
