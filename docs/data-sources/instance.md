---
page_title: "artesca_instance Data Source - artesca"
subcategory: ""
description: |-
  Returns metadata and current status for an ARTESCA instance.
---

# Data Source: artesca_instance

Returns metadata + current status for an ARTESCA instance. Combines the management API endpoints `GET /instance/{uuid}` (identity, provisioning state, public key) and `GET /instance/{uuid}/status` (most recent health snapshot).

Useful for:

- Asserting a minimum server version before applying configuration that depends on a recent feature.
- Surfacing instance health (`ip_address`, `last_seen`, `running_configuration_version`) as Terraform outputs for runbooks.
- Sanity-checking that the provider is talking to the expected ARTESCA instance.

## Example

```hcl
data "artesca_instance" "current" {}

output "artesca_instance_id" {
  value = data.artesca_instance.current.instance_id
}

output "artesca_server_version" {
  value = data.artesca_instance.current.server_version
}

# Optional: assert state during plan.
check "instance_confirmed" {
  assert {
    condition     = data.artesca_instance.current.state == "confirmed"
    error_message = "ARTESCA instance is not in 'confirmed' state."
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `instance_id` | String | Optional | Instance ID. Defaults to the provider's `instance_id` if omitted. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `created_at` | Instance creation timestamp (RFC 3339). |
| `state` | Provisioning state — `"confirmed"`, `"provisioning"`, or `"new"`. |
| `public_key` | PEM-encoded public key used to verify instance-issued tokens. |
| `ip_address` | Most recently reported IP address. |
| `last_seen` | Timestamp of the most recent status report (RFC 3339). |
| `running_configuration_version` | Currently running configuration overlay version. |
| `server_version` | Server build identifier. On production builds this is a release version; on development builds it can be a git ref. |

## Notes

- The data source intentionally does **not** surface the `metrics` block from `/status` or the nested `capabilities` / `latestConfigurationOverlay` objects — those are large, change frequently, and are better consumed directly via the API if needed.
- The server may return fewer fields than the swagger schema advertises (e.g. `friendlyName`, `organizationID`, `owner` are documented in the swagger but not populated by the server today). Only fields the server actually returns are modeled here.
