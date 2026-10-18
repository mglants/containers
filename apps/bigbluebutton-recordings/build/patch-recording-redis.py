from pathlib import Path
for name in ('Rakefile', 'scripts/rap-starter.rb', 'scripts/rap-enqueue.rb'):
    path = Path('/usr/local/bigbluebutton/core') / name
    text = path.read_text()
    old = 'Resque.redis = "#{redis_host}:#{redis_port}"'
    assert text.count(old) == 1, f'Unexpected Resque setup in {name}'
    path.write_text(text.replace(old, "Resque.redis = { host: redis_host, port: redis_port, password: props['redis_workers_password'] || props['redis_password'] }"))
