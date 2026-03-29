from fastapi import FastAPI, UploadFile, File, HTTPException, Form, Request, BackgroundTasks
from fastapi.responses import FileResponse, HTMLResponse
from pydantic import BaseModel
import subprocess
import os
import uuid
import sqlite3
import json
from typing import List
import threading
import time
import requests
from bs4 import BeautifulSoup
from openai import OpenAI
import tempfile

app = FastAPI(title="Meme Sounds API")

UPLOAD_DIR = os.getenv("UPLOAD_DIR", "/tmp/meme_sounds")
os.makedirs(UPLOAD_DIR, exist_ok=True)

DB_PATH = os.path.join(UPLOAD_DIR, "sounds.db")

client = OpenAI(api_key=os.getenv("OPENAI_API_KEY"))

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

def validate_audio_file(filepath: str) -> bool:
    cmd = ["ffprobe", "-v", "error", "-show_entries", "format=format_name", "-of", "default=noprint_wrappers=1:nokey=1", filepath]
    try:
        result = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        return bool(result.stdout.strip())
    except Exception:
        return False

def analyze_with_ai(filepath: str):
    tags = []
    is_safe = True
    transcription = ""

    if not os.getenv("OPENAI_API_KEY"):
        return ["no-ai", "stub"], True

    try:
        # Whisper Transcription
        with open(filepath, "rb") as audio_file:
            transcript_response = client.audio.transcriptions.create(
                model="whisper-1", 
                file=audio_file
            )
        transcription = transcript_response.text

        # LLM Moderation & Tagging
        system_prompt = "You are an AI that tags meme sounds based on transcription and context. Return JSON with 'tags' (array of strings) and 'is_safe' (boolean)."
        user_prompt = f"Transcription: '{transcription}'. Analyze this short audio clip. If it contains offensive, NSFW, or highly inappropriate content, set is_safe to false. Generate 3-5 descriptive, punchy tags."

        completion = client.chat.completions.create(
            model="gpt-4o",
            response_format={"type": "json_object"},
            messages=[
                {"role": "system", "content": system_prompt},
                {"role": "user", "content": user_prompt}
            ]
        )
        
        result = json.loads(completion.choices[0].message.content)
        tags = result.get("tags", ["meme"])
        is_safe = result.get("is_safe", True)
    except Exception as e:
        print(f"AI Analysis failed: {e}")
        tags = ["error", "unprocessed"]
        is_safe = False

    return tags, is_safe

def process_and_save_sound(raw_path: str, file_id: str):
    if not validate_audio_file(raw_path):
        os.remove(raw_path)
        raise ValueError("Invalid audio file format")

    processed_filename = f"{file_id}_processed.mp3"
    processed_path = os.path.join(UPLOAD_DIR, processed_filename)
    cmd = ["ffmpeg", "-y", "-i", raw_path, "-t", "1.5", "-af", "loudnorm=I=-16:TP=-1.5:LRA=11", processed_path]
    subprocess.run(cmd, check=True, stderr=subprocess.PIPE, stdout=subprocess.PIPE)
    
    os.remove(raw_path)
    
    duration = get_audio_duration(processed_path)
    
    # AI Integration
    tags, is_safe = analyze_with_ai(processed_path)
    
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute("INSERT INTO sounds VALUES (?, ?, ?, ?, ?)", 
              (file_id, processed_filename, json.dumps(tags), duration, is_safe))
    conn.commit()
    conn.close()
    
    return SoundMetadata(id=file_id, tags=tags, duration=duration, is_safe=is_safe)

@app.post("/ingest", response_model=SoundMetadata)
async def ingest_sound(file: UploadFile = File(...)):
    file_id = str(uuid.uuid4())
    raw_path = os.path.join(UPLOAD_DIR, f"{file_id}_raw.mp3")
    with open(raw_path, "wb") as f:
        f.write(await file.read())
    try:
        return process_and_save_sound(raw_path, file_id)
    except ValueError as e:
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        if os.path.exists(raw_path):
            os.remove(raw_path)
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/search", response_model=List[SoundMetadata])
async def search_sounds(query: str = ""):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    if query:
        c.execute("SELECT * FROM sounds WHERE tags LIKE ? AND is_safe = 1", (f"%{query}%",))
    else:
        c.execute("SELECT * FROM sounds WHERE is_safe = 1 LIMIT 50")
    rows = c.fetchall()
    conn.close()
    results = []
    for r in rows:
        results.append(SoundMetadata(id=r[0], tags=json.loads(r[2]), duration=r[3], is_safe=bool(r[4])))
    return results

@app.get("/play/{sound_id}")
async def play_sound(sound_id: str):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute("SELECT filename FROM sounds WHERE id=?", (sound_id,))
    row = c.fetchone()
    conn.close()
    if row:
        filepath = os.path.join(UPLOAD_DIR, row[0])
        if os.path.exists(filepath):
            return FileResponse(filepath, media_type="audio/mpeg")
    raise HTTPException(status_code=404, detail="Sound not found")

@app.get("/", response_class=HTMLResponse)
async def read_root():
    return """
    <!DOCTYPE html>
    <html>
    <head>
        <title>Meme Sounds Library</title>
        <style>
            body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
            .sound-card { border: 1px solid #ccc; padding: 15px; margin-bottom: 15px; border-radius: 8px; }
            .tags { color: #666; font-size: 0.9em; margin-bottom: 10px; }
            .tag { background: #eee; padding: 2px 6px; border-radius: 4px; margin-right: 5px; }
            button { padding: 8px 16px; cursor: pointer; background: #007bff; color: white; border: none; border-radius: 4px; }
            button:hover { background: #0056b3; }
            .upload-form { background: #f8f9fa; padding: 20px; margin-bottom: 30px; border-radius: 8px; border: 1px solid #dee2e6; }
            input[type="text"] { padding: 8px; width: 60%; margin-right: 10px; border: 1px solid #ccc; border-radius: 4px; }
        </style>
    </head>
    <body>
        <h1>Meme Sounds Library</h1>
        
        <div class="upload-form">
            <h3>Upload New Sound</h3>
            <form id="uploadForm">
                <input type="file" id="audioFile" accept="audio/*" required>
                <button type="submit">Upload & Process (Max 1.5s)</button>
            </form>
            <div id="uploadStatus" style="margin-top: 10px; color: green;"></div>
        </div>

        <div>
            <input type="text" id="searchInput" placeholder="Search sounds (e.g., 'funny', 'fail')...">
            <button onclick="searchSounds()">Search</button>
        </div>

        <div id="results" style="margin-top: 30px;"></div>

        <script>
            document.getElementById('uploadForm').onsubmit = async (e) => {
                e.preventDefault();
                const fileInput = document.getElementById('audioFile');
                const file = fileInput.files[0];
                const formData = new FormData();
                formData.append('file', file);
                
                const statusDiv = document.getElementById('uploadStatus');
                statusDiv.innerText = 'Uploading and processing...';
                statusDiv.style.color = 'blue';
                
                try {
                    const res = await fetch('/ingest', { method: 'POST', body: formData });
                    if (res.ok) {
                        statusDiv.innerText = 'Uploaded successfully!';
                        statusDiv.style.color = 'green';
                        fileInput.value = '';
                        searchSounds();
                    } else {
                        const errText = await res.text();
                        statusDiv.innerText = 'Upload failed: ' + errText;
                        statusDiv.style.color = 'red';
                    }
                } catch (err) {
                    statusDiv.innerText = 'Error: ' + err.message;
                    statusDiv.style.color = 'red';
                }
            };

            async function searchSounds() {
                const query = document.getElementById('searchInput').value;
                const res = await fetch('/search?query=' + encodeURIComponent(query));
                const sounds = await res.json();
                
                const resultsDiv = document.getElementById('results');
                if (sounds.length === 0) {
                    resultsDiv.innerHTML = '<p>No sounds found.</p>';
                    return;
                }
                
                resultsDiv.innerHTML = sounds.map(s => `
                    <div class="sound-card">
                        <div class="tags">
                            ${s.tags.map(t => `<span class="tag">${t}</span>`).join('')}
                        </div>
                        <p style="margin: 0 0 10px 0; font-size: 0.9em; color: #555;">Duration: ${s.duration.toFixed(2)}s</p>
                        <audio controls src="/play/${s.id}" preload="metadata"></audio>
                    </div>
                `).join('');
            }
            
            // Initial load
            searchSounds();
        </script>
    </body>
    </html>
    """

# Automated Sourcing Pipeline
def scraper_task():
    # A real automated scraper fetching audio files from public domain/creative commons sites
    # For demonstration, we simulate scraping a public API or website
    sources = [
        "https://www.soundhelix.com/examples/mp3/SoundHelix-Song-1.mp3",
    ]
    
    time.sleep(10) # wait for server to start
    
    for url in sources:
        try:
            print(f"Scraping {url}...")
            response = requests.get(url, stream=True, timeout=10)
            if response.status_code == 200:
                file_id = str(uuid.uuid4())
                raw_path = os.path.join(UPLOAD_DIR, f"{file_id}_scraped.mp3")
                with open(raw_path, 'wb') as f:
                    for chunk in response.iter_content(chunk_size=8192):
                        f.write(chunk)
                
                try:
                    process_and_save_sound(raw_path, file_id)
                    print(f"Successfully scraped and processed {url}")
                except Exception as e:
                    print(f"Failed to process scraped file: {e}")
        except Exception as e:
            print(f"Scraping failed for {url}: {e}")
            
    # Then loop periodically
    while True:
        time.sleep(86400) # Daily scrape

threading.Thread(target=scraper_task, daemon=True).start()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
