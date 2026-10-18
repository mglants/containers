#!/usr/bin/env python3
"""Stage pinned upstream nginx routes with checked container adaptations."""
import argparse
import os
from pathlib import Path
import re

# Expected replacement counts force review when an upstream route changes.
ROUTES = {'bbb-web.nginx': ('bigbluebutton-web/bbb-web.nginx',
                   [('http://127.0.0.1:8090', 'http://bbb-web', 10),
                    ('/var/bigbluebutton/captions/', '/presentations/captions/', 1)]),
 'presentation-slides.nginx': ('bigbluebutton-web/nginx-confs/presentation-slides.nginx',
                               [('/var/bigbluebutton/$meeting_id_2/',
                                 '/presentations/$meeting_id_2/',
                                 4)]),
 'bbb-html5.nginx': ('build/packages-template/bbb-html5/bbb-html5.nginx.static', []),
 'graphql.nginx': ('build/packages-template/bbb-graphql-middleware/graphql.nginx',
                   [('http://127.0.0.1:8378', 'http://bbb-middleware', 1),
                    ('http://127.0.0.1:8185', 'http://bbb-hasura', 2),
                    ('Host $host;', 'Host $http_host;', 3)]),
 'bbb-shared-notes-server.nginx': ('build/packages-template/bbb-shared-notes-server/bbb-shared-notes-server.nginx',
                                   [('http://127.0.0.1:8090', 'http://bbb-web', 1),
                                    ('http://localhost:8787', 'http://bbb-notes', 3),
                                    ('Host $host;', 'Host $http_host;', 3)]),
 'webrtc-sfu.nginx': ('build/packages-template/bbb-webrtc-sfu/webrtc-sfu.nginx',
                      [('http://127.0.0.1:3008', 'http://bbb-signaling', 1)]),
 'livekit.nginx': ('build/packages-template/bbb-livekit/livekit.nginx',
                   [('http://127.0.0.1:7880', 'http://bbb-livekit', 1)]),
 'learning-dashboard.nginx': ('bbb-learning-dashboard/learning-dashboard.nginx', [])}


def stage(source, output, version):
    output.mkdir(parents=True, exist_ok=True)
    revision = 'v' + version
    for name, (relative, replacements) in ROUTES.items():
        text = (source / relative).read_text()
        for old, new, count in replacements:
            if text.count(old) != count:
                raise ValueError(f'Upstream nginx changed: {relative}: expected {count} occurrences of {old}')
            text = text.replace(old, new)
        (output / name).write_text(f'# BBB {revision}: {relative}; container address/path adaptations.\n' + text)
    # Use the current player route plus the upstream legacy redirect and published media route.
    player = (source / 'build/packages-template/bbb-playback/playback.nginx').read_text()
    legacy = (source / 'record-and-playback/presentation/scripts/presentation.nginx').read_text()
    blocks = []
    for location in ('/playback/presentation/2.0/playback.html', '/presentation'):
        found = re.findall(r'location ' + re.escape(location) + r' \{[^{}]*\}', legacy)
        if len(found) != 1:
            raise ValueError(f'Upstream playback route changed: {location}')
        blocks.append(found[0])
    if blocks[1].count('/var/bigbluebutton/published') != 1:
        raise ValueError('Upstream published-media storage path changed')
    published = blocks[1].replace('/var/bigbluebutton/published', '/presentations/published')
    # The historical /presentation route did not need an index in our split storage layout.
    published = re.sub(r'^[ \t]*index[^;]+;\n', '', published, flags=re.M)
    (output / 'playback.nginx').write_text(f'# BBB {revision}: upstream playback and presentation routes.\n' + blocks[0] + '\n' + player + published + '\n')


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--source', type=Path, default=Path('/sources/bbb'))
    parser.add_argument('--output', type=Path, default=Path('/out/nginx'))
    parser.add_argument('--version', default=os.environ.get('VERSION'), required='VERSION' not in os.environ)
    args = parser.parse_args()
    stage(args.source, args.output, args.version)
