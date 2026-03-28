#!/usr/bin/env python3
import sys
import time
sys.stdout.buffer.write(b"[CLAUNE_TOOL]")
sys.stdout.buffer.flush()
time.sleep(0.1)
sys.stdout.buffer.write(b"[CLAUNE_SUCCESS]")
sys.stdout.buffer.flush()
time.sleep(0.1)
sys.stdout.buffer.write(b"[CLAUNE_ERROR]")
sys.stdout.buffer.flush()
time.sleep(1)
