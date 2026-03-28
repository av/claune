import pty
import os
import sys
import select
import tty
import termios
import subprocess
import signal
import fcntl
import struct
import json
import tempfile
import atexit
import shutil
import re
import math
import wave
from datetime import datetime

SOUNDS_DIR = os.path.join(os.path.dirname(__file__), "sounds")

def generate_sound(path, freq, duration, vol=0.5):
    sample_rate = 44100
    n_samples = int(sample_rate * duration)
    with wave.open(path, 'w') as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(sample_rate)
        for i in range(n_samples):
            t = float(i) / sample_rate
            val = int(vol * 32767.0 * math.sin(2.0 * math.pi * freq * t))
            w.writeframes(struct.pack('<h', val))

def init_sounds():
    if not os.path.exists(SOUNDS_DIR):
        os.makedirs(SOUNDS_DIR)

    sounds_to_gen = {
        "fanfare.wav": (600, 0.5), # start
        "drumroll.wav": (200, 0.2), # tool start
        "tada.wav": (800, 0.3),     # success
        "sad-trombone.wav": (150, 0.5), # error
        "applause.wav": (400, 0.4), # done
    }

    for file, (freq, duration) in sounds_to_gen.items():
        path = os.path.join(SOUNDS_DIR, file)
        if not os.path.exists(path) or os.path.getsize(path) == 0:
            generate_sound(path, freq, duration)

init_sounds()

active_procs = []

def clean_procs():
    global active_procs
    active_procs = [p for p in active_procs if p.poll() is None]

def get_config():
    config_path = os.path.expanduser("~/.claune.json")
    config = {
        "mute": False,
        "volume": 1.0,
        "sounds": {}
    }
    if os.path.exists(config_path):
        try:
            with open(config_path, "r") as f:
                user_conf = json.load(f)
                config.update(user_conf)
        except Exception:
            pass
    return config

def should_mute(config):
    if config.get("mute"):
        return True
    
    # Check smart muting: 11 PM to 7 AM local time
    if "mute" not in config: # only if not explicitly false/true in config, but we default False above... let's say if not explicit
        # We need to see if it's explicitly set. Let's re-read raw.
        try:
            config_path = os.path.expanduser("~/.claune.json")
            if os.path.exists(config_path):
                with open(config_path, "r") as f:
                    raw = json.load(f)
                    if "mute" in raw:
                        return raw["mute"]
        except Exception:
            pass
        
        now = datetime.now()
        if now.hour >= 23 or now.hour < 7:
            return True
            
    return config.get("mute", False)

def play_sound(type_):
    config = get_config()
    if should_mute(config):
        return
        
    clean_procs()
    
    default_map = {
        "cli:start": "fanfare.wav",
        "tool:start": "drumroll.wav",
        "tool:success": "tada.wav",
        "tool:error": "sad-trombone.wav",
        "cli:done": "applause.wav",
    }
    
    # Check custom overrides
    sound_file = config.get("sounds", {}).get(type_)
    if sound_file:
        sound_path = os.path.expanduser(sound_file)
    else:
        file = default_map.get(type_)
        if not file:
            return
        sound_path = os.path.join(SOUNDS_DIR, file)

    if os.path.exists(sound_path) and os.path.getsize(sound_path) > 0:
        if sys.platform == "darwin":
            cmd = ["afplay"]
            if config.get("volume") is not None:
                cmd.extend(["-v", str(config["volume"])])
            cmd.append(sound_path)
        elif sys.platform == "win32":
            cmd = ["powershell", "-c", f"(New-Object Media.SoundPlayer '{sound_path}').PlaySync()"]
        else:
            player = "aplay" if shutil.which("aplay") else "paplay"
            cmd = [player, sound_path]
            
        try:
            p = subprocess.Popen(
                cmd,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            active_procs.append(p)
        except Exception:
            pass

def inject_hook_config():
    config_path = os.path.expanduser("~/.claude.json")
    config_data = {}
    if os.path.exists(config_path):
        try:
            with open(config_path, "r") as f:
                config_data = json.load(f)
        except Exception:
            pass

    existing_prompt = config_data.get("systemPrompt", "")
    directive = " [SYSTEM INSTRUCTION: You are in Claune mode. Act slightly dramatic and whimsical. When you use a tool, output exactly [CLAUNE_TOOL]. When you report a success, output [CLAUNE_SUCCESS]. When you report an error, output [CLAUNE_ERROR]. Do this silently before your normal output.]"
    config_data["systemPrompt"] = existing_prompt + directive

    fd, temp_path = tempfile.mkstemp(suffix=".json", prefix="claune_config_")
    with os.fdopen(fd, "w") as f:
        json.dump(config_data, f)

    atexit.register(lambda: os.remove(temp_path) if os.path.exists(temp_path) else None)
    return temp_path

def get_winsize(fd):
    try:
        winsize = fcntl.ioctl(fd, termios.TIOCGWINSZ, b"\x00" * 8)
        return struct.unpack("HHHH", winsize)
    except Exception:
        return (24, 80, 0, 0)

def set_winsize(fd, row, col, xpix=0, ypix=0):
    winsize = struct.pack("HHHH", row, col, xpix, ypix)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, winsize)

def process_stream_buffer(buffer):
    MARKERS = {
        b"[CLAUNE_TOOL]": "tool:start",
        b"[CLAUNE_SUCCESS]": "tool:success",
        b"[CLAUNE_ERROR]": "tool:error"
    }
    
    for marker, sound_type in MARKERS.items():
        if marker in buffer:
            play_sound(sound_type)
            buffer = buffer.replace(marker, b"")
            
    longest_partial_len = 0
    for marker in MARKERS:
        for i in range(1, len(marker)):
            if buffer.endswith(marker[:i]) and i > longest_partial_len:
                longest_partial_len = i
                
    utf8_partial_len = 0
    for i in range(1, min(5, len(buffer) + 1)):
        b = buffer[-i]
        if b & 0x80 == 0:
            break
        if b & 0xC0 == 0xC0:
            if b & 0xE0 == 0xC0: expected = 2
            elif b & 0xF0 == 0xE0: expected = 3
            elif b & 0xF8 == 0xF0: expected = 4
            else: expected = 1
            
            if i < expected:
                utf8_partial_len = i
            break
            
    keep_len = max(longest_partial_len, utf8_partial_len)
    
    if keep_len > 0:
        out_bytes = buffer[:-keep_len]
        rem_bytes = buffer[-keep_len:]
        return rem_bytes, out_bytes
        
    return b"", buffer

def spawn_pty(argv):
    play_sound("cli:start")
    
    pid, master_fd = pty.fork()
    if pid == pty.CHILD:
        os.execvp(argv[0], argv)

    is_atty = sys.stdin.isatty()
    old_tty = None
    if is_atty:
        old_tty = termios.tcgetattr(sys.stdin)
        tty.setraw(sys.stdin.fileno())

        def sigwinch_handler(sig, data):
            row, col, xpix, ypix = get_winsize(sys.stdin.fileno())
            set_winsize(master_fd, row, col, xpix, ypix)

        signal.signal(signal.SIGWINCH, sigwinch_handler)
        sigwinch_handler(None, None)

    inputs = [sys.stdin, master_fd]
    stream_buffer = bytearray()
    
    # JSON parsing buffer and state
    json_buffer = ""

    try:
        while inputs:
            r, w, e = select.select(inputs, [], [])

            if sys.stdin in r:
                try:
                    data = os.read(sys.stdin.fileno(), 1024)
                    if not data:
                        inputs.remove(sys.stdin)
                        os.write(master_fd, b"\x04")
                    else:
                        os.write(master_fd, data)
                except OSError:
                    if sys.stdin in inputs:
                        inputs.remove(sys.stdin)

            if master_fd in r:
                try:
                    data = os.read(master_fd, 1024)
                except OSError:
                    break
                if not data:
                    break

                stream_buffer.extend(data)
                rem_bytes, out_bytes = process_stream_buffer(stream_buffer)
                stream_buffer = bytearray(rem_bytes)
                
                if out_bytes:
                    out_str = out_bytes.decode("utf-8", errors="ignore")
                    json_buffer += out_str
                    if len(json_buffer) > 4096:
                        json_buffer = json_buffer[-4096:]
                        
                    # More robust regex for JSON markers
                    if re.search(r'{"type"\s*:\s*"tool_use"', json_buffer):
                        play_sound("tool:start")
                        json_buffer = re.sub(r'{"type"\s*:\s*"tool_use"', '', json_buffer, count=1)
                    if re.search(r'{"type"\s*:\s*"tool_result"', json_buffer):
                        play_sound("tool:success")
                        json_buffer = re.sub(r'{"type"\s*:\s*"tool_result"', '', json_buffer, count=1)
                        
                    sys.stdout.buffer.write(out_bytes)
                    sys.stdout.buffer.flush()

    finally:
        if stream_buffer:
            sys.stdout.buffer.write(bytes(stream_buffer))
            sys.stdout.buffer.flush()
            
        if is_atty and old_tty is not None:
            termios.tcsetattr(sys.stdin, termios.TCSADRAIN, old_tty)

        play_sound("cli:done")

        try:
            _, status = os.waitpid(pid, 0)
            sys.exit(os.waitstatus_to_exitcode(status))
        except ChildProcessError:
            pass

if __name__ == "__main__":
    temp_config_path = inject_hook_config()

    args = sys.argv[1:]

    # Only mock if explicitly requested via CLAUNE_TEST_MODE
    use_mock = os.environ.get("CLAUNE_TEST_MODE") == "1"

    if use_mock:
        mock = os.path.join(os.path.dirname(__file__), "mock_claude.py")
        if os.path.exists(mock):
            cmd = ["python3", mock] + args
        else:
            sys.stderr.write("Error: mock_claude.py is missing\n")
            sys.exit(1)
    else:
        cmd = ["claude"] + args + ["--settings", temp_config_path]

    spawn_pty(cmd)
