from fastapi import FastAPI, UploadFile, File, HTTPException, BackgroundTasks
from fastapi.responses import FileResponse
from pydantic import BaseModel
import subprocess
import os
import uuid
import sqlite3
import json
from typing import List
import threading
import time

app = FastAPI(title="Meme Sounds API")

UPLOAD_DIR = "/tmp/meme_sounds"
os.makedirs(UPLOAD_DIR, exist_ok=True)

# Initialize SQLite DB
DB_PATH = "sounds.db"
def init_db():
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute('''CREATE TABLE IF NOT EXISTS sounds
                 (id TEXT PRIMARY KEY, filename TEXT, tags TEXT, duration REAL, is_safe BOOLEAN)''')
    conn.commit()
    conn.close()

init_db()

class SoundMetadata(BaseModel):
    id: str
    filename: str
    tags: List[str]
    duration: float
    is_safe: bool

def get_audio_duration(filepath: str) -> float:
    cmd = ["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filepath]
    try:
        result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        return float(result.stdout.strip())
    except Exception:
        return 0.0

def process_and_save_sound(raw_path: str, file_id: str):
    processed_path = os.path.join(UPLOAD_DIR, f"{file_id}_processed.mp3")
    cmd = ["ffmpeg", "-y", "-i", raw_path, "-t", "1.5", "-af", "loudnorm=I=-16:TP=-1.5:LRA=11", processed_path]
    subprocess.run(cmd, check=True, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    
    duration = get_audio_duration(processed_path)
    
    # Mock AI Layer for Whisper and LLM
    transcription = "simulated transcription of the meme"
    tags = ["meme", "funny", "auto-generated", "semantic"]
    is_safe = True
    
    # Semantic Search mock / Vector DB mock
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute("INSERT INTO sounds VALUES (?, ?, ?, ?, ?)", 
              (file_id, processed_path, json.dumps(tags), duration, is_safe))
    conn.commit()
    conn.close()
    
    return SoundMetadata(id=file_id, filename=processed_path, tags=tags, duration=duration, is_safe=is_safe)

@app.post("/ingest", response_model=SoundMetadata)
async def ingest_sound(file: UploadFile = File(...)):
    file_id = str(uuid.uuid4())
    raw_path = os.path.join(UPLOAD_DIR, f"{file_id}_raw.mp3")
    with open(raw_path, "wb") as f:
        f.write(await file.read())
    try:
        return process_and_save_sound(raw_path, file_id)
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))

@app.get("/search", response_model=List[SoundMetadata])
async def search_sounds(query: str):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    # Mock semantic search using SQL LIKE
    c.execute("SELECT * FROM sounds WHERE tags LIKE ?", (f"%{query}%",))
    rows = c.fetchall()
    conn.close()
    results = []
    for r in rows:
        results.append(SoundMetadata(id=r[0], filename=r[1], tags=json.loads(r[2]), duration=r[3], is_safe=bool(r[4])))
    return results

@app.get("/play/{sound_id}")
async def play_sound(sound_id: str):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute("SELECT filename FROM sounds WHERE id=?", (sound_id,))
    row = c.fetchone()
    conn.close()
    if row and os.path.exists(row[0]):
        return FileResponse(row[0], media_type="audio/mpeg")
    raise HTTPException(status_code=404, detail="Sound not found")

# Scraper background task
def scraper_task():
    while True:
        # Mock scraping process
        time.sleep(3600)

threading.Thread(target=scraper_task, daemon=True).start()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
