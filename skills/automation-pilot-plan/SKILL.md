---
name: automation-pilot-plan
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: Clarify an SAP Automation Pilot design through a structured interview and produce an implementation-ready specification.
---

# SAP Automation Pilot Planning

## Critical Rules (MANDATORY)

- First interpret the user's intent from the request and settled conversation: intended outcome, target systems, and known constraints. Ground that interpretation using [discovery](../automation-pilot-discovery/SKILL.md) to inspect existing commands and reusable Inputs before asking questions or considering new implementations. Record evaluated candidates and their fit, or the searched scope and evidence for a no-result. Use those findings to clarify only consequential gaps; propose new behavior only for needs the inspected content cannot meet.

A valid command can automate the wrong operation. Produce an implementation-ready
specification using the decisions already settled in the conversation. Use this sequence:
**interpret intent → discovery → clarify and specify → authoring → executions / scheduling**.

When the user delegates workflow design, choose reasonable defaults, document assumptions and
acceptance scenarios, and proceed within the authorized scope. Use the interview below only
for consequential decisions that remain open; delegation does not require another design
approval. Local-only work can record live feasibility as an explicit unverified assumption.

Map the automation as a **design tree**. Each decision branches into its dependent decisions, such as outcome → target systems and scope → exclusions or trigger → inputs. Work in **rounds**. The **frontier** contains decisions whose prerequisites are settled. Ask those questions in one round, then wait for the user's answers. Save questions that depend on an open decision for a later round.

Format a round like this:

```
❓ **Q1** - **<question title>**: <question body, may include choices>

➡️ <your recommended answer>

---

❓ **Q2** - **<question title>**: <question body, may include choices>

➡️ <your recommended answer>
```

Each round of answers reshapes the tree: settled decisions push the frontier outward. Recompute it and ask the next round.

Keep settled decisions from the conversation; do not interview them again. Ask only about
consequential gaps. For each new decision, present a recommendation, then wait. Ask the user to settle exclusions, destructive steps, and protected resources.

The frontier becomes empty after the user settles each item when its prerequisites allow: **outcome** (one sentence from the user's perspective) · **target systems** (specific names) · **scope and exclusions** (protected resources; state-changing automations need explicit exclusions) · **trigger model** (on-demand / scheduled / event) · **inputs** (type, requiredness, sensitivity, source) · **outputs** · **failure behavior** (including post-failure guarantees) · **success criteria**.

Finish the interview after visiting each branch and recording each assumption. Write the concrete specification in the conversation before asking for approval. Use the inspected discovery
evidence to verify feasibility and dependencies; revisit discovery when settled decisions change the required capabilities or Input keys. If implementation is already
authorized and no consequential gaps remain, proceed from the settled specification; otherwise
request final approval of that reviewable specification. Include the outcome, target systems, scope and exclusions, trigger model, inputs, outputs, failure behavior, assumptions, dependencies, and acceptance scenarios. Mark each accepted default as vetoable. Include a failure path and an exclusion case. Reference secret sources without recording their values. Give command authoring the specification with inspected command and Input IDs, reuse constraints, and any remaining implementation gaps.
