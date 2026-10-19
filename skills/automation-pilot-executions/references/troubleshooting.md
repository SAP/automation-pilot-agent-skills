# Execution troubleshooting

Collect execution evidence before matching an error pattern.

## Contents

- [Evidence sequence](#evidence-sequence)
- [Classifying failures](#classifying-failures)
- [Command and input errors](#command-and-input-errors)
- [HTTP, authentication, and connectivity errors](#http-authentication-and-connectivity-errors)
- [Script and SQL errors](#script-and-sql-errors)
- [Control-flow and expression errors](#control-flow-and-expression-errors)
- [Execution limits and platform errors](#execution-limits-and-platform-errors)

## Evidence sequence

1. Call `automation-pilot:get_execution`. Record the status, top-level `error`, `currentExecutorPath`, progress message, and executor states and timestamps. Find the executor where progress stopped; do not assume that the final array entry failed.
2. Call `automation-pilot:get_execution` with `include_input=true` when input values or encoding may matter. Compare the resolved keys and representations with the command contract without reproducing sensitive values.
3. Call `automation-pilot:get_execution_log` for the relevant executor path. The root path is `""`. If logging was not enabled before the run or the log has expired, state that limitation.
4. Call `automation-pilot:get_command` when the evidence points to validation, an expression, executor configuration, or another definition-level cause.

Conclude only after the available evidence accounts for the observed state. Quote the decisive error fragment, explain the cause in user terms, and name the next action.

## Classifying failures

Match the message from `automation-pilot:get_execution` or the failing executor's log against these patterns. Classify the cause before proposing a fix:

- **Run request/input:** the selected command or values supplied to this execution are missing, malformed, or incorrect. Correct them before considering a new run.
- **Command:** the command definition, expression, validation, or executor configuration is wrong. Send focused expression reproduction and correction to Dynamic Expressions, then hand the verified change to Command Authoring; hand other definition fixes directly to Command Authoring.
- **Environment:** the target service, credentials, network, quota, or tenant configuration caused the failure. Correct the environment or retry when it recovers.
- **Platform:** Automation Pilot rejected or suspended the run because of a platform limit or internal failure. Reduce the workload or escalate with the execution ID.

Transient timeouts, connectivity failures, rate limits, 5xx responses, and temporary sandbox failures may succeed on a new run. Fix deterministic failures such as bad input, 4xx responses, invalid expressions, and size limits first. Re-run only with user authorization and when repeating the command is safe; an execution may have changed external state before failing. After automatic retries run out, the final message repeats the underlying failure.

## Command and input errors

| Message pattern | Classification | Meaning and next step |
|---|---|---|
| `Missing input for <scope>: '<key>' is required` | Run request/input | The caller omitted the required value or the executor could not resolve it. Confirm with `automation-pilot:get_execution` (`include_input=true`), then supply it through `inputValues` or `inputReferences`. If a schedule triggered the run, fix the schedule's input binding. |
| `Missing valid combination of input values for authentication` or `Invalid combination of input values for the authentication related keys` | Run request/input | Authentication inputs are incomplete or conflict. Supply one supported authentication method, such as client credentials, a refresh token, a pre-built authorization header, or a certificate. Do not combine methods unless the command requires it. |
| Sensitive input rejects a hardcoded default | Command | A sensitive input cannot carry a literal `defaultValue`; it must reference an Automation Pilot Input. |
| `Command not found` or 404 on trigger | Run request/input | The command ID may have the wrong catalog, name, or version, or someone may have deleted the command. Verify the exact ID with `automation-pilot:get_command` before considering another run. |

## HTTP, authentication, and connectivity errors

| Message pattern | Classification | Meaning and next step |
|---|---|---|
| `HTTP request failed` with `Status: <code>` | Environment in most cases | Read the status and response body. For 401/403, fix credentials or scopes; for 404, check the URL and resource; for 429, wait or reduce request rate; for 5xx, retry after the target recovers. If the status is an acceptable result, add it to the command's success-code configuration. |
| `HTTP request timeout occurred` | Environment or command | Check whether the endpoint responds within the configured period. Increase the timeout when the expected operation needs more time. If a timeout is an explicitly expected outcome, encode that behavior in the command contract instead of reinterpreting this failed run. |
| `Connectivity issue`, `Connectivity timeout issue`, or `Host '<host>' is unknown` | Environment | Check the hostname, DNS, network path, destination or proxy settings, and target availability. Retry transient failures after ruling out configuration errors. |
| SSL certificate, hostname, handshake, or TLS error | Environment | Verify the server certificate chain, certificate hostname, and compatible TLS configuration. Do not disable certificate verification as a workaround. |
| OAuth error, failure to obtain an authorization header, or missing CSRF token | Run request/input or environment | Read the returned OAuth details, then check the token URL, credentials, grant type, scopes, and CSRF fetch request. A temporary token-service failure may succeed on a new run. |
| Destination is not allowed, Cloud Connector mapping is missing, or the Cloud Connector cannot reach the backend | Environment | Use an allowed destination and verify the subaccount connection, Location ID, access-control mapping, and backend availability. |
| `HTTP response content length is too big` | Command | Reduce the response with filtering, pagination, aggregation, or a narrower query instead of carrying the full payload through the execution. |

## Script and SQL errors

| Message pattern | Classification | Meaning and next step |
|---|---|---|
| `Script execution failed with exit code '<n>'` | Command or environment | Read the script output in the error. Fix the script or its inputs. Exit code `124` indicates a timeout. Add a non-zero code to the accepted exit codes when it represents success. |
| Script quota exhausted or preparation took too long | Environment | Reduce or stagger concurrent script executions and retry after capacity becomes available. Escalate repeated sandbox startup failures with the execution ID. |
| Script storage limit exceeded | Command | Reduce temporary files and generated artifacts, stream data where possible, and clean up intermediate files. |
| Database error with `errorCode` and `errorMessage` | Command or environment | Use the database's error text to distinguish invalid SQL, missing objects, constraints, permissions, and connectivity. Change accepted error codes when the command permits the database condition. |
| SQL result-size or timeout error | Command | Select fewer columns or rows, filter or aggregate the result, and optimize the statement. Increase the timeout when the operation needs more time. |
| Invalid or unsupported JDBC connection | Environment | Supply a supported HANA JDBC URL and remove invalid or disallowed connection properties. When using Cloud Connector, use an authentication method supported by that path. |

## Control-flow and expression errors

| Message pattern | Classification | Meaning and next step |
|---|---|---|
| `Validation failed` | Command | A `validate` or assertion condition did not match. Compare expected and actual values, then fix the data flow or condition. |
| `Repetition reached the maximum number of tries` | Command or environment | The polling condition did not become true. Inspect the polled operation and repeat condition before increasing the maximum count. |
| `INPUT_REQUIRED` | Expected execution state | The execution is waiting on a Confirm step. Inspect the requested choice. Responding or aborting requires the Automation Pilot UI or raw API. |
| Execution remains `RUNNING` | Needs evidence | Check `progressMessage` and executor states for a delay, retry backoff, external timeout, or repeat loop. Classify it as stuck when the evidence shows no progress. |

## Execution limits and platform errors

| Message pattern | Classification | Meaning and next step |
|---|---|---|
| Suspended after exceeding the 21-day runtime limit | Platform | Split the workflow and avoid unbounded waits or polling loops. `SUSPENDED` is terminal and is not a command failure. |
| Execution, request, or output exceeded its size limit | Command | Reduce inputs and output, trim HTTP or SQL results, and avoid accumulating large arrays or batch data between steps. |
| Running-execution quota reached or insufficient quota for a batch | Environment | Wait for active executions to finish, reduce concurrency, or use a smaller batch. |
| Unexpected execution error or a message requesting SAP Automation Pilot support | Platform | Capture the execution ID and relevant log, then contact support. Do not guess at an internal cause. |
