from __future__ import annotations

import json
import os
from functools import lru_cache
from typing import Annotated, Any
from urllib.parse import quote

import httpx
from fastmcp import FastMCP
from pydantic import Field

mcp = FastMCP("SAP Automation Pilot MCP")


@lru_cache(maxsize=1)
def _load_config() -> tuple[str, str]:
    hostname = os.environ.get("AUTOMATION_PILOT_HOSTNAME", "").strip().rstrip("/")
    token = os.environ.get("AUTOMATION_PILOT_TOKEN", "").strip()

    if not hostname:
        raise RuntimeError(
            "Automation Pilot hostname is not configured. "
            "Set AUTOMATION_PILOT_HOSTNAME."
        )
    if not token:
        raise RuntimeError(
            "Automation Pilot token is not configured. "
            "Set AUTOMATION_PILOT_TOKEN."
        )

    base_url = f"https://{hostname}/api/v1"

    return base_url, token


@lru_cache(maxsize=1)
def _client() -> httpx.Client:
    base_url, token = _load_config()
    return httpx.Client(
        base_url=base_url,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
        },
        timeout=30.0,
        follow_redirects=False,
    )


def _request(
    method: str,
    path: str,
    *,
    params: dict[str, Any] | None = None,
    body: dict[str, Any] | None = None,
    headers: dict[str, str] | None = None,
    omit_response_fields: frozenset[str] = frozenset(),
    suppress_error_payload: bool = False,
) -> Any:
    try:
        if params is not None:
            params = {key: value for key, value in params.items() if value is not None}
        response = _client().request(
            method, path, params=params, json=body, headers=headers
        )
    except httpx.RequestError as exc:
        raise RuntimeError(f"Automation Pilot request failed: {exc}") from exc

    if response.content:
        try:
            payload: Any = response.json()
        except ValueError:
            payload = response.text
    else:
        payload = None

    if omit_response_fields:
        payload = _without_fields(payload, omit_response_fields)

    if response.is_error:
        if suppress_error_payload:
            raise RuntimeError(
                f"Automation Pilot returned HTTP {response.status_code}; "
                "response omitted for sensitive Input write."
            )
        raise RuntimeError(
            f"Automation Pilot returned HTTP {response.status_code}: "
            f"{json.dumps(payload, ensure_ascii=False)}"
        )

    return payload


def _without_input_values(input_definition: dict[str, Any]) -> dict[str, Any]:
    """Return Input metadata without the stored values."""
    return {key: value for key, value in input_definition.items() if key != "values"}


def _without_fields(payload: Any, fields: frozenset[str]) -> Any:
    """Recursively remove sensitive response fields before returning or raising."""
    if isinstance(payload, dict):
        return {
            key: _without_fields(value, fields)
            for key, value in payload.items()
            if key not in fields
        }
    if isinstance(payload, list):
        return [_without_fields(value, fields) for value in payload]
    return payload


_SUMMARY_KEYS = ("id", "catalog", "name", "description", "tags")


def _summary(definition: dict[str, Any]) -> dict[str, Any]:
    """Return only the general identifying fields of a command or Input.

    Keeps id, catalog, name, description, and tags so a listing stays cheap;
    callers read the full contract or definition with the dedicated get_* tools.
    """
    return {key: definition[key] for key in _SUMMARY_KEYS if key in definition}


def _without_configuration(command: dict[str, Any]) -> dict[str, Any]:
    """Return a command definition without its implementation.

    Drops the composite `configuration` (executors, expressions, control
    blocks) while keeping the caller-facing contract: identifying fields,
    version, inputKeys, outputKeys, and tags.
    """
    return {key: value for key, value in command.items() if key != "configuration"}


@mcp.tool
def get_connection_info() -> dict[str, Any]:
    """Validate the Automation Pilot connection and return a non-secret account summary.

    The summary includes the application URL, tenant and identity details, permissions,
    and service plan. The configured token is never returned.
    """
    server_info = _request("GET", "/server-info")
    user_info = _request("GET", "/user-information")
    tenant_info = _request("GET", "/tenant")

    return {
        "connected": True,
        "url": server_info.get("autoPiTenantUrl"),
        "tenantId": tenant_info.get("id"),
        "username": user_info.get("username"),
        "permissions": user_info.get("permissions"),
        "plan": user_info.get("appPlan"),
    }


@mcp.tool
def list_scheduled_executions() -> list[dict[str, Any]]:
    """List all scheduled executions owned by the current Automation Pilot tenant."""
    return _request("GET", "/scheduled-executions")


@mcp.tool
def get_scheduled_execution(
    scheduled_execution_id: Annotated[
        str,
        Field(description="ID of the scheduled execution to retrieve."),
    ],
) -> dict[str, Any]:
    """Get detailed information for one scheduled execution."""
    resource_id = quote(scheduled_execution_id, safe="")
    return _request("GET", f"/scheduled-executions/{resource_id}")


@mcp.tool
def create_scheduled_execution(
    scheduled_execution: Annotated[
        dict[str, Any],
        Field(description="Complete Automation Pilot scheduled-execution definition."),
    ],
) -> dict[str, Any]:
    """Create a scheduled execution. Pass the Automation Pilot scheduled-execution JSON body unchanged."""
    return _request("POST", "/scheduled-executions", body=scheduled_execution)


@mcp.tool
def update_scheduled_execution(
    scheduled_execution_id: Annotated[
        str,
        Field(description="ID of the scheduled execution to replace."),
    ],
    scheduled_execution: Annotated[
        dict[str, Any],
        Field(description="Complete replacement scheduled-execution definition."),
    ],
) -> dict[str, Any]:
    """Replace/update a scheduled execution."""
    resource_id = quote(scheduled_execution_id, safe="")
    return _request("PUT", f"/scheduled-executions/{resource_id}", body=scheduled_execution)


@mcp.tool
def delete_scheduled_execution(
    scheduled_execution_id: Annotated[
        str,
        Field(description="ID of the scheduled execution to delete."),
    ],
) -> None:
    """Delete a scheduled execution."""
    resource_id = quote(scheduled_execution_id, safe="")
    _request("DELETE", f"/scheduled-executions/{resource_id}")


# --- Catalogs ---


@mcp.tool
def list_catalogs() -> list[dict[str, Any]]:
    """List Automation Pilot catalogs available to the current service identity."""
    return _request("GET", "/catalogs")


# --- Commands ---


@mcp.tool
def list_commands(
    catalog_id: Annotated[
        str | None,
        Field(
            description="Catalog ID used to filter commands, for example 'http-sapcp'."
        ),
    ] = None,
    tag: Annotated[
        list[str] | None,
        Field(description="Tags used to filter commands."),
    ] = None,
) -> list[dict[str, Any]]:
    """List Automation Pilot commands, optionally filtered by catalog or tags.

    Each result carries only the general fields (id, catalog, name, description,
    tags) to keep the listing cheap. Read a command's input/output contract with
    get_command_contract, or its full definition (including the implementation)
    with get_command.
    """
    params = {"catalog": catalog_id, "tag": tag}
    commands = _request("GET", "/commands", params=params)
    return [_summary(command) for command in commands]


@mcp.tool
def get_command(
    command_id: Annotated[
        str,
        Field(
            description="Full command ID in catalog:name:version format, for example "
            "'http-sapcp:ExecuteHttpRequest:1'."
        ),
    ],
) -> dict[str, Any]:
    """Retrieve one Automation Pilot command definition by its catalog:name:version ID."""
    resource_id = quote(command_id, safe="")
    return _request("GET", f"/commands/{resource_id}")


@mcp.tool
def get_command_contract(
    command_id: Annotated[
        str,
        Field(
            description="Full command ID in catalog:name:version format, for example "
            "'http-sapcp:ExecuteHttpRequest:1'."
        ),
    ],
) -> dict[str, Any]:
    """Retrieve one Automation Pilot command's contract without its implementation.

    Returns the caller-facing contract — id, catalog, name, description, version,
    inputKeys, outputKeys, and tags — and omits the composite `configuration`
    (executors, expressions, control blocks). Use this to trigger or schedule a
    command when you only need its input/output keys. To read or change how the
    command works, use get_command instead.
    """
    resource_id = quote(command_id, safe="")
    command = _request("GET", f"/commands/{resource_id}")
    return _without_configuration(command)


@mcp.tool
def create_command(
    command: Annotated[
        dict[str, Any],
        Field(description="Complete Automation Pilot command definition."),
    ],
) -> dict[str, Any]:
    """Create an Automation Pilot command. Pass the command definition JSON body unchanged."""
    return _request("POST", "/commands", body=command)


@mcp.tool
def update_command(
    command_id: Annotated[
        str,
        Field(description="Full catalog:name:version ID of the command to replace."),
    ],
    command: Annotated[
        dict[str, Any],
        Field(description="Complete replacement Automation Pilot command definition."),
    ],
) -> dict[str, Any]:
    """Replace/update an Automation Pilot command by its catalog:name:version ID."""
    resource_id = quote(command_id, safe="")
    return _request("PUT", f"/commands/{resource_id}", body=command)


@mcp.tool
def delete_command(
    command_id: Annotated[
        str,
        Field(description="Full catalog:name:version ID of the command to delete."),
    ],
) -> None:
    """Delete an Automation Pilot command by its catalog:name:version ID."""
    resource_id = quote(command_id, safe="")
    _request("DELETE", f"/commands/{resource_id}")


# --- Inputs ---


@mcp.tool
def list_inputs(
    catalog_id: Annotated[
        str | None,
        Field(description="Catalog ID used to filter reusable Inputs."),
    ] = None,
    tag: Annotated[
        str | list[str] | None,
        Field(description="Tag or tags used to filter reusable Inputs."),
    ] = None,
) -> list[dict[str, Any]]:
    """List reusable Inputs, optionally filtered by catalog or tag.

    Each result carries only the general fields (id, catalog, name, description,
    tags) to keep the listing cheap. Read one Input's full non-secret metadata
    with get_input. Stored values are never returned.
    """
    params = {"catalog": catalog_id, "tag": tag}
    inputs = _request("GET", "/inputs", params=params)
    return [_summary(item) for item in inputs]


@mcp.tool
def get_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
) -> dict[str, Any]:
    """Retrieve reusable Input metadata by its catalog:name:version ID.

    The stored values are omitted from the result.
    """
    resource_id = quote(input_id, safe="")
    input_definition = _request("GET", f"/inputs/{resource_id}")
    return _without_input_values(input_definition)


@mcp.tool
def create_input(
    input_definition: Annotated[
        dict[str, Any],
        Field(
            description="Complete Automation Pilot Input definition, including values that should be stored."
        ),
    ],
) -> dict[str, Any]:
    """Create a reusable Automation Pilot Input.

    Values are sent to Automation Pilot but are never returned by get_input.
    """
    return _request(
        "POST",
        "/inputs",
        body=input_definition,
        omit_response_fields=frozenset({"values"}),
        suppress_error_payload=True,
    )


@mcp.tool
def update_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
    input_definition: Annotated[
        dict[str, Any],
        Field(
            description="Complete replacement Input definition, including every value that must be preserved."
        ),
    ],
) -> dict[str, Any]:
    """Replace a reusable Automation Pilot Input.

    This is a full replacement. Prefer patch_input for value rotation or focused
    metadata changes because get_input deliberately withholds stored values.
    """
    resource_id = quote(input_id, safe="")
    return _request(
        "PUT",
        f"/inputs/{resource_id}",
        body=input_definition,
        omit_response_fields=frozenset({"values"}),
        suppress_error_payload=True,
    )


@mcp.tool
def patch_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
    input_patch: Annotated[
        dict[str, Any],
        Field(
            description="Merge patch containing only Input fields or values that should change."
        ),
    ],
) -> dict[str, Any]:
    """Partially update a reusable Input without replacing omitted values."""
    resource_id = quote(input_id, safe="")
    return _request(
        "PATCH",
        f"/inputs/{resource_id}",
        body=input_patch,
        headers={"Content-Type": "application/merge-patch+json"},
        omit_response_fields=frozenset({"values"}),
        suppress_error_payload=True,
    )


@mcp.tool
def delete_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
) -> None:
    """Delete a reusable Input by its catalog:name:version ID."""
    resource_id = quote(input_id, safe="")
    _request("DELETE", f"/inputs/{resource_id}")


@mcp.tool
def release_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
) -> dict[str, Any] | None:
    """Irreversibly release a reusable Automation Pilot Input."""
    resource_id = quote(input_id, safe="")
    return _request("PUT", f"/inputs/{resource_id}/release")


@mcp.tool
def deprecate_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
) -> dict[str, Any] | None:
    """Deprecate a reusable Automation Pilot Input."""
    resource_id = quote(input_id, safe="")
    return _request("PUT", f"/inputs/{resource_id}/deprecate")


@mcp.tool
def restore_input(
    input_id: Annotated[
        str,
        Field(description="Full reusable Input ID in catalog:name:version format."),
    ],
) -> dict[str, Any] | None:
    """Restore a deprecated reusable Automation Pilot Input."""
    resource_id = quote(input_id, safe="")
    return _request("PUT", f"/inputs/{resource_id}/restore")


# --- MCP servers ---


@mcp.tool
def list_mcp_servers() -> list[dict[str, Any]]:
    """List all Automation Pilot MCP servers owned by the current tenant."""
    return _request("GET", "/mcp-servers")


@mcp.tool
def get_mcp_server(
    mcp_server_id: Annotated[
        str,
        Field(description="ID of the Automation Pilot MCP server to retrieve."),
    ],
) -> dict[str, Any]:
    """Retrieve one Automation Pilot MCP server by ID."""
    resource_id = quote(mcp_server_id, safe="")
    return _request("GET", f"/mcp-servers/{resource_id}")


@mcp.tool
def create_mcp_server(
    mcp_server: Annotated[
        dict[str, Any],
        Field(description="Complete Automation Pilot MCP server definition."),
    ],
) -> dict[str, Any]:
    """Create an Automation Pilot MCP server. Pass the MCP server JSON body unchanged."""
    return _request("POST", "/mcp-servers", body=mcp_server)


@mcp.tool
def update_mcp_server(
    mcp_server_id: Annotated[
        str,
        Field(description="ID of the Automation Pilot MCP server to replace."),
    ],
    mcp_server: Annotated[
        dict[str, Any],
        Field(description="Complete replacement Automation Pilot MCP server definition."),
    ],
) -> dict[str, Any]:
    """Replace/update an Automation Pilot MCP server by ID."""
    resource_id = quote(mcp_server_id, safe="")
    return _request("PUT", f"/mcp-servers/{resource_id}", body=mcp_server)


@mcp.tool
def delete_mcp_server(
    mcp_server_id: Annotated[
        str,
        Field(description="ID of the Automation Pilot MCP server to delete."),
    ],
) -> None:
    """Delete an Automation Pilot MCP server by ID."""
    resource_id = quote(mcp_server_id, safe="")
    _request("DELETE", f"/mcp-servers/{resource_id}")


# --- Executions ---


@mcp.tool
def trigger_execution(
    execution: Annotated[
        dict[str, Any],
        Field(
            description="Execution request containing commandId and optional "
            "inputValues, inputReferences, and tags."
        ),
    ],
) -> dict[str, Any]:
    """Trigger an Automation Pilot command. Pass the execution JSON body unchanged.

    The body carries commandId plus optional inputValues, inputReferences
    (a flat list of existing Input IDs, e.g. ['tenant:Credentials:1']), and tags.
    """
    return _request("POST", "/executions", body=execution)


@mcp.tool
def list_executions(
    status: Annotated[
        str | list[str] | None,
        Field(description="Execution status or statuses used to filter results."),
    ] = None,
    command_id: Annotated[
        str | list[str] | None,
        Field(description="Command ID or IDs used to filter executions."),
    ] = None,
    execution_id: Annotated[
        str | list[str] | None,
        Field(description="Execution ID or IDs used to filter results."),
    ] = None,
    started_after: Annotated[
        int | None,
        Field(description="Return executions started after this epoch-millisecond time."),
    ] = None,
    started_before: Annotated[
        int | None,
        Field(description="Return executions started before this epoch-millisecond time."),
    ] = None,
    tag: Annotated[
        str | list[str] | None,
        Field(description="Tag or tags used to filter executions."),
    ] = None,
    limit: Annotated[
        int | None,
        Field(description="Maximum number of executions to return."),
    ] = None,
) -> list[dict[str, Any]]:
    """List Automation Pilot executions, filtered by status, command, ID, time window, tag, or limit."""
    params = {
        "status": status,
        "commandId": command_id,
        "executionId": execution_id,
        "startedAfter": started_after,
        "startedBefore": started_before,
        "tag": tag,
        "limit": limit,
    }
    return _request("GET", "/executions", params=params)


@mcp.tool
def get_execution(
    execution_id: Annotated[
        str,
        Field(description="The ID of the execution to retrieve."),
    ],
    include_input: Annotated[
        bool,
        Field(description="Whether to include the input values used for the execution."),
    ] = False,
    include_output: Annotated[
        bool,
        Field(
            description="Whether to include the output values produced by the execution. "
            "FINISHED executions have complete output; ABORTED and SUSPENDED "
            "executions may have partial output."
        ),
    ] = False,
) -> dict[str, Any]:
    """Retrieve one Automation Pilot execution by ID.

    Returns the execution's status and details. Set include_input to also fetch
    the resolved input, and include_output to also fetch persisted output.
    FINISHED executions have complete output; ABORTED and SUSPENDED executions
    may have partial output.
    """
    resource_id = quote(execution_id, safe="")
    execution = _request("GET", f"/executions/{resource_id}")

    if include_input:
        execution["input"] = _request("GET", f"/executions/{resource_id}/input")
    if include_output:
        execution["output"] = _request("GET", f"/executions/{resource_id}/output")

    return execution


@mcp.tool
def get_execution_log(
    execution_id: Annotated[
        str,
        Field(description="ID of the Automation Pilot execution."),
    ],
    executor_path: Annotated[
        str,
        Field(description="Path or alias of the executor whose log to retrieve."),
    ],
) -> dict[str, Any]:
    """Retrieve the execution log for a single executor within an Automation Pilot execution.

    executor_path is the path to the executor.
    """
    resource_id = quote(execution_id, safe="")
    executor = quote(executor_path, safe="")
    return _request("GET", f"/executions/{resource_id}/logs/{executor}")


if __name__ == "__main__":
    mcp.run()
