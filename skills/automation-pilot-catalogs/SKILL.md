---
name: automation-pilot-catalogs
license: Apache-2.0
metadata:
  author: SAP
  version: 1.0.0
description: >
  Creates custom SAP Automation Pilot catalogs. Use when a user wants a new catalog with a technical name, display name, and optional description. Do not use merely to browse or inspect catalogs or content.
---

# SAP Automation Pilot Catalogs

## Goal

Create purposeful custom catalogs without modifying SAP-provided catalogs or scattering related content across unnecessary catalogs. Catalog updates and deletion are intentionally unsupported.

## Decide whether a catalog is needed

Prefer an existing custom catalog when it already has the same purpose and lifecycle. Create a new one when the user needs a durable separation such as a distinct team, product, environment, or content lifecycle. Do not create a catalog merely because one command or Input is being created.

## Creating a custom catalog

Collect and validate the direct `automation-pilot:create_catalog` arguments:

- `technical_name` is required, immutable, alphanumeric, and 1-10 characters. Automation Pilot combines it with the tenant identifier to generate the catalog `id`.
- `display_name` is required, 1-64 characters, and may contain ASCII letters, digits, spaces, hyphens, and underscores.
- `description` is optional and at most 512 characters.

Call `automation-pilot:create_catalog` with the validated fields, capture its returned ID, then read that ID with `automation-pilot:get_catalog` and compare the technical name, display name, and description. A conflict response means the catalog already exists; inspect that catalog instead of retrying with a guessed technical name.
