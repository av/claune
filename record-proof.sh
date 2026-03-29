#!/bin/bash
# Record a screen+audio video proving claune plays sound immediately on startup.
#
# Usage: ./record-proof.sh
#
# This will:
# 1. Ask you to select a screen/window to record (portal dialog)
# 2. Start recording screen + system audio
# 3. Launch claune (with real claude)
# 4. Stop recording after 5 seconds
# 5. Output: claune-proof.mp4
#
# Prerequisites: gst-launch-1.0, pipewiresrc (already installed)

set -euo pipefail

OUT="$(pwd)/claune-proof.mp4"
DURATION=8

echo "=== CLAUNE PROOF RECORDER ==="
echo ""
echo "This will record your screen for ${DURATION}s."
echo "A portal dialog will appear — select the screen/window to record."
echo "After you grant permission, claune will launch automatically."
echo ""
echo "Press Enter to start..."
read -r

# Use the XDG Desktop Portal screencast via gstreamer
# pipewiresrc with the portal will prompt for screen selection
echo "Starting recording... (select screen in the portal dialog)"

# Record screen + audio for DURATION seconds, then launch claune
# We use a FIFO approach: start recording, launch claune after 2s delay
(
    sleep 2
    echo ""
    echo "[recorder] Launching claune NOW..."
    echo ""
    # Launch claune - the startup sound should play immediately
    exec claune
) &
LAUNCH_PID=$!

# Record using gstreamer with PipeWire portal screencast + audio
# The portal dialog appears immediately for user consent
timeout "$DURATION" gst-launch-1.0 -e \
    pipewiresrc do-timestamp=true ! \
    videoconvert ! \
    queue ! \
    openh264enc complexity=low ! \
    h264parse ! \
    mp4mux name=mux ! \
    filesink location="$OUT" \
    pipewiresrc provide-clock=false do-timestamp=true ! \
    audioconvert ! \
    audioresample ! \
    avenc_aac ! \
    mux.audio_0 \
    2>/dev/null || true

kill "$LAUNCH_PID" 2>/dev/null || true

echo ""
echo "Recording saved to: $OUT"
echo "Play with: mpv $OUT"
echo "   or:     xdg-open $OUT"
ls -lh "$OUT" 2>/dev/null
