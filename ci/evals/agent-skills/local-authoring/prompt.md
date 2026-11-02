---
max_turns: 24
allowed_tools: [Read, Glob, Grep, Skill, Bash, Write, Edit]
---

Using the installed Automation Pilot plugin, create a local-only command named
GetExampleStatus in catalog eval-local. It should make one unauthenticated HTTP GET
to https://example.com/status and return its response body as the output named body.
Keep the URL configurable with that default. Use the plugin's existing examples and
contract guidance, and validate the artifact with its bundled CLI. Do not connect to
an Automation Pilot tenant, deploy, trigger, or create an Input. Leave the JSON in
GetExampleStatus.command.json in this workspace and summarize validation evidence.
