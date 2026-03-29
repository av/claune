import os
import subprocess
import json
import sqlite3
import math
import uuid
import tempfile
import boto3
from celery import Celery
from openai import OpenAI

UPLOAD_DIR = os.getenv("UPLOAD_DIR", "/tmp/meme_sounds")
os.makedirs(UPLOAD_DIR, exist_ok=True)
DB_PATH = os.path.join(UPLOAD_DIR, "sounds.db")

app = Celery(
    "tasks", broker="sqla+sqlite:///" + os.path.join(UPLOAD_DIR, "celerydb.sqlite")
)
app.conf.update(
    result_backend="db+sqlite:///" + os.path.join(UPLOAD_DIR, "celerydb.sqlite")
)

USE_S3 = os.getenv("USE_S3", "false").lower() == "true"

s3 = None
BUCKET_NAME = os.getenv("S3_BUCKET_NAME", "meme-sounds")

if USE_S3:
    try:
        s3 = boto3.client(
            "s3",
            endpoint_url=os.getenv("S3_ENDPOINT_URL"),
            aws_access_key_id=os.getenv("AWS_ACCESS_KEY_ID", "test"),
            aws_secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY", "test"),
        )
        s3.create_bucket(Bucket=BUCKET_NAME)
    except:
        pass


def get_openai_client():
    api_key = os.getenv("OPENAI_API_KEY")
    if not api_key:
        return None
    return OpenAI(api_key=api_key)


def get_audio_duration(filepath: str) -> float:
    cmd = [
        "ffprobe",
        "-v",
        "error",
        "-show_entries",
        "format=duration",
        "-of",
        "default=noprint_wrappers=1:nokey=1",
        filepath,
    ]
    try:
        result = subprocess.run(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True
        )
        return float(result.stdout.strip())
    except Exception:
        return 0.0


def validate_audio_file(filepath: str) -> bool:
    cmd = [
        "ffprobe",
        "-v",
        "error",
        "-show_entries",
        "format=format_name",
        "-of",
        "default=noprint_wrappers=1:nokey=1",
        filepath,
    ]
    try:
        result = subprocess.run(
            cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True
        )
        return bool(result.stdout.strip())
    except Exception:
        return False


@app.task
def process_audio_task(raw_path: str, file_id: str):
    if not validate_audio_file(raw_path):
        os.remove(raw_path)
        return "Invalid file"

    processed_filename = f"{file_id}_processed.ogg"
    processed_path = os.path.join(UPLOAD_DIR, processed_filename)

    # Zero-crossing trim logic: we apply afade to ensure it cuts cleanly without clicking
    cmd = [
        "ffmpeg",
        "-y",
        "-i",
        raw_path,
        "-t",
        "1.5",
        "-c:a",
        "libvorbis",
        "-af",
        "loudnorm=I=-16:TP=-1.5:LRA=11,afade=t=in:ss=0:d=0.01,afade=t=out:st=1.49:d=0.01",
        processed_path,
    ]
    subprocess.run(cmd, check=True, stderr=subprocess.PIPE, stdout=subprocess.PIPE)

    os.remove(raw_path)
    duration = get_audio_duration(processed_path)

    client = get_openai_client()
    tags = []
    is_safe = True
    embedding = []

    if client:
        try:
            with open(processed_path, "rb") as audio_file:
                transcript_response = client.audio.transcriptions.create(
                    model="whisper-1", file=audio_file
                )
            transcription = transcript_response.text

            completion = client.chat.completions.create(
                model="gpt-4o",
                response_format={"type": "json_object"},
                messages=[
                    {
                        "role": "system",
                        "content": "You are an AI that tags meme sounds. Return JSON with 'tags' (array of strings) and 'is_safe' (boolean).",
                    },
                    {
                        "role": "user",
                        "content": f"Transcription: '{transcription}'. Generate 3-5 tags. Set is_safe to false if NSFW.",
                    },
                ],
            )
            result = json.loads(completion.choices[0].message.content)
            tags = result.get("tags", ["meme"])
            is_safe = result.get("is_safe", True)

            # Embeddings
            emb_res = client.embeddings.create(
                input=" ".join(tags) + " " + transcription,
                model="text-embedding-ada-002",
            )
            embedding = emb_res.data[0].embedding
        except Exception as e:
            tags = ["error"]
            is_safe = False
            embedding = [0] * 1536
    else:
        tags = ["no-ai"]
        embedding = [0] * 1536

    if not is_safe and "error" in tags:
        if os.path.exists(processed_path):
            os.remove(processed_path)
        return "AI Failed"

    # Upload to S3 or save locally
    s3_key = f"sounds/{processed_filename}"
    if s3:
        try:
            s3.upload_file(processed_path, BUCKET_NAME, s3_key)
            s3_url = f"{s3.meta.endpoint_url}/{BUCKET_NAME}/{s3_key}"
            if os.path.exists(processed_path):
                os.remove(processed_path)
        except Exception:
            # Fallback to local
            s3_url = f"/play_local/{processed_filename}"
    else:
        # Save locally instead
        local_target = os.path.join(UPLOAD_DIR, processed_filename)
        if processed_path != local_target:
            import shutil

            shutil.copy(processed_path, local_target)
        s3_url = f"/play_local/{processed_filename}"
        # Keep the file if saving locally

    conn = sqlite3.connect(DB_PATH)
    c = conn.cursor()
    c.execute(
        "INSERT INTO sounds VALUES (?, ?, ?, ?, ?, ?, ?)",
        (
            file_id,
            processed_filename,
            json.dumps(tags),
            duration,
            is_safe,
            json.dumps(embedding),
            s3_url,
        ),
    )
    conn.commit()
    conn.close()

    return "Success"
