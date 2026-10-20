# Validating the Skills

This repo's skills are validated with the [agent-ecosystem skill-validator](https://github.com/agent-ecosystem/skill-validator).
It checks structure compliance, content quality, cross-language contamination, and
link validity, and can add an LLM judge for quality scoring against an
OpenAI-compatible LLM gateway.

## 1. Install the validator

Pick whichever fits your setup:

```bash
# Homebrew
brew tap agent-ecosystem/tap
brew install skill-validator

# Go
go install github.com/agent-ecosystem/skill-validator/cmd/skill-validator@latest

skill-validator --version   # confirm it's on PATH
```

Pre-built binaries for macOS, Linux, and Windows (amd64 + arm64) are on the
[releases page](https://github.com/agent-ecosystem/skill-validator/releases).

## 2. Run the checks

`check` runs all four non-LLM validators (structure, links, content,
contamination) over every skill found under a directory:

```bash
mkdir -p validation-reports

skill-validator check skills \
  --strict \
  -o markdown > validation-reports/report.md
```

## 3. Add the LLM quality score (optional)

The scorer sends each file to a judge model. Point it at an OpenAI-compatible
gateway and set the key:

| Variable | Value |
|---|---|
| `OPENAI_API_KEY` | your gateway API key |
| `OPENAI_BASE_URL` | e.g. `http://localhost:6655/litellm/v1` |

```bash
OPENAI_API_KEY="<your-key>" \
OPENAI_BASE_URL="http://localhost:6655/litellm/v1" \
skill-validator score evaluate skills \
  --provider openai \
  --model anthropic--claude-4.8-opus \
  -o markdown > validation-reports/score.md
```

SKILL.md is scored 1-5 on Clarity, Actionability, Token Efficiency, Scope
Discipline, Directive Precision, and Novelty. Reference files are scored on
Clarity, Instructional Value, Token Efficiency, Novelty, and Skill Relevance.

Scores are cached. View them without re-calling the API:

```bash
skill-validator score report skills --list       # tabular summary
skill-validator score report skills --compare    # compare across models
```

Add `--rescore` to `score evaluate` to overwrite the cache.

## 4. Read the results

- **Markdown** (`-o markdown`) — GitHub-flavored, human-readable per-skill verdicts.
- **JSON** (`-o json`) — machine-readable.
- **compact** (`-o compact`) — one line per skill.
- **text** (default) — verbose.

Exit codes: `0` clean, `1` errors, `2` warnings only, `3` CLI/usage error.
