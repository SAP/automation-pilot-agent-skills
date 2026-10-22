#!/usr/bin/env python3
"""Offline packaging checks; no tenant requests or credential access."""
import json
import re
import subprocess
from pathlib import Path
from urllib.parse import unquote
import yaml

ROOT = Path(__file__).resolve().parent.parent
skills = sorted((ROOT / 'skills').glob('*/SKILL.md'))
names = set()
for path in skills:
    match = re.match(r'^---\n(.*?)\n---\n', path.read_text(), re.S)
    assert match, f'{path}: missing frontmatter'
    front = yaml.safe_load(match[1])
    name = front['name']
    assert name == path.parent.name and re.fullmatch(r'[a-z0-9]+(?:-[a-z0-9]+)*', name)
    assert name not in names and len(name) <= 64
    names.add(name)
    assert isinstance(front['description'], str) and 0 < len(front['description']) <= 1024
    assert set(front) <= {'name', 'description', 'license', 'metadata', 'disable-model-invocation', 'allowed-tools'}
    if 'disable-model-invocation' in front:
        assert isinstance(front['disable-model-invocation'], bool)
for path in [*ROOT.glob('*.md'), *(ROOT / 'skills').rglob('*.md')]:
    # Markdown relative file links; anchors and external URLs are checked by their owners.
    for link in re.findall(r'\]\(([^)]+)\)', path.read_text()):
        link = link.split('#', 1)[0].strip('<>')
        if not link or link.startswith('/') or re.match(r'\w+://', link):
            continue
        assert (path.parent / unquote(link)).exists(), f'{path}: missing link {link}'
for path in [*(ROOT / '.claude-plugin').glob('*.json'), *(ROOT / 'skills').rglob('*.json')]:
    json.loads(path.read_text())
manifest = json.loads((ROOT / '.claude-plugin/plugin.json').read_text())
marketplace = json.loads((ROOT / '.claude-plugin/marketplace.json').read_text())
assert manifest['name'] == marketplace['plugins'][0]['name']
assert 'userConfig' not in manifest, 'unmapped configuration must not promise credential injection'
cli = ROOT / 'skills/automation-pilot-cli/scripts/autopi'
subprocess.run([str(cli), 'version'], check=True)
registry = json.loads(subprocess.check_output([str(cli), 'surface']))
for resource in registry:
    subprocess.run([str(cli), resource['name'], '--help'], check=True, stdout=subprocess.DEVNULL)
    for verb in resource['verbs']:
        subprocess.run([str(cli), resource['name'], verb, '--help'], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
for path in sorted((ROOT / 'skills/automation-pilot-command-authoring/examples').glob('*.command.json')):
    subprocess.run([str(cli), 'lint', str(path)], check=True)
policy = json.loads(subprocess.check_output([str(cli), 'policy']))
feature_names = {feature['name'] for feature in policy['features']}
documented_features = re.findall(r'^\| `(feature:[^`]+)` \|', (ROOT / 'skills/automation-pilot-tags/SKILL.md').read_text(), re.M)
assert documented_features and set(documented_features) <= feature_names, 'documented feature tags must be supported'
for duration in policy['controlDurations']:
    assert duration in (ROOT / 'skills/automation-pilot-cli/references/command-reference.md').read_text()
print(f'Validated {len(skills)} skills, links, JSON, manifests, templates, and shared policy')
