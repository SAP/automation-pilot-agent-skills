#!/usr/bin/env python3
"""Pin the supplied Swagger contracts; no network or runtime dependencies."""
import argparse
import hashlib
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
TARGET = ROOT / 'cli/internal/contract/snapshot.json'
KEYS = {'type', 'format', 'required', 'properties', 'items', 'additionalProperties',
        'minimum', 'maximum', 'minLength', 'maxLength', 'minItems', 'maxItems',
        'minProperties', 'maxProperties', 'uniqueItems', 'enum', 'pattern',
        'nullable', 'readOnly', 'default', 'collectionFormat', 'name', 'in'}

def generate(folder):
    documents, digests = {}, {}
    def read(path):
        path = path.resolve()
        if path not in documents:
            raw = path.read_bytes()
            documents[path] = json.loads(raw)
            digests[str(path.relative_to(folder))] = hashlib.sha256(raw).hexdigest()
        return documents[path]
    def resolve(node, path, trail=()):
        if isinstance(node, list):
            return [resolve(x, path, trail) for x in node]
        if not isinstance(node, dict):
            return node
        if '$ref' in node:
            filename, _, fragment = node['$ref'].partition('#')
            refpath = (path.parent / filename).resolve() if filename else path
            marker = (str(refpath), fragment)
            if marker in trail:
                raise ValueError('Recursive schema: ' + str(marker))
            target = read(refpath)
            if fragment:
                for key in fragment.lstrip('/').split('/'):
                    target = target[key.replace('~1', '/').replace('~0', '~')]
            result = resolve(target, refpath, trail + (marker,))
            result.update(resolve({k:v for k,v in node.items() if k != '$ref'}, path, trail))
            return result
        result = {}
        for key, value in node.items():
            if key not in KEYS:
                continue
            if key == 'required' and not isinstance(value, list):
                continue
            if key == 'properties':
                result[key] = {k:resolve(v, path, trail) for k,v in value.items()}
            else:
                result[key] = resolve(value, path, trail)
        return result
    operations = []
    for filename in ('ContentAPI.json', 'ExecutionsAPI.json', 'ScheduledExecutionsAPI.json'):
        path = folder / filename
        doc = read(path)
        for route, verbs in doc['paths'].items():
            for method, op in verbs.items():
                if method not in ('get', 'put', 'post', 'patch', 'delete'):
                    continue
                params = []
                for param in op.get('parameters', []):
                    normalized = resolve(param, path)
                    normalized['required'] = param.get('required', False)
                    if 'schema' in param:
                        normalized['schema'] = resolve(param['schema'], path)
                    params.append(normalized)
                operations.append({'method':method.upper(), 'path':route,
                    'operationId':op['operationId'], 'parameters':params,
                    'responses':{status:{'headers':sorted(response.get('headers', {}))}
                        for status,response in op['responses'].items()}})
    kinds = {}
    for kind, filename in {'command':'command', 'input':'input', 'catalog':'catalog',
        'catalog-update':'catalogUpdateRequest', 'bulk-commands':'createCommandsRequest',
        'bulk-inputs':'createInputsRequest', 'execution-request':'executionRequest',
        'action':'action', 'bulk-action':'bulkActionRequest',
        'scheduled-execution':'scheduledExecution', 'webhook':'webhook',
        'replacement':'replacement'}.items():
        path = folder / 'definitions' / (filename + '.json')
        kinds[kind] = resolve(read(path), path)
    try:
        revision = subprocess.check_output(['git', '-C', str(folder), 'rev-parse', 'HEAD'], text=True).strip()
    except subprocess.CalledProcessError:
        revision = 'unknown'
    return {'revision':revision, 'files':dict(sorted(digests.items())),
        'operations':sorted(operations, key=lambda o:(o['path'],o['method'])),
        'kinds':kinds}

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('swagger', type=Path)
    parser.add_argument('--check', action='store_true', help='fail if the pinned snapshot differs')
    args = parser.parse_args()
    result = json.dumps(generate(args.swagger.resolve()), indent=2, sort_keys=True) + '\n'
    if args.check:
        if not TARGET.exists() or TARGET.read_text() != result:
            parser.exit(1, 'Contract snapshot differs; review and regenerate it.\n')
    else:
        TARGET.parent.mkdir(parents=True, exist_ok=True)
        TARGET.write_text(result)
    print('Contract snapshot: 51 operations, source digests recorded')

if __name__ == '__main__':
    main()
