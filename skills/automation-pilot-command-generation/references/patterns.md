# Common Command Patterns

This reference covers common patterns for building SAP Automation Pilot commands.

## IMPORTANT: Naming Conventions

- **Generated commands**: Use `<<<TENANT_ID>>>` suffix (e.g., `mycommands-<<<TENANT_ID>>>:MyCommand:1`)
- **Built-in SAP commands**: Use `sapcp` suffix (e.g., `http-sapcp:HttpRequest:1`)
- **ForEach**: Always use version 2 (`utils-sapcp:ForEach:2`), never version 1

## Pattern 1: Simple HTTP Request with Validation

Basic pattern for making an authenticated API call with error handling.

```json
{
  "id": "mycommands-<<<TENANT_ID>>>:GetResource:1",
  "catalog": "mycommands-<<<TENANT_ID>>>",
  "name": "GetResource",
  "description": "Retrieves a resource from the API",
  "version": 1,
  "inputKeys": {
    "region": {
      "type": "string",
      "required": true,
      "description": "Cloud Foundry region"
    },
    "user": {
      "type": "string",
      "required": true,
      "description": "User for authentication"
    },
    "password": {
      "type": "string",
      "required": true,
      "sensitive": true,
      "description": "Password for authentication"
    },
    "resourceId": {
      "type": "string",
      "required": true,
      "description": "ID of the resource to retrieve"
    }
  },
  "outputKeys": {
    "resource": {
      "type": "object",
      "description": "The retrieved resource"
    }
  },
  "configuration": {
    "values": [
      {
        "alias": "regionData",
        "valueFrom": {
          "inputReference": "metadata-sapcp:CfRegionData:1",
          "inputKey": "$(.execution.input.region)"
        }
      }
    ],
    "output": {
      "resource": "$(.getResource.output.body | toObject)"
    },
    "executors": [
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "getResource",
        "input": {
          "url": "$(.regionData.cfApiUrl)/v3/resources/$(.execution.input.resourceId | toUrlEncoded)",
          "method": "GET",
          "headers": "{\"Accept\": \"application/json\"}",
          "user": "$(.execution.input.user)",
          "password": "$(.execution.input.password)",
          "tokenUrl": "$(.regionData.uaaTokenUrl)",
          "clientId": "cf",
          "timeout": "10"
        },
        "validate": {
          "semantic": "OR",
          "conditions": [
            {
              "semantic": "OR",
              "cases": [
                {
                  "expression": "$(.getResource.output.status)",
                  "operator": "EQUALS",
                  "semantic": "OR",
                  "values": ["200"]
                }
              ]
            }
          ]
        },
        "autoRetry": {
          "maxCount": 3,
          "delay": "5s",
          "logic": "INCREMENTAL",
          "applyOnValidation": false,
          "when": {
            "semantic": "OR",
            "conditions": [
              {
                "semantic": "OR",
                "cases": [
                  {
                    "expression": "$([408, 429, 500, 502, 503, 504, -1] | filter(. == $.getResource.output.status) | length)",
                    "operator": "EQUALS",
                    "semantic": "OR",
                    "values": ["1"]
                  }
                ]
              }
            ]
          }
        },
        "errorMessages": [
          {
            "message": "Failed to get resource: $(.getResource.output.body)",
            "when": {
              "semantic": "OR",
              "conditions": [
                {
                  "semantic": "OR",
                  "cases": [
                    {
                      "expression": "$(.getResource.output.status)",
                      "operator": "NOT_EQUALS",
                      "semantic": "OR",
                      "values": ["200"]
                    }
                  ]
                }
              ]
            }
          }
        ]
      }
    ],
    "listeners": []
  }
}
```

## Pattern 2: Polling Until Complete

Wait for an async operation to complete by polling.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "checkStatus",
  "input": {
    "url": "$(.regionData.apiUrl)/v3/jobs/$(.execution.input.jobId)",
    "method": "GET",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)",
    "tokenUrl": "$(.regionData.uaaTokenUrl)",
    "timeout": "20"
  },
  "validate": {
    "semantic": "OR",
    "conditions": [
      {
        "semantic": "OR",
        "cases": [
          {
            "expression": "$(.checkStatus.output.body | toObject.state)",
            "operator": "NOT_EQUALS",
            "semantic": "OR",
            "values": ["FAILED"]
          }
        ]
      }
    ]
  },
  "repeat": {
    "maxCount": 120,
    "delay": "10s",
    "failOnMaxCount": true,
    "until": {
      "semantic": "OR",
      "conditions": [
        {
          "semantic": "OR",
          "cases": [
            {
              "expression": "$(.checkStatus.output.body | toObject.state)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["COMPLETE", "SUCCEEDED"]
            }
          ]
        }
      ]
    }
  },
  "errorMessages": [
    {
      "message": "Job failed: $(.checkStatus.output.body | toObject.errors | map(.detail) | join(\", \"))",
      "when": {
        "semantic": "OR",
        "conditions": [
          {
            "semantic": "OR",
            "cases": [
              {
                "expression": "$(.checkStatus.output.body | toObject.state)",
                "operator": "EQUALS",
                "semantic": "OR",
                "values": ["FAILED"]
              }
            ]
          }
        ]
      }
    }
  ]
}
```

## Pattern 3: Conditional Step Execution

Execute step only when a condition is met.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "optionalStep",
  "input": {
    "url": "$(.regionData.apiUrl)/v3/optional-action",
    "method": "POST",
    "body": "$(.execution.input.data)"
  },
  "when": {
    "semantic": "OR",
    "conditions": [
      {
        "semantic": "AND",
        "cases": [
          {
            "expression": "$(.execution.input.enableOptional)",
            "operator": "EQUALS",
            "semantic": "OR",
            "values": ["true"]
          },
          {
            "expression": "$(.previousStep.output.body | toObject.requiresAction)",
            "operator": "EQUALS",
            "semantic": "OR",
            "values": ["true"]
          }
        ]
      }
    ]
  }
}
```

## Pattern 4: Multi-Step Workflow with Dependencies

Chain steps where each depends on the previous.

```json
{
  "configuration": {
    "values": [
      {
        "alias": "regionData",
        "valueFrom": {
          "inputReference": "metadata-sapcp:CfRegionData:1",
          "inputKey": "$(.execution.input.region)"
        }
      }
    ],
    "output": {
      "result": "$(.finalStep.output.body | toObject)"
    },
    "executors": [
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "step1",
        "input": {
          "url": "$(.regionData.cfApiUrl)/v3/first",
          "method": "GET"
        }
      },
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "step2",
        "input": {
          "url": "$(.regionData.cfApiUrl)/v3/second/$(.step1.output.body | toObject.id)",
          "method": "GET"
        }
      },
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "finalStep",
        "input": {
          "url": "$(.regionData.cfApiUrl)/v3/final",
          "method": "POST",
          "body": "$({\"step1Id\": .step1.output.body | toObject.id, \"step2Data\": .step2.output.body | toObject} | toMinifiedJsonString)"
        }
      }
    ]
  }
}
```

## Pattern 5: Batch Processing with ForEach

Process multiple items using ForEach. **Always use ForEach:2, never ForEach:1.**

```json
{
  "configuration": {
    "output": {
      "results": "$(.processItems.output.outputs)"
    },
    "executors": [
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "getItems",
        "input": {
          "url": "$(.execution.input.apiUrl)/items",
          "method": "GET"
        }
      },
      {
        "execute": "utils-sapcp:ForEach:2",
        "alias": "processItems",
        "input": {
          "command": "mycommands-<<<TENANT_ID>>>:ProcessSingleItem:1",
          "inputs": "$(.getItems.output.body | toObject.items | map({itemId: .id, name: .name}))",
          "defaultValues": "{\"apiUrl\": \"$(.execution.input.apiUrl)\"}",
          "batchSize": "5"
        }
      }
    ]
  }
}
```

## Pattern 6: ID Resolution with Caching

Resolve names to IDs only when needed.

```json
{
  "configuration": {
    "output": {
      "resolvedId": "$(if .execution.input.identifier | isGuid then .execution.input.identifier else .resolveId.output.body | toObject.resources[0].guid end)"
    },
    "executors": [
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "resolveId",
        "input": {
          "url": "$(.regionData.apiUrl)/v3/resources?names=$(.execution.input.identifier | toUrlEncoded)",
          "method": "GET"
        },
        "when": {
          "semantic": "OR",
          "conditions": [
            {
              "semantic": "OR",
              "cases": [
                {
                  "expression": "$(.execution.input.identifier | isGuid)",
                  "operator": "EQUALS",
                  "semantic": "OR",
                  "values": ["false"]
                }
              ]
            }
          ]
        },
        "validate": {
          "semantic": "OR",
          "conditions": [
            {
              "semantic": "OR",
              "cases": [
                {
                  "expression": "$(.resolveId.output.body | toObject.pagination.total_results)",
                  "operator": "EQUALS",
                  "semantic": "OR",
                  "values": ["1"]
                }
              ]
            }
          ]
        }
      }
    ]
  }
}
```

## Pattern 7: Timeout with Graceful Handling

Implement custom timeout logic.

```json
{
  "configuration": {
    "executors": [
      {
        "execute": "utils-sapcp:Void:1",
        "alias": "startTime",
        "input": {
          "message": "$(now)"
        }
      },
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "pollStatus",
        "input": {
          "url": "$(.execution.input.statusUrl)",
          "method": "GET"
        },
        "validate": {
          "semantic": "OR",
          "conditions": [
            {
              "semantic": "AND",
              "cases": [
                {
                  "expression": "$(.pollStatus.output.body | toObject.state)",
                  "operator": "NOT_EQUALS",
                  "semantic": "OR",
                  "values": ["FAILED"]
                },
                {
                  "expression": "$(.execution.input.timeoutMinutes * 60 - (now - (.startTime.output.message | toNumber)) < 15)",
                  "operator": "NOT_EQUALS",
                  "semantic": "OR",
                  "values": ["true"]
                }
              ]
            }
          ]
        },
        "repeat": {
          "maxCount": 1000,
          "delay": "15s",
          "failOnMaxCount": false,
          "until": {
            "semantic": "OR",
            "conditions": [
              {
                "semantic": "OR",
                "cases": [
                  {
                    "expression": "$(.pollStatus.output.body | toObject.state)",
                    "operator": "EQUALS",
                    "semantic": "OR",
                    "values": ["COMPLETE"]
                  }
                ]
              }
            ]
          }
        },
        "errorMessages": [
          {
            "message": "Timeout exceeded: $(.execution.input.timeoutMinutes) minutes",
            "when": {
              "semantic": "OR",
              "conditions": [
                {
                  "semantic": "AND",
                  "cases": [
                    {
                      "expression": "$(.pollStatus.output.body | toObject.state)",
                      "operator": "NOT_EQUALS",
                      "semantic": "OR",
                      "values": ["COMPLETE"]
                    },
                    {
                      "expression": "$(.execution.input.timeoutMinutes * 60 - (now - (.startTime.output.message | toNumber)) < 15)",
                      "operator": "EQUALS",
                      "semantic": "OR",
                      "values": ["true"]
                    }
                  ]
                }
              ]
            }
          }
        ]
      }
    ]
  }
}
```

## Pattern 8: CSRF Token Handling

Make requests that require CSRF tokens.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "postWithCsrf",
  "input": {
    "url": "$(.execution.input.targetUrl)",
    "method": "POST",
    "headers": "{\"Content-Type\": \"application/json\"}",
    "body": "$(.execution.input.payload)",
    "csrfUrl": "$(.execution.input.csrfUrl)",
    "user": "$(.execution.input.user)",
    "password": "$(.execution.input.password)"
  }
}
```

## Pattern 9: Response Transformation

Transform response body inline.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "getAndTransform",
  "input": {
    "url": "$(.regionData.apiUrl)/v3/items",
    "method": "GET",
    "responseBodyTransformer": "toObject.resources | map({id: .guid, name: .name, state: .state})"
  }
}
```

## Pattern 10: Dynamic Progress Messages

Provide user feedback during long operations.

```json
{
  "execute": "http-sapcp:HttpRequest:1",
  "alias": "longOperation",
  "input": {
    "url": "$(.execution.input.statusUrl)",
    "method": "GET"
  },
  "progressMessage": "Status: $(.longOperation.output.body | toObject.state) - Progress: $(.longOperation.output.body | toObject.progress)%",
  "repeat": {
    "maxCount": 100,
    "delay": "10s",
    "until": {
      "semantic": "OR",
      "conditions": [
        {
          "semantic": "OR",
          "cases": [
            {
              "expression": "$(.longOperation.output.body | toObject.state)",
              "operator": "EQUALS",
              "semantic": "OR",
              "values": ["COMPLETE"]
            }
          ]
        }
      ]
    }
  }
}
```

## Pattern 11: Sensitive Data Handling

Handle credentials and sensitive responses.

```json
{
  "inputKeys": {
    "clientSecret": {
      "type": "string",
      "required": true,
      "sensitive": true,
      "description": "OAuth client secret"
    }
  },
  "outputKeys": {
    "credentials": {
      "type": "object",
      "sensitive": true,
      "description": "Service credentials"
    }
  },
  "configuration": {
    "executors": [
      {
        "execute": "http-sapcp:SensitiveHttpRequest:1",
        "alias": "getCredentials",
        "input": {
          "url": "$(.execution.input.credentialsUrl)",
          "method": "GET",
          "omitBodyInErrorMessage": "true"
        }
      }
    ]
  }
}
```

## Pattern 12: Parallel Execution Preparation

Prepare inputs for parallel ForEach:2 processing.

```json
{
  "configuration": {
    "output": {
      "processedItems": "$(.processAll.output.outputs | map(select(.status == \"success\")))"
    },
    "executors": [
      {
        "execute": "http-sapcp:HttpRequest:1",
        "alias": "listItems",
        "input": {
          "url": "$(.execution.input.apiUrl)/items",
          "method": "GET"
        }
      },
      {
        "execute": "utils-sapcp:Void:1",
        "alias": "prepareInputs",
        "input": {
          "message": "$(.listItems.output.body | toObject.items | map({itemId: .id, config: $.execution.input.config}) | toMinifiedJsonString)"
        }
      },
      {
        "execute": "utils-sapcp:ForEach:2",
        "alias": "processAll",
        "input": {
          "command": "mycommands-<<<TENANT_ID>>>:ProcessItem:1",
          "inputs": "$(.prepareInputs.output.message | toArray)",
          "batchSize": "10"
        }
      }
    ]
  }
}
```
