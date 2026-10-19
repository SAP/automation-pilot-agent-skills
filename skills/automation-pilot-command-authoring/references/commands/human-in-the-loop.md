# Human-in-the-Loop: GetUserChoice & GetUserInput

Pause an execution mid-flight and wait for a person, then continue with their answer. Use for an
approval gate, picking a target the flow can't know at trigger time, or confirming a destructive
action. For a value known at trigger time or fetched from a system, take a normal `inputKey` or an
HTTP/SQL step. Both prompt and answer are visible and have no sensitive variant, so never request
secrets at runtime.

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

Because the pause blocks indefinitely, avoid HITL steps in unattended or scheduled flows unless an
approval has to happen.

## Minimal snippet

A choice step, and a later step gated on the returned key:

```json
{
  "execute": "utils-sapcp:GetUserChoice:1",
  "alias": "askEnv",
  "input": {
    "title": "Select the environment to promote to",
    "options": "{\"dev\":\"Development\",\"staging\":\"Staging\",\"prod\":\"Production\"}"
  }
}
```

A downstream `when` branches on `$(.askEnv.output.selection)` with `"values": ["prod"]` (the option
key, not the label). `GetUserInput` is the same shape with only `title`.
