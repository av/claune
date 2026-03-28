#!/usr/bin/env python3
import sys
import time

sys.stdout.buffer.write(b"Starting task... ")
sys.stdout.buffer.flush()
time.sleep(1)

# This should trigger sound and be hidden
sys.stdout.buffer.write(b"[CLAUNE_TOOL]")
sys.stdout.buffer.flush()

time.sleep(1)
sys.stdout.buffer.write(b" done!\n")
sys.stdout.buffer.flush()
