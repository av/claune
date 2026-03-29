import json

with open("/home/everlier/.claude.json", "r") as f:
    data = json.load(f)

data["hooks"] = {
    "PreToolUse": [
        {
            "matcher": ".*",
            "hooks": [
                {
                    "type": "command",
                    "command": "echo 'custom hook'",
                    "timeout": 5
                }
            ]
        }
    ]
}

with open("/home/everlier/.claude.json", "w") as f:
    json.dump(data, f, indent=2)
