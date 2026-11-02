# Human-in-the-Loop: GetUserChoice & GetUserInput

Pause an execution mid-flight and wait for a person, then continue with their answer. Use for an
approval gate, picking a target the flow can't know at trigger time, or confirming a destructive
action. For a value known at trigger time or fetched from a system, take a normal `inputKey` or an
HTTP/SQL step. Both prompt and answer are visible and have no sensitive variant, so never request
secrets at runtime.

Resolve the executor contract live with `autopi commands contract` before tenant use. The
IDs and field tables below are reference snapshots; local-only authoring records unavailable
contracts as assumptions.

## Executors

| Executor | Required inputs | Prompts for |
|----------|-----------------|-------------|
| `utils-sapcp:GetUserChoice:1` | `title`, `options` | A pick from `options`, a stringified `{"key":"label"}` map. The key is returned. |
| `utils-sapcp:GetUserInput:1` | `title` | Free text. The entered string is returned. |

Both output one key, `selection` (string): the chosen option's key, or the entered text. Read it
as `$(.<alias>.output.selection)`.

## Inputs

- `title`: the prompt shown to the human, giving them the context to decide. A string that can
  embed dynamic expressions to pull in data from earlier steps (e.g.
  `"Promote build $(.build.output.version) to which environment?"`).
- `options` (`GetUserChoice` only): the choices to display, as a `{"key":"label"}` map. The
  key is what the step returns in `selection` and what you compare against downstream. The
  value is the human-readable label shown for context. Like any object input it is passed
  stringified, and either side can carry dynamic expressions to build the choices from earlier
  output.

## Execution behavior

The step moves the whole execution to `INPUT_REQUIRED` and holds there.

A human must then provide the required data, a predefined choice or free text, to continue the
execution and shape what it does next.

At `INPUT_REQUIRED`, report the current prompt, offered choices, and execution ID. Applying
an answer belongs to the executions workflow named in the authoring skill's handoffs.
The answer is the offered key or authorized free text; monitoring leaves the decision
with the human.

The step waits for an answer, subject to execution limits and any configured Auto-Abort. Use it
in unattended or scheduled flows only when that wait is part of the intended behavior; specify
who will answer and how an unanswered run is handled.

## Minimal snippet

A choice step, and a later step gated on the returned key:

```json
{
  "execute": "utils-sapcp:GetUserChoice:1",
  "alias": "askEnv",
  "input": {
    "title": "Select the environment to promote to",
    "options": "{\"dev\":\"Development\",\"staging\":\"Staging\",\"prod\":\"Production\"}"
  },
  "description": "Ask which environment the caller intends",
  "progressMessage": null, "initialDelay": null, "pause": null,
  "when": null, "validate": null, "autoRetry": null, "repeat": null,
  "errorMessages": []
}
```

A downstream `when` branches on `$(.askEnv.output.selection)` with `"values": ["prod"]` (the option
key, not the label). `GetUserInput` is the same shape with only `title`.

Validate free text against the downstream contract before using it as a target or operation
argument. Treat the answer as data; it does not expand the command's intended scope.
