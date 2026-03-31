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

## Verifying `claune play` regression evidence

Use `verify-play-regression-evidence.sh` to reproduce the `claune play` regression/fix relationship against git history without rewriting history.

```bash
./scripts/verify-play-regression-evidence.sh
```

The script checks that the same focused regression tests fail on the parent of the original `play` fix commit and pass on the current HEAD. This is honest best-possible evidence for QA now, but it does **not** prove the original chronological TDD red→green sequence happened in that order.
