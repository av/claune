import subprocess
import time

p = subprocess.Popen(["python3", "claune"], env={"CLAUNE_TEST_MODE": "1", "PATH": "/usr/bin:/bin"})
p.wait()
