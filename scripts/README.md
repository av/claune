# Claune Scripts

## Downloading sounds from boardsounds.com

Use the `download-boardsound.sh` script to easily download any sound from boardsounds.com without needing to dig into browser dev tools!

### Usage

Find the sound you want on boardsounds.com (e.g. `https://boardsounds.com/sound-effects/minecraft-click`)
Take the slug from the end of the URL (e.g. `minecraft-click`) and run:

```bash
./scripts/download-boardsound.sh minecraft-click my-file-name.mp3
```

If you don't specify an output file, it will default to `<slug>.mp3`.

### Example

```bash
./scripts/download-boardsound.sh among-us-role-reveal-sound
# Downloads to among-us-role-reveal-sound.mp3
```
