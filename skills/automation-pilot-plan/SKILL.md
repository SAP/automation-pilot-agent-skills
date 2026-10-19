---
name: automation-pilot-plan
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: Clarify an SAP Automation Pilot design through a structured interview and produce an implementation-ready specification.
disable-model-invocation: true
---

# SAP Automation Pilot Planning

A valid command can automate the wrong operation. Interview the user until you agree on the required behavior, then write an implementation-ready specification. Use this sequence: **plan → discovery → authoring → executions / scheduling**.

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

Present each decision with a recommended answer, then wait. Ask the user to settle exclusions, destructive steps, and protected resources.

The frontier becomes empty after the user settles each item when its prerequisites allow: **outcome** (one sentence from the user's perspective) · **target systems** (specific names) · **scope and exclusions** (protected resources; state-changing automations need explicit exclusions) · **trigger model** (on-demand / scheduled / event) · **inputs** (type, requiredness, sensitivity, source) · **outputs** · **failure behavior** (including post-failure guarantees) · **success criteria**.

Finish the interview after visiting each branch and recording each assumption. **Wait for the user to confirm the shared understanding before acting.** Then write the specification in the conversation. Include the outcome, target systems, scope and exclusions, trigger model, inputs, outputs, failure behavior, assumptions, dependencies, and acceptance scenarios. Mark each accepted default as vetoable. Include a failure path and an exclusion case. Reference secret sources without recording their values. Ask discovery to find reusable building blocks, then give the specification to command authoring.
