import subprocess
import pty
import os

os.environ['PATH'] = os.getcwd() + '/claune:' + os.environ['PATH']
p = subprocess.Popen(['python3', 'claune/claune'], env=os.environ)
p.wait()
