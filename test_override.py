import json, os, subprocess

with open(os.path.expanduser("~/.claune.json"), "w") as f:
    json.dump({"sounds": {"tool:success": "/tmp/custom.wav"}}, f)
with open("/tmp/custom.wav", "w") as f:
    f.write("fake")

globals()['__file__'] = 'claune'
globals()['__name__'] = 'not_main'

class FakePopen:
    def __init__(self, cmd, *args, **kwargs):
        print('RAN:', cmd)
    def poll(self): return None

subprocess.Popen = FakePopen

with open('claune', 'r') as f:
    code = f.read()

exec(code, globals())
play_sound('tool:success')
