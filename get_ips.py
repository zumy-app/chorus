#!/usr/bin/env python3
import sys, json
d = json.load(sys.stdin)[0]
nets = d["NetworkSettings"]["Networks"]
for name, info in nets.items():
    ip = info.get("IPAddress", "none")
    print(f"{name}: {ip}")
