#!/bin/bash
# Download a sound from boardsounds.com based on its slug
# Usage: ./download-boardsound.sh <slug> [output_file.mp3]
# Example: ./download-boardsound.sh minecraft-click minecraft.mp3

if [ -z "$1" ]; then
    echo "Usage: $0 <slug> [output_file.mp3]"
    echo "Example: $0 minecraft-click"
    exit 1
fi

SLUG=$1
OUTPUT=${2:-"${SLUG}.mp3"}
URL="https://boardsounds.com/sound-effects/${SLUG}"

echo "Fetching page: $URL"
AUDIO_URL=$(curl -s "$URL" | grep -oE 'https://boardsounds.com/api/sound/play/[^"]+\.mp3' | head -n 1)

if [ -z "$AUDIO_URL" ]; then
    echo "Error: Could not find audio URL for $SLUG"
    exit 1
fi

echo "Downloading from: $AUDIO_URL"
curl -s -L "$AUDIO_URL" -o "$OUTPUT"
echo "Saved to $OUTPUT"
