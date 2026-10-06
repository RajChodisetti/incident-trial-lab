#!/usr/bin/env python3
"""Capture validated local HTML using installed Chrome; no dependencies or desktop capture."""
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import tempfile

root = Path(__file__).resolve().parent.parent
chrome = Path('/Applications/Google Chrome.app/Contents/MacOS/Google Chrome')
if not chrome.is_file():
    raise SystemExit('Installed Google Chrome is required; nothing will be installed.')
subprocess.run(['go', 'run', './cmd/evidence-demo'], cwd=root, check=True)
out = root / 'docs/screenshots'
out.mkdir(exist_ok=True)
results = []
for name, page in [('01-incident', 'index'), ('02-comparison', 'comparison'), ('03-provenance', 'provenance')]:
    image = out / (name + '.png')
    with tempfile.TemporaryDirectory(prefix='demo-browser-', dir=root / '.build') as profile:
        captured = Path(profile) / 'capture.png'
        args = [str(chrome), '--headless=new', '--user-data-dir=' + profile,
                '--disable-background-networking', '--disable-background-mode',
                '--disable-sync', '--disable-extensions', '--disable-component-update',
                '--disable-breakpad', '--disable-crash-reporter', '--disable-gpu',
                '--no-first-run', '--no-default-browser-check', '--hide-scrollbars',
                '--window-size=1440,1200', '--screenshot=' + str(captured),
                (root / '.build/demo' / (page + '.html')).as_uri()]
        with (root / '.build' / (name + '-chrome.log')).open('wb') as log:
            process = subprocess.Popen(args, stdout=log, stderr=log)
            try:
                process.wait(timeout=25)
            except subprocess.TimeoutExpired:
                # Popen identifies only the process started by this invocation.
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
        if not captured.is_file():
            raise SystemExit('Local screenshot was not produced: ' + name)
        data = captured.read_bytes()
        if data[:8] != b'\x89PNG\r\n\x1a\n' or struct.unpack('>II', data[16:24]) != (1440, 1200):
            raise SystemExit('Unexpected screenshot format or dimensions.')
        # Overwrite bytes only after validation, preserving existing Library xattrs.
        image.write_bytes(data)
        results.append({'file': image.name, 'page': page + '.html', 'width': 1440,
                        'height': 1200, 'sha256': hashlib.sha256(data).hexdigest(),
                        'source_html_sha256': hashlib.sha256((root / '.build/demo' / (page + '.html')).read_bytes()).hexdigest()})
        print('Captured ' + image.name, flush=True)
(out / 'manifest.json').write_text(json.dumps({'mode': 'actual_headless_browser_of_recorded_evidence',
    'replay_loop': 'f2261299764ffd63', 'images': results}, indent=2) + '\n')
