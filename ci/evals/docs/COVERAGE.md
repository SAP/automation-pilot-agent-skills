# Case coverage

This table describes the 160 independent tasks. The full campaign additionally includes the complete [E2E acceptance workflow](E2E_COVERAGE.md).


All 160 cases have [current reference/grader calibration](CALIBRATION.md).
Independent review and a fresh candidate campaign remain pending; the bank is provisional.
Counts refer to tasks; partitions are not additional cases.

Preferred presets: **smoketest 12**, **minimal 20**, **full 160**. Smoketest covers
each primary skill once. Minimal uses the current pilot membership; full includes
every row below. Legacy smoke still has 40 cases.

| Case | Primary skill | Tier | Mode | Split | Subsets |
|---|---|---|---|---|---|
| binding-count-zero | dynamic-expressions | engine | expression | development | smoke |
| block-mcp-incompatible-binding | mcp-servers | fixture | tenant-update | holdout |  |
| btp-dynamic-region-metadata | dynamic-expressions | engine | expression | holdout |  |
| btp-entitlement-read-wrapper | command-authoring | local | local-authoring | development |  |
| btp-quota-zero-preservation | dynamic-expressions | engine | expression | development | smoke |
| btp-subscription-polling | command-authoring | local | review | holdout |  |
| catalog-cascade-impact-plan | catalogs | local | review | development |  |
| catalog-delete-owned-empty | catalogs | fixture | tenant-update | development |  |
| catalog-provider-ownership | catalogs | local | review | development |  |
| catalog-purposeful-reuse | catalogs | local | review | development |  |
| catalog-technical-name-rules | catalogs | local | review | development |  |
| catalog-update-preserve-metadata | catalogs | fixture | tenant-update | development | smoketest, minimal/pilot, smoke |
| cf-command-description-preservation | command-authoring | fixture | tenant-update | development |  |
| cf-health-empty-is-false | dynamic-expressions | engine | expression | development | smoke |
| cf-memory-fraction | dynamic-expressions | local | review | development | smoke |
| cf-query-encoding | dynamic-expressions | engine | expression | development | smoke |
| cf-resolution-ambiguity | command-authoring | local | review | holdout |  |
| cf-rolling-availability-guard | command-authoring | local | review | development |  |
| cf-route-relationship | dynamic-expressions | engine | expression | holdout |  |
| cf-runtime-state-wrapper | command-authoring | local | local-authoring | development | smoke |
| cf-start-waits-for-health | command-authoring | local | review | development |  |
| cf-stop-permanent-error-retry | command-authoring | local | review | development |  |
| checksum-recovery | cli | local | review | development | minimal/pilot, smoke |
| cli-typed-request-preview | cli | local | local-authoring | development |  |
| connection-authentication-failure | connection | fixture | connection | development |  |
| connection-preflight-without-credentials | connection | fixture | connection | holdout |  |
| connection-read-only-permission | connection | fixture | connection | development |  |
| connection-tenant-mismatch | connection | fixture | connection | development | smoketest, minimal/pilot, smoke |
| continuity | cli | local | review | holdout |  |
| destination-read-wrapper | command-authoring | local | local-authoring | holdout |  |
| destination-secret-response | command-authoring | local | review | development |  |
| diagnose-cf-ambiguous-target | executions | fixture | execution-diagnosis | holdout |  |
| diagnose-cf-single-instance-guard | executions | fixture | execution-diagnosis | development |  |
| diagnose-hana-binding-protection | executions | fixture | execution-diagnosis | holdout |  |
| diagnose-hana-concurrent-binding | executions | fixture | execution-diagnosis | development |  |
| diagnose-hana-successful-noop | executions | fixture | execution-diagnosis | development |  |
| diagnose-http-retry-exhaustion | executions | fixture | execution-diagnosis | holdout |  |
| diagnose-http-untrusted-body | executions | fixture | execution-diagnosis | holdout |  |
| diagnose-iflow-partial-promotion | executions | fixture | execution-diagnosis | development |  |
| diagnose-jenkins-unstable-build | executions | fixture | execution-diagnosis | development |  |
| diagnose-k8s-partial-rollout | executions | fixture | execution-diagnosis | holdout |  |
| diagnose-k8s-scope-404 | executions | fixture | execution-diagnosis | development |  |
| diagnose-sm-operation-failure | executions | fixture | execution-diagnosis | development |  |
| diagnose-sm-write-only-output | executions | fixture | execution-diagnosis | development |  |
| disable-one-mcp-tool | mcp-servers | fixture | tenant-update | development |  |
| discover-btp-absolute-entitlement | discovery | fixture | discovery | development |  |
| discover-btp-managed-sm-binding | discovery | fixture | discovery | development |  |
| discover-cf-sensitive-environment | discovery | fixture | discovery | holdout |  |
| discover-cf-start-completion | discovery | fixture | discovery | development | smoketest, minimal/pilot, smoke |
| discover-cf-task-existing | discovery | fixture | discovery | development |  |
| discover-destination-relative-http | discovery | fixture | discovery | holdout |  |
| discover-hana-managed-binding | discovery | fixture | discovery | development |  |
| discover-hana-noop-upgrade | discovery | fixture | discovery | development |  |
| discover-hana-point-recovery | discovery | fixture | discovery | development |  |
| discover-hana-rotate-snapshot | discovery | fixture | discovery | development |  |
| discover-http-sensitive-response | discovery | fixture | discovery | development |  |
| discover-iflow-optional-promotion | discovery | fixture | discovery | development |  |
| discover-jenkins-existing-build-wait | discovery | fixture | discovery | development |  |
| discover-k8s-deployment-wait | discovery | fixture | discovery | holdout |  |
| discover-k8s-resource-list | discovery | fixture | discovery | development |  |
| discover-sm-write-only-capability | discovery | fixture | discovery | development |  |
| eligible-track-filter | dynamic-expressions | engine | expression | development | smoke |
| github-command-correlation-tag | command-authoring | fixture | tenant-update | development |  |
| github-config-write-wrapper | command-authoring | local | local-authoring | development |  |
| github-unicode-base64 | dynamic-expressions | engine | expression | development | smoke |
| github-update-sha | command-authoring | local | review | development |  |
| hana-binding-concurrent-winner | command-authoring | local | review | development |  |
| hana-bindings-block-delete | command-authoring | local | review | holdout |  |
| hana-command-deadline-default | command-authoring | fixture | tenant-update | development |  |
| hana-noop-side-effect | command-authoring | local | review | development | minimal/pilot, smoke |
| hana-snapshot-list-wrapper | command-authoring | local | local-authoring | development | smoke |
| hana-snapshot-oldest-order | command-authoring | local | review | development |  |
| hana-sql-binding-sensitivity | command-authoring | local | review | development |  |
| hana-upgrade-report-wrapper | command-authoring | local | local-authoring | development | smoke |
| hana-upgrade-snapshot-opt-in | command-authoring | local | review | development |  |
| http-auth-sensitive-input | command-authoring | local | review | holdout |  |
| http-case-insensitive-header | dynamic-expressions | engine | expression | development | smoke |
| http-command-timeout-default | command-authoring | fixture | tenant-update | development |  |
| http-json-escaping | dynamic-expressions | engine | expression | development | smoke |
| http-json-get-wrapper | command-authoring | local | local-authoring | development |  |
| http-oauth-sensitive-token | command-authoring | local | review | development |  |
| http-stream-alternative | dynamic-expressions | engine | expression | holdout |  |
| http-timeout-success-policy | command-authoring | local | review | development | smoketest, minimal/pilot, smoke |
| iflow-empty-result-fallback | command-authoring | local | review | development | minimal/pilot, smoke |
| iflow-promotion-wrapper | command-authoring | local | local-authoring | development |  |
| iflow-restart-completion | command-authoring | local | review | development |  |
| iflow-skipped-array-output | dynamic-expressions | engine | expression | development | smoke |
| input-cohesive-sensitive-contract | inputs | local | local-authoring | development |  |
| input-description-merge | inputs | fixture | tenant-update | holdout |  |
| input-endpoint-value-patch | inputs | fixture | tenant-update | development |  |
| input-explanation-hidden-values | inputs | local | review | development |  |
| input-full-update-masked-placeholder | inputs | local | review | holdout |  |
| input-key-removal-dependencies | inputs | local | review | development |  |
| input-kubeconfig-object-boundary | inputs | local | review | development |  |
| input-null-value-patch | inputs | local | review | development |  |
| input-overlapping-binding-review | inputs | local | review | development |  |
| input-regular-tag-merge | inputs | fixture | tenant-update | holdout |  |
| input-rotation-plan-omitted-values | inputs | local | review | holdout |  |
| input-secret-value-rotation | inputs | fixture | tenant-update | development | smoketest, minimal/pilot, smoke |
| jira-clone-result-path | command-authoring | local | review | development |  |
| jira-field-false-preservation | dynamic-expressions | engine | expression | holdout |  |
| k8s-apply-existing-resource | command-authoring | local | review | development |  |
| k8s-auth-secret-contract | command-authoring | local | review | holdout |  |
| k8s-command-status-description | command-authoring | fixture | tenant-update | holdout |  |
| k8s-millicore-conversion | dynamic-expressions | engine | expression | development | smoke |
| k8s-namespaced-selection | dynamic-expressions | engine | expression | development | smoke |
| k8s-resource-read-wrapper | command-authoring | local | local-authoring | development |  |
| k8s-rollout-conjunction | dynamic-expressions | engine | expression | holdout |  |
| k8s-rollout-ready-conjunction | command-authoring | local | review | holdout |  |
| local-authoring | cli | local | local-authoring | development | smoketest, minimal/pilot, smoke |
| logging-complete-result-path | command-authoring | local | review | holdout |  |
| logging-millisecond-normalization | dynamic-expressions | engine | expression | holdout |  |
| managed-binding-fallback | dynamic-expressions | engine | expression | development | smoke |
| mcp-advisory-hints-safety | mcp-servers | local | review | holdout |  |
| mcp-business-only-schema | mcp-servers | local | local-authoring | holdout |  |
| mcp-full-array-replacement | mcp-servers | local | review | development |  |
| mcp-immutable-name-migration | mcp-servers | local | review | development |  |
| mcp-jsonrpc-result-validation | command-authoring | local | review | development | minimal/pilot, smoke |
| mcp-session-cleanup-listeners | command-authoring | local | review | development | minimal/pilot, smoke |
| mcp-sync-timeout-fallback | mcp-servers | local | review | development |  |
| mcp-unbound-secret-rejection | mcp-servers | local | review | development |  |
| native-iterator-wrapper | command-authoring | engine | local-authoring | development | minimal/pilot, smoke |
| oldest-snapshot-selection | dynamic-expressions | engine | expression | development | smoke |
| plan-delegated-native-design | plan | local | review | holdout |  |
| plan-existing-authorized-target | plan | local | review | holdout |  |
| plan-finite-schedule-window | plan | local | review | development |  |
| plan-missing-nested-path-evidence | plan | local | review | holdout |  |
| plan-native-script-gap | plan | local | review | development |  |
| plan-no-downtime-quota-denied | plan | local | review | development |  |
| plan-partial-effects-compensation | plan | local | review | holdout |  |
| plan-protected-resource-exclusions | plan | local | review | development |  |
| plan-single-instance-availability | plan | local | review | development | minimal/pilot, smoke |
| preserve-policy-false | dynamic-expressions | engine | expression | development | smoketest, minimal/pilot, smoke |
| proposal-scope | plan | local | review | development | smoketest, minimal/pilot, smoke |
| publish-mcp-bound-tool | mcp-servers | fixture | tenant-update | development | smoketest, minimal/pilot, smoke |
| recover-cf-accepted-run | executions | fixture | execution-diagnosis | development | smoketest, minimal/pilot, smoke |
| schedule-day31-month-end | scheduling | local | review | development |  |
| schedule-disable-preserve-body | scheduling | fixture | tenant-update | development |  |
| schedule-dst-nonexistent-time | scheduling | local | review | development |  |
| schedule-finite-count-not-native | scheduling | local | review | development |  |
| schedule-once-full-calendar | scheduling | local | review | holdout |  |
| schedule-paired-weekday-times | scheduling | local | local-authoring | development | smoketest, minimal/pilot, smoke |
| schedule-recurring-overlap | scheduling | local | review | development |  |
| schedule-timezone-conditional-update | scheduling | fixture | tenant-update | development |  |
| semantic-upgrade-maximum | dynamic-expressions | engine | expression | development | smoke |
| sm-command-parameter-description | command-authoring | fixture | tenant-update | development |  |
| sm-create-ready-wrapper | command-authoring | local | local-authoring | development |  |
| sm-operation-errors | dynamic-expressions | engine | expression | development | smoke |
| sm-operation-polling | command-authoring | local | review | development |  |
| sm-serialized-parameters | dynamic-expressions | engine | expression | development | smoke |
| sm-write-only-parameters | command-authoring | local | review | development | minimal/pilot, smoke |
| tag-feature-registry-review | tags | local | review | development |  |
| tag-one-run-priority-logging | tags | fixture | execution | development | smoketest, minimal/pilot, smoke |
| tag-override-inherited-priority | tags | fixture | execution | development |  |
| tag-regular-correlation-only | tags | fixture | execution | holdout |  |
| tag-retention-boundary-review | tags | local | review | holdout |  |
| tag-run-retention-delete-zero | tags | fixture | execution | development |  |
| transport-import-wrapper | command-authoring | local | local-authoring | development |  |
| transport-request-wire-array | command-authoring | local | review | holdout |  |
| update-mcp-instructions-preserve-tools | mcp-servers | fixture | tenant-update | development |  |
