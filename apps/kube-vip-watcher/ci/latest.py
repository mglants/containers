#!/usr/bin/env python

import requests
import json

# Get the latest version of matchbox

URL = "https://api.github.com/repos/BBQigniter/kube-vip-watcher/tags"

def get_latest(channel):
    r = requests.get(URL)
    data = json.loads(r.text)
    versions = [ release['name'].replace('v', '') for release in data ]
    latest_version = max(versions)

    return str(latest_version)

if __name__ == "__main__":
    import sys
    channel = sys.argv[1]
    print(get_latest(channel))
