# Build Summary (Round 2 Refinement)

Fixed ALL gaps and bugs identified by the QA agent:

1. **Data Ingestion Layer**: Replaced the mock crawler with a real `BeautifulSoup`-based web scraper that fetches from MyInstants, parses the DOM for play buttons, and downloads/processes raw meme audio files automatically in the background.
2. **Fatal Startup Crash**: Refactored the OpenAI client initialization to evaluate the API key dynamically (`get_openai_client()`). The application now starts successfully and falls back to stub logic if `OPENAI_API_KEY` is not present, avoiding immediate startup crashes.
3. **Strict Duration Compliance**: Changed the processing output format from `.mp3` to `.ogg` (using `libvorbis`). This bypasses the inherent MP3 frame padding issue, ensuring that processed audio file durations are strictly <= 1.5 seconds (exactly 1.50s if originally longer).
4. **Brittle Search Implementation**: Updated the search query to use SQLite's native `json_each` function (`WHERE json_each.value = ?`), guaranteeing exact string matching against the tags array instead of naive substring matching with `LIKE`.
5. **Silent Rejection on AI Failure**: Added explicit exception handling when the AI analysis falls back to an "error" state. The API now throws a 500 error instead of silently defaulting to `is_safe=False`, and the UI correctly handles and displays the rejection message to the user.
