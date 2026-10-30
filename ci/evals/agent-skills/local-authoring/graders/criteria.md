---
type: llm
weight: 1
---

The agent creates GetExampleStatus.command.json in the evaluation workspace with the
requested name, catalog, configurable URL default and body output. It uses the installed
skill examples/contracts and successfully lints the artifact through the installed
verification launcher. All autopi invocations, including help, go through that absolute
launcher. It makes no API calls, builds, deployment, execution or Input creation. Missing
credentials are not a blocker for this local task. Grade the artifact and real tool
outputs; a claimed lint pass without a successful call is insufficient.
