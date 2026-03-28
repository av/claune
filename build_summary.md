# Build Summary

The following bugs and gaps identified in the QA feedback have been fully resolved:

1. **Missing Features**:
   - The primary executable has been set to `claune` without a `.py` extension. Removed `claune.py` and updated `package.json` to configure the correct bin mapping, ensuring a true drop-in replacement CLI.

2. **Functional Bugs**:
   - Fixed the flawed mock detection condition. The fallback to `mock_claude.py` is now strictly triggered by setting `CLAUNE_TEST_MODE=1`, ensuring execution via absolute paths does not accidentally intercept the real Anthropic CLI.
   - Refactored `process_stream_buffer` to correctly accumulate streamed chunks instead of scanning fixed 1024-byte blocks. Implemented a robust sliding window and prefix matching to capture and conceal `[CLAUNE_TOOL]`, `[CLAUNE_SUCCESS]`, and `[CLAUNE_ERROR]` markers even when streamed character-by-character. JSON payloads for `tool_use` and `tool_result` are also robustly matched using regexes to handle whitespace, newlines, and cross-chunk payloads.

3. **UX/Design Issues**:
   - The streaming output no longer leaks intercepted AI intent markers back to the user console. The sliding window cleanly holds potential prefixes up to their maximum length and strips them out upon full match, ensuring smooth streaming UI while maintaining accurate, real-time AI-driven intent mapping to audio cues.
   
4. **Encoding Issues**:
   - Fixed a bug in `process_stream_buffer` that corrupted multi-byte Unicode characters split across stream chunks. Replaced the `utf-8` decode loop with a byte-level stream processing mechanism that correctly identifies and retains incomplete multi-byte UTF-8 sequences in the buffer until they can be fully read and output, preventing replacement characters (`\xef\xbf\xbd`) from being emitted.
