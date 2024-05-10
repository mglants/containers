#!/usr/bin/env python

import requests
import json
import semver


# Get the latest version of matchbox

URL = "https://api.github.com/repos/isc-projects/kea/git/matching-refs/tags/Kea-"

def get_latest(channel):
    r = requests.get(URL)
    data = json.loads(r.text)
    versions = [ semver.Version.parse(release['ref'].replace('refs/tags/Kea-', '')) for release in data ]
    latest_version = max(versions)

    return str(latest_version).removeprefix("v")

if __name__ == "__main__":
    import sys
    channel = sys.argv[1]
    print(get_latest(channel))
