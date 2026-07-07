# Available Command Catalogs

This reference lists all available built-in catalogs and their key commands.

## IMPORTANT: Catalog Naming Convention

- **Built-in SAP commands** (this file): Use `sapcp` suffix when referencing (e.g., `http-sapcp:HttpRequest:1`)
- **Your generated commands**: Use `<<<TENANT_ID>>>` suffix (e.g., `mycommands-<<<TENANT_ID>>>:MyCommand:1`)
- **ForEach**: Always use `ForEach:2`, never `ForEach:1` (deprecated)

## NOTE: This List May Not Be Complete

SAP continuously adds new commands and catalogs to Automation Pilot. If a catalog or command you need is not listed here, use the **[catalog-explorer](../../automation-pilot-catalog-explorer/SKILL.md)** skill to search for it via the API:

```bash
# List all available catalogs
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/catalogs?own=false"

# Search for commands in a specific catalog
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands?catalog=<catalog-id>"

# Get full command definition
curl -s -u "$AUTOPI_USERNAME:$AUTOPI_PASSWORD" \
  "https://$AUTOPI_HOSTNAME/api/v1/commands/<catalog>:<command>:<version>"
```

## aicore-sapcp (AI Core)

Generative AI operations.

| Command | Description |
|---------|-------------|
| `GenerateGpt4OmniCompletion:1` | GPT-4 Omni completion (recommended) |

Note: GPT-3.5 and GPT-4 non-Omni commands are deprecated. Use GPT-4 Omni.

## ans-sapcp (Alert Notification)

Alert Notification Service.

| Command | Description |
|---------|-------------|
| `ListMatchedAnsEvents:1` | List matched events |
| `ListUndeliveredAnsEvents:1` | List undelivered events |
| `SendAnsEvent:1` | Send alert event |

## applm-sapcp (Application Lifecycle)

Application lifecycle management.

| Command | Description |
|---------|-------------|
| `AddCfAppInstances:1` | Increase app instance count |
| `AddNeoAppInstances:1` | Add Neo app instances |
| `ApplyJavaAppOsPatches:1` | Apply OS patches to Java app |
| `DisableJavaProcess:1` | Disable Java process |
| `EnableJavaProcess:1` | Enable Java process |
| `GetCfAppEnvironmentVariables:1` | Get app environment variables |
| `GetCfAppInstances:1` | Get all app instances |
| `GetCfAppState:2` | Get application state and details |
| `GetDataSourceBindings:1` | Get Neo data source bindings |
| `GetJavaAppState:1` | Get Java app details |
| `GetJavaApps:1` | List Java apps in Neo subaccount |
| `GetJavaProcessState:1` | Get Java process details |
| `GetSensitiveCfAppEnvironmentVariables:1` | Get sensitive app env vars |
| `ListCfApps:1` | List all CF applications in space |
| `RemoveCfApp:1` | Delete CF application |
| `RemoveCfAppInstances:1` | Decrease app instance count |
| `RemoveNeoAppInstances:1` | Remove Neo app instances |
| `RestageCfApp:1` | Restage CF application |
| `RestartCfApp:1` | Restart CF application (rolling or quick) |
| `RestartJavaApp:1` | Restart Neo Java application |
| `RestartJavaProcess:1` | Restart Java process |
| `SetCfAppEnvironmentVariables:1` | Set app environment variables |
| `SetCfAppInstances:1` | Set exact app instance count |
| `SetCfAppResources:1` | Set app disk/memory quota |
| `SetNeoAppInstances:1` | Set Neo app instance count |
| `StartCfApp:1` | Start CF application |
| `StartJavaApp:1` | Start Neo Java application |
| `StartJavaProcess:1` | Start Java process |
| `StopCfApp:1` | Stop CF application |
| `StopJavaApp:1` | Stop Neo Java application |
| `StopJavaProcess:1` | Stop Java process |
| `TerminateCfAppInstance:1` | Terminate specific app instance |
| `TriggerAddNeoAppInstance:1` | Trigger async Neo instance add |
| `TriggerRemoveCfApp:1` | Trigger async app removal |
| `TriggerRemoveNeoAppInstance:1` | Trigger async Neo instance remove |
| `TriggerStartCfApp:1` | Trigger async app start |
| `TriggerStopCfApp:1` | Trigger async app stop |

## autopi-sapcp (Self-Management)

Automation Pilot operations on itself.

| Command | Description |
|---------|-------------|
| `CheckInputExistence:1` | Check if input exists |
| `CreateInput:1` | Create input |
| `ExecuteAction:2` | Execute action on execution |
| `ExecuteActionWithDelay:2` | Execute action with delay |
| `RunExecution:1` | Trigger and wait for execution |
| `UpdateInput:1` | Update input |

## baf-sapcp (Business Agent Foundation)

SAP Business Agent Foundation for AI agents and conversation threads. (Internal use)

| Command | Description |
|---------|-------------|
| `AwaitAgentReply:1` | Poll for agent response |
| `CreateThread:1` | Create conversation thread |
| `DeleteThread:1` | Delete thread and messages |
| `GetAgents:1` | List configured agents |
| `GetThread:1` | Get thread details and history |
| `InvokeThread:1` | Send message to thread |
| `SendMessage:1` | Create thread, send message, wait for response |

## calmhm-sapcp (Cloud ALM Health Monitoring)

SAP Cloud ALM for Operations - health monitoring and remediation.

| Command | Description |
|---------|-------------|
| `AddCloudAlmCfAppInstance:1` | Add CF app instance |
| `GetCloudAlmCfAppEvents:1` | Get CF app events |
| `GetCloudAlmCfAppLogs:1` | Get CF app logs |
| `GetCloudAlmCfAppState:1` | Get CF app state |
| `PushCloudAlmGaugeMetric:1` | Report gauge metrics |
| `PushCloudAlmQuotaMetric:1` | Report quota metrics |
| `PushCloudAlmStatusMetric:1` | Report status metrics |
| `RemoveCloudAlmCfAppInstance:1` | Remove CF app instance |
| `RestageCloudAlmCfApp:1` | Restage CF app |
| `RestartCloudAlmCfApp:1` | Restart CF app |

## cf-sapcp (Cloud Foundry)

Cloud Foundry operations.

| Command | Description |
|---------|-------------|
| `AssignCfOrgRole:1` | Assign role in org |
| `AssignCfSpaceRole:1` | Assign role in space |
| `BindCfServiceInstance:1` | Bind service to app |
| `CancelCfTask:1` | Cancel running task |
| `CreateCfServiceBinding:1` | Create service binding |
| `CreateCfServiceInstance:1` | Create service instance |
| `CreateCfSpace:1` | Create new space |
| `CreateCfTask:1` | Create task for CF application |
| `DeleteCfServiceInstance:1` | Delete service instance |
| `DeleteCfSpace:1` | Delete space |
| `ExecuteCfTask:1` | Create task and wait for completion |
| `GetAccessToken:1` | Get CF access token |
| `GetCfAppServiceBindings:1` | List service bindings for app |
| `GetCfServiceInstance:1` | Get service instance details |
| `GetCfSpace:1` | Get space details |
| `GetCfTask:1` | Get task status |
| `GetRefreshToken:1` | Get CF refresh token |
| `ListCfOrgUsers:1` | List users in org |
| `ListCfOrgs:1` | List all organizations |
| `ListCfServiceInstances:1` | List service instances in space |
| `ListCfSpaceUsers:1` | List users in space |
| `ListCfSpaces:1` | List spaces in org |
| `ListCfTasks:1` | List tasks for application |
| `ResolveCfIds:1` | Resolve org/space/app names to GUIDs |
| `TriggerUpdateCfServiceInstance:1` | Trigger async service instance update |
| `UnassignCfOrgRole:1` | Unassign role from org |
| `UnassignCfSpaceRole:1` | Unassign role from space |
| `UnbindCfServiceInstance:1` | Unbind service from app |
| `UpdateCfServiceInstance:1` | Update service instance |
| `WaitCfJob:1` | Wait for async job completion |

## cis-sapcp (Cloud Management Service)

SAP Cloud Management Service for directories, subaccounts, and entitlements.

| Command | Description |
|---------|-------------|
| `CreateDirectory:2` | Create directory in global account |
| `CreateSubAccount:1` | Create subaccount |
| `CreateSubscription:2` | Subscribe to multitenant app |
| `DecreaseQuotaEntitlement:2` | Decrease quota |
| `DeleteDirectoryEntitlement:2` | Remove directory entitlement |
| `DeleteEntitlement:2` | Remove entitlement |
| `DeleteServiceManagerBinding:1` | Delete SM instance and binding |
| `DeleteSubAccount:1` | Delete subaccount |
| `DeleteSubscription:2` | Unsubscribe from app |
| `EnableCloudFoundry:1` | Enable CF in subaccount |
| `GetDirectory:1` | Get directory details |
| `GetSubAccount:1` | Get subaccount details |
| `IncreaseQuotaEntitlement:2` | Increase quota |
| `ListDirectoryEntitlements:1` | List directory entitlements |
| `ListEntitlements:1` | List subaccount entitlements |
| `ListSubAccounts:1` | List all subaccounts |
| `ListSubscriptions:1` | List subscriptions |
| `ObtainServiceManagerBinding:1` | Create SM instance and binding |
| `SetDirectoryEntitlement:2` | Set directory entitlement |
| `SetEntitlement:2` | Set service plan entitlement |
| `TriggerDeleteDirectory:1` | Trigger async directory deletion |
| `UpdateDirectory:1` | Update directory |

## cld-sapcp (Cloud Landscape Directory)

SPC Cloud Landscape Directory for querying systems, tenants, and host instances. (Internal use)

| Command | Description |
|---------|-------------|
| `GetCldSystem:1` | Retrieve system from CLD |
| `GetCldTenant:1` | Retrieve tenant from CLD |
| `ListCldHostInstances:1` | List host instances of system |
| `ListCldResources:1` | List resources with OData filter |
| `ListCldSystems:1` | List systems by properties |
| `ListCldTenants:1` | List tenants by properties |

## ctms-sapcp (Cloud Transport Management)

Cloud Transport Management Service for transport nodes, routes, and requests.

| Command | Description |
|---------|-------------|
| `CreateTransportNode:1` | Create transport node |
| `CreateTransportRoute:1` | Create transport route |
| `DeleteTransportNode:1` | Delete transport node |
| `DeleteTransportRoute:1` | Delete transport route |
| `ForwardTransportRequests:1` | Forward requests |
| `GetTransportNode:1` | Get transport node details |
| `GetTransportRoute:1` | Get transport route details |
| `ImportAllTransportRequests:1` | Import all requests |
| `ImportTransportRequests:1` | Import specified requests |
| `ListTransportNodes:1` | List all transport nodes |
| `ListTransportRequests:1` | List transport requests |
| `ListTransportRoutes:1` | List all transport routes |
| `RemoveTransportRequests:1` | Remove requests |
| `ResetTransportRequests:1` | Reset requests |
| `UpdateTransportNode:1` | Update transport node |
| `UpdateTransportRoute:1` | Update transport route |

## dblm-sapcp (Database Lifecycle)

Database operations.

| Command | Description |
|---------|-------------|
| `ApplyDbSystemOsPatches:1` | Apply OS patches |
| `GetDatabases:1` | List Neo databases |
| `GetDbSystemHealthOverview:1` | Get DB system health |
| `GetDbSystems:1` | List Neo DB systems |
| `GetHanaCloudInstanceUpdateVersions:1` | Get available update versions |
| `GetNeoDbSystemSelfService:1` | Get Neo DB self-service ID |
| `GetNeoDbSystemUpdateVersions:1` | Get Neo DB update versions |
| `PollDbSystemUpdate:1` | Poll update status |
| `PollHanaCloudInstanceOperation:1` | Poll HANA Cloud operation |
| `PollRestartDbSystem:1` | Poll restart status |
| `RestartDbSystem:1` | Restart Neo DB system |
| `RestartHanaCloudInstance:1` | Restart HANA Cloud instance |
| `StartHanaCloudInstance:1` | Start HANA Cloud instance |
| `StopHanaCloudInstance:1` | Stop HANA Cloud instance |
| `TriggerRestartDbSystem:1` | Trigger async restart |
| `TriggerStartHanaCloudInstance:1` | Trigger async start |
| `TriggerStopHanaCloudInstance:1` | Trigger async stop |
| `TriggerUpdateDbSystem:1` | Trigger async update |
| `TriggerUpdateHanaCloudInstance:1` | Trigger async update |
| `UpdateDbSystem:1` | Update Neo DB system |
| `UpdateHanaCloudInstance:1` | Update HANA Cloud instance |
| `WaitDbSystemUpdate:1` | Wait for update completion |
| `WaitHanaCloudInstanceOperation:1` | Wait for HANA Cloud operation |

## dest-sapcp (Destination Service)

Destination management.

| Command | Description |
|---------|-------------|
| `CreateCfDestination:1` | Create destination |
| `DeleteCfDestination:1` | Delete destination |
| `GetCfDestination:1` | Get destination |
| `ListCfDestinations:1` | List destinations |
| `PatchCfDestination:1` | Update destination |

## dynatrace-sapcp (Dynatrace)

Dynatrace monitoring.

| Command | Description |
|---------|-------------|
| `GetDynatraceEvent:1` | Get specific event |
| `GetDynatraceEvents:1` | Get events |
| `GetDynatraceProblemDetails:1` | Get problem details |
| `GetDynatraceProblemsFeed:1` | Get problems feed |
| `GetDynatraceProblemsStatus:1` | Get problems overview |
| `PushDynatraceMetric:1` | Push custom metric |

## email-sapcp (Email)

Email operations.

| Command | Description |
|---------|-------------|
| `SendEmail:1` | Send email via SMTP |

### SendEmail:1 Key Options

- `host`: SMTP server host
- `port`: SMTP port
- `user`/`password`: Authentication
- `from`, `to`, `cc`, `bcc`: Recipients
- `subject`, `body`: Content
- `attachments`: File attachments
- `tls`: Enable TLS

## github-sapcp (GitHub)

GitHub operations.

| Command | Description |
|---------|-------------|
| `AddGitHubIssueComment:1` | Add comment to issue |
| `ChangeGitHubIssueStatus:1` | Change issue status |
| `CreateGitHubIssue:1` | Create issue |
| `CreateOrUpdateGitHubFile:1` | Create or update file |
| `DeleteGitHubFile:1` | Delete file |
| `GetGitHubFile:1` | Get file content |
| `GetGitHubIssue:1` | Get issue details |

## http-sapcp (HTTP Operations)

Execute HTTP requests with various authentication methods.

| Command | Description |
|---------|-------------|
| `GetTokenViaClientCredentials:1` | Get OAuth token via client credentials |
| `GetTokenViaPasswordCredentials:1` | Get OAuth token via password grant |
| `GetTokenViaRefreshToken:1` | Get OAuth token via refresh token |
| `HttpRequest:1` | Execute HTTP request with OAuth, Basic Auth, custom headers |
| `SensitiveHttpRequest:1` | HTTP request with sensitive response handling |

### HttpRequest Key Options

- **Authentication**: `user`/`password`, `clientId`/`clientSecret`, `authorizationHeader`, `refreshToken`
- **OAuth**: `tokenUrl` for OAuth flows
- **CSRF**: `csrfUrl` for CSRF-protected endpoints
- **TLS**: `trustedCerts`, `clientCert`, `trustAnyCert`
- **Cloud Connector**: `sccEnabled`, `sccLocationId`
- **Transform**: `responseBodyTransformer` for inline response transformation

## jenkins-sapcp (Jenkins)

Jenkins CI/CD operations.

| Command | Description |
|---------|-------------|
| `ExecuteJenkinsBuild:1` | Trigger and wait for completion |
| `PollJenkinsBuild:1` | Poll build status |
| `TriggerJenkinsBuild:1` | Trigger build |
| `WaitJenkinsBuild:1` | Wait for build completion |

## jira-sapcp (JIRA Integration)

JIRA issue management.

| Command | Description |
|---------|-------------|
| `AddJiraIssueComment:1` | Add comment to issue |
| `AddJiraIssueWatcher:1` | Add watcher to issue |
| `AddJiraIssueWebLink:1` | Add web link to issue |
| `ChangeJiraIssueStatus:1` | Change issue status/transition |
| `CloneJiraIssue:1` | Clone an issue |
| `CreateJiraIssue:1` | Create new issue |
| `ExecuteScriptAndAttachFileToJiraIssue:1` | Run script and attach output |
| `GetJiraIssue:1` | Get issue details |
| `LinkJiraIssues:1` | Link two issues |
| `ListJiraIssues:1` | Search issues with criteria |
| `RemoveJiraIssueWatcher:1` | Remove watcher from issue |
| `SetJiraIssueAssignee:1` | Set issue assignee |
| `UpdateJiraIssueAdditionalFields:1` | Update additional fields |
| `UpdateJiraIssueDescription:1` | Update issue description |

## kubernetes-sapcp (Kubernetes)

Kubernetes operations.

| Command | Description |
|---------|-------------|
| `ApplyK8sResource:1` | Create or update resource |
| `CreateK8sResource:1` | Create resource from manifest |
| `DeleteK8sResource:1` | Delete resource |
| `GetK8sResource:1` | Get resource details |
| `GetKubernetesNodeMetrics:1` | Get node metrics |
| `GetKubernetesNodesMetrics:1` | Get all nodes metrics |
| `GetKubernetesPodLog:1` | Get pod logs |
| `GetKubernetesPodMetrics:1` | Get pod metrics |
| `GetKubernetesPodsMetrics:1` | Get all pods metrics |
| `ListK8sResources:1` | List resources |
| `ListKubernetesResources:2` | List resources (v2) |
| `PatchK8sResource:1` | Patch resource partially |
| `PollRestartKubernetesDaemonset:1` | Poll daemonset restart status |
| `PollRestartKubernetesDeployment:1` | Poll deployment restart status |
| `PollRestartKubernetesStatefulset:1` | Poll statefulset restart status |
| `RestartKubernetesDaemonset:1` | Rolling restart daemonset |
| `RestartKubernetesDeployment:1` | Rolling restart deployment |
| `RestartKubernetesStatefulset:1` | Rolling restart statefulset |
| `TriggerDeleteKubernetesPod:1` | Delete specific pod |
| `TriggerDeleteKubernetesPods:1` | Delete multiple pods |
| `TriggerRestartKubernetesDaemonset:1` | Trigger async daemonset restart |
| `TriggerRestartKubernetesDeployment:1` | Trigger async deployment restart |
| `TriggerRestartKubernetesStatefulset:1` | Trigger async statefulset restart |
| `UpdateK8sResource:1` | Update existing resource |

## metadata-sapcp (Metadata)

Reusable input references.

| Input Reference | Description |
|-----------------|-------------|
| `CfRegionData:1` | CF region configurations (API URLs, UAA endpoints) |
| `NeoRegionData:1` | Neo environment configurations |

### CfRegionData Fields

- `cfApiUrl`: Cloud Foundry API endpoint
- `uaaTokenUrl`: UAA token endpoint
- `uaaAuthUrl`: UAA authorization endpoint
- `sccTunnel`: Cloud Connector tunnel endpoint

### Usage

```json
"values": [
  {
    "alias": "regionData",
    "valueFrom": {
      "inputReference": "metadata-sapcp:CfRegionData:1",
      "inputKey": "$(.execution.input.region)"
    }
  }
]
```

## monitoring-sapcp (Monitoring)

SAP monitoring operations.

| Command | Description |
|---------|-------------|
| `DbSystemHealthOverview:1` | Get DB system health |
| `GetCfAppEvents:1` | Get CF app events |
| `GetDbSystemMetrics:1` | Get DB system metrics |
| `GetJavaAppMetrics:1` | Get Java app metrics |
| `GetSimplifiedDbSystemCriticalMetrics:1` | Get critical DB metrics |
| `GetSimplifiedJavaAppCriticalMetrics:1` | Get critical Java app metrics |
| `JavaAppHealthOverview:1` | Get Java app health |

## scripts-sapcp (Script Execution)

Execute custom scripts. See **`automation-pilot-executor-executescript/SKILL.md`** for full parameter reference and examples.

| Command | Description |
|---------|-------------|
| `ExecuteScript:1` | Execute Bash script (legacy — `script` must be Base64 encoded) |
| `ExecuteScript:2` | Execute Bash script (recommended — plain text `script`) |
| `ExecutePythonScript:1` | Execute Python script (`script` = plain Python, `packages` = pip deps) |
| `ExecuteNodeJsScript:1` | Execute Node.js script (`script` = plain JS, `packages` = npm deps, `npmrc` for private registries) |
| `ExecutePowerShellScript:1` | Execute PowerShell script (`script` = plain PS, `modules` = PS module deps) |
| `SensitiveExecuteScript:2` | Bash script — output marked sensitive |
| `SensitiveExecutePythonScript:1` | Python script — output marked sensitive |
| `SensitiveExecuteNodeJsScript:1` | Node.js script — output marked sensitive |
| `SensitiveExecutePowerShellScript:1` | PowerShell script — output marked sensitive |

All script commands share these common inputs: `script` (required), `parameters`, `stdin` (sensitive), `environment`, `timeout` (15–600s), `successExitCodes`.  
Output keys: `exitCode` (number), `result` (array of output lines, last 64 KB).

## sm-sapcp (Service Manager)

Service Manager operations.

| Command | Description |
|---------|-------------|
| `CreateServiceBinding:1` | Create binding |
| `CreateServiceInstance:1` | Create service instance |
| `DeleteServiceBinding:1` | Delete binding |
| `DeleteServiceInstance:1` | Delete service instance |
| `GetServiceBinding:1` | Get binding details |
| `GetServiceDetails:1` | Get service offering and plan IDs |
| `GetServiceInstance:1` | Get service instance details |
| `GetServiceInstanceParameters:1` | Get instance parameters |
| `ListServiceBindings:1` | List bindings |
| `ListServiceInstances:1` | List all service instances |
| `TriggerCreateServiceInstance:1` | Trigger async instance creation |
| `TriggerDeleteServiceInstance:1` | Trigger async instance deletion |
| `TriggerUpdateServiceInstance:1` | Trigger async instance update |
| `UpdateServiceInstance:1` | Update service instance |
| `WaitServiceBindingOperation:1` | Wait for binding operation |
| `WaitServiceInstanceOperation:1` | Wait for instance operation |

## sql-sapcp (SQL)

SQL operations.

| Command | Description |
|---------|-------------|
| `ExecuteHanaCloudSqlStatement:1` | Execute SQL on HANA Cloud |
| `SensitiveExecuteHanaCloudSqlStatement:1` | Execute SQL with sensitive handling |

## utils-sapcp (Utilities)

General purpose utilities for flow control.

| Command | Description |
|---------|-------------|
| `AwaitExecution:1` | Wait for execution to complete |
| `Delay:1` | Wait for specified duration (minutes) |
| `ForEach:2` | Iterate over array, execute command for each item |
| `GenerateOneTimePasscode:1` | Generate OTP |
| `GetExecutionInput:1` | Retrieve execution input |
| `GetExecutionOutput:1` | Retrieve execution output |
| `GetExecutionStatus:1` | Get current execution status |
| `GetUserChoice:1` | Pause for user selection (INPUT_REQUIRED state) |
| `GetUserInput:1` | Pause for user text input (INPUT_REQUIRED state) |
| `ObtainLock:1` | Acquire distributed lock |
| `ReleaseLock:1` | Release distributed lock |
| `SensitiveForEach:2` | ForEach with sensitive data handling |
| `SensitiveVoid:2` | Void with sensitive data handling |
| `TriggerExecution:1` | Trigger a command execution |
| `Void:1` | No-op command for data transformation/validation |

### Delay:1 Usage

Pauses execution for a fixed number of minutes (1–10080). Unlike `initialDelay` or `repeat.delay` (which are executor-level properties), `Delay:1` is a standalone executor step.

```json
{
  "execute": "utils-sapcp:Delay:1",
  "alias": "wait",
  "input": {
    "minutes": "5"
  },
  "description": null,
  "progressMessage": "Waiting 5 minutes...",
  "initialDelay": null,
  "pause": null,
  "when": null,
  "validate": null,
  "autoRetry": null,
  "repeat": null,
  "errorMessages": [],
  "dryRun": null
}
```

### ForEach:2 Key Options

- `command`: Command ID to execute (use `<<<TENANT_ID>>>` for your commands)
- `inputs`: Array of input objects
- `defaultValues`: Default values merged into each input
- `batchSize`: Number of parallel executions (1-20)

## xsuaa-sapcp (Authorization)

XSUAA and role collection management.

| Command | Description |
|---------|-------------|
| `AssignRoleCollections:1` | Assign role collections to user/group |
| `CreateRoleCollection:1` | Create role collection |
| `DeleteRoleCollection:1` | Delete role collection |
| `GetRoleCollection:1` | Get role collection details |
| `GetSubAccountUser:1` | Get subaccount user |
| `ListRoleCollections:1` | List all role collections |
| `ListSubAccountUsers:1` | List subaccount users |
| `RemoveSubAccountUser:1` | Remove user from subaccount |
| `UnassignRoleCollections:1` | Unassign role collections |

## Tags

Commands use tags for metadata:

| Tag | Description |
|-----|-------------|
| `env` | Environment type (cf, k8s, neo) |
| `autopi:deprecated` | Command is deprecated |
| `autopi:replacement` | Has a replacement command |
| `autopi:internal` | Internal use only |
| `autopi:released` | Production ready |
