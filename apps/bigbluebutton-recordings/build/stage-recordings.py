#!/usr/bin/env python3
"""Stage BBB recording scripts and adapt journald logging for containers."""
from pathlib import Path
import os
import re
import shutil

root = Path('/sources/bbb/record-and-playback')
out = Path('/out/core')
out.mkdir(parents=True)
for directory in ('lib', 'scripts'):
    shutil.copytree(root / 'core' / directory, out / directory)
for name in ('Gemfile', 'Gemfile.lock', 'Rakefile'):
    shutil.copyfile(root / 'core' / name, out / name)
for component in ('presentation', 'notes', 'podcast', 'screenshare', 'video'):
    shutil.copytree(root / component / 'scripts', out / 'scripts', dirs_exist_ok=True)
# The notes scripts still load package defaults directly; use BBB's override-aware
# reader so published links honor the configured public hostname and protocol.
for phase in ('process', 'publish'):
    path = out / 'scripts' / phase / 'notes.rb'
    text = path.read_text()
    old = "BigBlueButton.load_yaml('../../core/scripts/bigbluebutton.yml')"
    if text.count(old) != 1:
        raise ValueError(f'Upstream notes configuration loading changed: {phase}')
    text = text.replace(old, 'BigBlueButton.read_props')
    text, count = re.subn(r'Logger\.new\("#\{log_dir\}/notes/[^\n]+', 'Logger.new($stdout)', text)
    if count != 1:
        raise ValueError(f'Upstream notes logging changed: {phase}')
    path.write_text(text)
for path in list(out.rglob('*.rb')) + [out / 'scripts/caption/presentation']:
    if path.is_file():
        text = path.read_text()
        text = re.sub(r'Journald::Logger\.new\([^\n)]*\)', 'BigBlueButton::ContainerLogger.new', text)
        path.write_text(text)
props = out / 'scripts/bigbluebutton.yml'
version = os.environ['VERSION']
props.write_text(re.sub(r"^bbb_version:.*", f"bbb_version: '{version}'", props.read_text(), flags=re.M))

boot = out / 'lib/boot.rb'
boot.write_text(boot.read_text() + """
require 'logger'
module BigBlueButton
  # Preserve Journald's tagged/block API while writing container logs to stdout.
  class ContainerLogger < Logger
    def initialize
      super($stdout)
    end
    def tag(tags)
      previous = progname
      self.progname = tags.inspect
      if block_given?
        begin
          yield
        ensure
          self.progname = previous
        end
      else
        self
      end
    end
  end
end
""")
