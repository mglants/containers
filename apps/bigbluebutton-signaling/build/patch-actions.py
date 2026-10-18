from pathlib import Path
p = Path('src/imports/redis.ts')
s = p.read_text()
old = 'url: `redis://${REDIS_HOST}:${REDIS_PORT}`, // Construct URL from host and port.'
assert s.count(old) == 1, 'Unexpected actions Redis configuration'
assert s.count('socket: {') == 1
s = s.replace(old, 'password: process.env.BBB_REDIS_PASSWORD || undefined,')
s = s.replace('socket: {', 'socket: {\n      host: REDIS_HOST, port: REDIS_PORT,')
p.write_text(s)
