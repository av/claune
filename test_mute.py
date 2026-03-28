import json, datetime

config = {"mute": False, "volume": 1.0, "sounds": {}}

def should_mute(config):
    if config.get("mute"):
        return True
    
    # Check smart muting: 11 PM to 7 AM local time
    if "mute" not in config: # only if not explicitly false/true in config, but we default False above... let's say if not explicit
        # We need to see if it's explicitly set. Let's re-read raw.
        try:
            config_path = "/tmp/dummy" # os.path.expanduser("~/.claune.json")
            if os.path.exists(config_path):
                with open(config_path, "r") as f:
                    raw = json.load(f)
                    if "mute" in raw:
                        return raw["mute"]
        except Exception:
            pass
        
        # Fake it's midnight
        return True
            
    return config.get("mute", False)

print("Smart muting check returns:", should_mute(config))
