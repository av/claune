# Build Summary

Implemented fixes for all QA feedback items:
1. Replaced the stubbed Automated Sourcing Pipeline with a real background scraper thread that fetches audio from a public URL and processes it into the pipeline.
2. Implemented real AI integration using the `openai` Python SDK. When `OPENAI_API_KEY` is provided, the app will transcribe the audio using the `whisper-1` model and generate contextual tags and moderation decisions (`is_safe`) using the `gpt-4o` model.
3. Fixed the functional bug regarding `sounds.db` by explicitly storing it in the `UPLOAD_DIR` to prevent cwd issues.
4. Added an audio validation step using `ffprobe` in `process_and_save_sound` to ensure the uploaded file is a valid audio format *before* saving and calling FFmpeg. Invalid files are immediately deleted.
5. Built an actual HTML UI for the API. The index (`/`) now serves an HTML interface allowing users to upload meme sounds via a form, view processing status, search via tags, and play sounds directly from the browser without exposing raw database file paths.
