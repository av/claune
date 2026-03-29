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
import numpy as np
from worker import process_audio_task, get_openai_client, UPLOAD_DIR, DB_PATH

app = FastAPI(title="Meme Sounds API")

def init_db():
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute('''CREATE TABLE IF NOT EXISTS sounds
                 (id TEXT PRIMARY KEY, filename TEXT, tags TEXT, duration REAL, is_safe BOOLEAN, embedding TEXT, url TEXT)''')
    conn.commit()
    conn.close()

init_db()

class SoundMetadata(BaseModel):
    id: str
    tags: List[str]
    duration: float
    is_safe: bool
    url: str = ""

def cosine_similarity(v1, v2):
    return float(np.dot(v1, v2) / (np.linalg.norm(v1) * np.linalg.norm(v2)))

@app.post("/ingest")
async def ingest_sound(file: UploadFile = File(...)):
    file_id = str(uuid.uuid4())
    raw_path = os.path.join(UPLOAD_DIR, f"{file_id}_raw.mp3")
    with open(raw_path, "wb") as f:
        f.write(await file.read())
    # Send to scalable worker queue
    process_audio_task.delay(raw_path, file_id)
    return {"status": "processing", "id": file_id}

@app.get("/search", response_model=List[SoundMetadata])
async def search_sounds(query: str = ""):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    
    if query:
        client = get_openai_client()
        query_emb = None
        if client:
            try:
                emb_res = client.embeddings.create(input=query, model="text-embedding-ada-002")
                query_emb = emb_res.data[0].embedding
            except:
                pass
                
        c.execute("SELECT id, filename, tags, duration, is_safe, embedding, url FROM sounds WHERE is_safe = 1")
        rows = c.fetchall()
        
        results = []
        for r in rows:
            tags = json.loads(r[2])
            emb = json.loads(r[5]) if r[5] else []
            url = r[6] if len(r) > 6 and r[6] else ""
            
            score = 0.0
            if query_emb and emb and len(emb) == len(query_emb):
                score = cosine_similarity(query_emb, emb)
            elif query.lower() in [t.lower() for t in tags] or any(query.lower() in t.lower() for t in tags):
                score = 1.0
            
            if score > 0.7:  # Semantic similarity threshold
                results.append((score, SoundMetadata(id=r[0], tags=tags, duration=r[3], is_safe=bool(r[4]), url=url)))
        
        results.sort(key=lambda x: x[0], reverse=True)
        conn.close()
        return [r[1] for r in results[:50]]
    else:
        c.execute("SELECT id, filename, tags, duration, is_safe, url FROM sounds WHERE is_safe = 1 LIMIT 50")
        rows = c.fetchall()
        conn.close()
        results = []
        for r in rows:
            url = r[5] if len(r) > 5 and r[5] else ""
            results.append(SoundMetadata(id=r[0], tags=json.loads(r[2]), duration=r[3], is_safe=bool(r[4]), url=url))
        return results

@app.get("/play/{sound_id}")
async def play_sound(sound_id: str):
    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute("SELECT filename, url FROM sounds WHERE id=?", (sound_id,))
    row = c.fetchone()
    conn.close()
    if row:
        return {"url": row[1]} if row[1] else {"error": "Not uploaded yet"}
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
        <h1>Meme Sounds Library (AI Powered)</h1>
        
        <div class="upload-form">
            <h3>Upload New Sound</h3>
            <form id="uploadForm">
                <input type="file" id="audioFile" accept="audio/*" required>
                <button type="submit">Upload & Process in Background</button>
            </form>
            <div id="uploadStatus" style="margin-top: 10px; color: green;"></div>
        </div>

        <div>
            <input type="text" id="searchInput" placeholder="Semantic search (e.g., 'doge hit', 'fail')...">
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
                statusDiv.innerText = 'Uploading to worker queue...';
                statusDiv.style.color = 'blue';
                
                try {
                    const res = await fetch('/ingest', { method: 'POST', body: formData });
                    if (res.ok) {
                        statusDiv.innerText = 'Queued successfully! Please wait a moment and refresh search.';
                        statusDiv.style.color = 'green';
                        fileInput.value = '';
                    } else {
                        statusDiv.innerText = 'Upload failed.';
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
                    resultsDiv.innerHTML = '<p>No sounds found. Try a different query.</p>';
                    return;
                }
                
                resultsDiv.innerHTML = sounds.map(s => `
                    <div class="sound-card">
                        <div class="tags">
                            ${s.tags.map(t => `<span class="tag">${t}</span>`).join('')}
                        </div>
                        <p style="margin: 0 0 10px 0; font-size: 0.9em; color: #555;">Duration: ${s.duration.toFixed(2)}s</p>
                        <audio controls src="${s.url}" preload="metadata"></audio>
                    </div>
                `).join('');
            }
            
            searchSounds();
        </script>
    </body>
    </html>
    """

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
