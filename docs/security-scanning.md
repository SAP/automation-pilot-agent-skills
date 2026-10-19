# Security Scanning the Skills

This repo's skills are scanned with the [Cisco AI Defense skill-scanner](https://github.com/cisco-ai-defense/skill-scanner).
This guide shows how to re-run the scan with an LLM judge against an OpenAI-compatible
LLM gateway.

## 1. Install the scanner (isolated)

Use a throwaway virtualenv so nothing lands in the repo:

```bash
uv venv /tmp/skill-scanner-venv --python 3.13
uv pip install --python /tmp/skill-scanner-venv/bin/python cisco-ai-skill-scanner
/tmp/skill-scanner-venv/bin/skill-scanner --version   # expect: skill-scanner 2.3.0+
```

## 2. Configure the LLM provider

The scanner reads these environment variables for an OpenAI-compatible endpoint:

| Variable | Value |
|---|---|
| `SKILL_SCANNER_LLM_BASE_URL` | e.g. `http://localhost:6655/litellm/v1` |
| `SKILL_SCANNER_LLM_API_KEY`  | your gateway API key |
| `SKILL_SCANNER_LLM_MODEL`    | e.g. `anthropic--claude-4.8-opus` or `gpt-5.6-sol` |
| `SKILL_SCANNER_LLM_FORCE_JSON_OBJECT` | **set to `1`** |
| `SKILL_SCANNER_LLM_MAX_TOKENS` | `16384` (avoids truncated JSON) |

## 3. Run the scan

Scans all skills recursively and writes Markdown + JSON reports:

```bash
mkdir -p scan-reports

SKILL_SCANNER_LLM_BASE_URL="http://localhost:6655/litellm/v1" \
SKILL_SCANNER_LLM_API_KEY="<your-key>" \
SKILL_SCANNER_LLM_MODEL="anthropic--claude-4.8-opus" \
SKILL_SCANNER_LLM_FORCE_JSON_OBJECT=1 \
SKILL_SCANNER_LLM_MAX_TOKENS=16384 \

/tmp/skill-scanner-venv/bin/skill-scanner scan-all skills \
  --recursive \
  --use-llm \
  --llm-provider openai-compatible \
  --policy balanced \
  --format markdown \
  --format json \
  --output-markdown scan-reports/report-claude-4.8-opus.md \
  --output-json     scan-reports/report-claude-4.8-opus.json
```

## 4. Read the results

- **Markdown** (`scan-reports/report-*.md`) — human-readable per-skill verdicts.
- **JSON** (`scan-reports/report-*.json`) — machine-readable; `summary` has counts
  by severity, `results[]` has per-skill findings.

## Useful flags

| Flag | Purpose |
|---|---|
| `--policy {balanced,low-noise,quiet,strict,permissive}` | Noise level (`quiet` requires `--use-llm`) |
| `--format html --output-html report.html` | Interactive HTML report |
| `--use-behavioral` | Dataflow / taint analysis |
| `--fail-on-severity high` | Non-zero exit for CI gating |

Run `skill-scanner scan-all --help` for the full list.
