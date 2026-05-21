# Command Generation Skill

This skill helps create SAP Automation Pilot commands by providing patterns, expression documentation, and catalog references.

## Contents

```
command-generation/
├── SKILL.md                 # Core skill definition
├── README.md                # This file
├── references/
│   ├── catalogs.md          # Available commands by catalog
│   ├── expressions.md       # Dynamic expressions guide
│   ├── patterns.md          # Common command patterns
│   └── *.pdf                # Official SAP documentation
└── examples/
    ├── GetResourceWithRetry.command.json  # HTTP with retry/validation
    ├── WaitForOperation.command.json      # Polling pattern
    └── ProcessAppsBatch.command.json      # Batch processing
```

## Usage

The skill activates automatically when you ask to:

- "Create an automation pilot command"
- "Build an autopi command"
- "Help with dynamic expressions"
- "Create a composite command"

## Discovering Commands

For commands not listed in `catalogs.md`, use the **catalog-explorer** skill to query the API directly:

```bash
source .claude/skills/catalog-explorer/scripts/autopi-catalog-config.sh

# List all catalogs
autopi_api GET "/catalogs?own=false" | jq '.[] | {id, name}'

# Find commands in a catalog
autopi_api GET "/commands?catalog=applm-sapcp" | jq '.[] | {id, name}'
```
