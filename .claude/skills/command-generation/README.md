# SAP Automation Pilot Skill for Claude Code

A Claude Code skill that helps create SAP Automation Pilot commands by providing access to 380+ reference commands, dynamic expression documentation, and common patterns.

## Installation

### Option 1: Clone/Copy to Project (Recommended)

Copy the skill directory into your project's `.claude/skills/` folder:

```bash
# From your project root
mkdir -p .claude/skills
cp -r /path/to/sap-automation-pilot .claude/skills/
```

The skill will be automatically discovered when Claude Code runs in that project.

### Option 2: Install Globally

Copy to your global Claude Code skills directory:

```bash
mkdir -p ~/.claude/skills
cp -r /path/to/sap-automation-pilot ~/.claude/skills/
```

This makes the skill available in all projects.

### Option 3: Git Submodule

Add as a submodule in your project:

```bash
git submodule add <repo-url> .claude/skills/sap-automation-pilot
```

### Option 4: Share as ZIP

Package and distribute as a ZIP file:

```bash
# Create distributable ZIP
cd .claude/skills
zip -r sap-automation-pilot.zip sap-automation-pilot/

# Recipients extract to their .claude/skills/
unzip sap-automation-pilot.zip -d ~/.claude/skills/
```

## Usage

Once installed, the skill activates automatically when you ask Claude Code to:

- "Create an automation pilot command"
- "Build an autopi command"
- "Help with dynamic expressions"
- "Create a composite command"
- "Write a workflow command for SAP BTP"

### Example Prompts

```
Create an automation pilot command that restarts a CF app and waits for it to be healthy

Build a command that lists all JIRA issues and posts a summary to Slack

Help me write a polling command that waits for a service instance to be ready
```

## Skill Contents

```
sap-automation-pilot/
├── SKILL.md                    # Core skill definition
├── README.md                   # This file
├── references/
│   ├── expressions.md          # Dynamic expressions guide
│   ├── patterns.md             # 12 common command patterns
│   ├── catalogs.md             # All available catalogs
│   ├── *.pdf                   # Official SAP documentation
│   └── content/                # 380+ reference commands
│       ├── http/               # HTTP requests
│       ├── cf/                 # Cloud Foundry
│       ├── applm/              # App lifecycle
│       ├── jira/               # JIRA integration
│       └── ...                 # 20+ more catalogs
└── examples/
    ├── composite-http.json     # HTTP with retry/validation
    ├── polling-workflow.json   # Polling pattern
    └── foreach-batch.json      # Batch processing
```

## Updating the Skill

To update the packaged content library:

```bash
# Re-copy content from source
cp -r /path/to/autopi-content/content/* .claude/skills/sap-automation-pilot/references/content/
```

## Verification

Check the skill is installed correctly:

```bash
# Should show SKILL.md with valid frontmatter
head -5 ~/.claude/skills/sap-automation-pilot/SKILL.md

# Expected output:
# ---
# name: sap-automation-pilot
# description: This skill should be used when...
# version: 1.0.0
# ---
```

## Requirements

- Claude Code CLI
- No additional dependencies

## License

Internal use - SAP Automation Pilot reference content.
