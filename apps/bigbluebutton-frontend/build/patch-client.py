#!/usr/bin/env python3
"""Preserve non-default ports in BBB's API discovery URL."""
from pathlib import Path
import subprocess

path = Path('/sources/bbb/bigbluebutton-html5/imports/api/bbb-web-api/index.ts')
text = path.read_text()
old = '''    const { hostname, protocol } = window.location;

    return new URL(route, `${protocol}//${hostname}${serverPathPrefix}`);'''
new = '''    const { origin } = window.location;

    return new URL(route, `${origin}${serverPathPrefix}`);'''
assert text.count(old) == 1, 'Upstream API discovery changed; review the port-preservation patch'
text = text.replace(old, new)
path.write_text(text)
# Execute the actual patched method body, rather than a second implementation.
body = text.split('private static buildURL(route: string) {', 1)[1].split('\n  }', 1)[0]
script = '''const assert = require('node:assert/strict');
const buildURL = new Function('window', 'route', process.argv[1]);
for (const [page, expected] of [
 ['https://localhost:18443/html5client/', 'https://localhost:18443/bigbluebutton/api'],
 ['https://bbb.example/html5client/', 'https://bbb.example/bigbluebutton/api'],
 ['https://localhost:18443/node/html5client', 'https://localhost:18443/node/bigbluebutton/api'],
 ['https://[::1]:18443/html5client/', 'https://[::1]:18443/bigbluebutton/api'],
]) {
 assert.equal(buildURL({location: new URL(page)}, 'bigbluebutton/api').href, expected);
}
console.log('PASS actual client API discovery: custom/default port, cluster prefix and IPv6');
'''
subprocess.run(['node', '-e', script, body], check=True)
