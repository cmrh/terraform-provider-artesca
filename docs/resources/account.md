---
page_title: "artesca_account Resource - artesca"
subcategory: "Accounts"
description: |-
  Manages an ARTESCA account (S3 user) via the management API. Automatically generates S3 credentials on creation.
---

# artesca_account

Manages an ARTESCA account (S3 user) via the management API. Automatically generates S3 credentials on creation.

## Example

```hcl
resource "artesca_account" "team_a" {
  name  = "team-a"
  email = "team-a@example.com"
}

output "team_a_credentials" {
  value = {
    access_key = artesca_account.team_a.access_key
    secret_key = artesca_account.team_a.secret_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | String | Yes | Account name. Forces replacement. |
| `email` | String | No | Account email address. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | Account ID. |
| `arn` | Account ARN. |
| `canonical_id` | Canonical ID of the account. |
| `access_key` | S3 access key. Sensitive. Only available at creation. |
| `secret_key` | S3 secret key. Sensitive. Only available at creation. |

## Import

```bash
tofu import artesca_account.team_a team-a
```

After import, `access_key` and `secret_key` will be unknown. Use `artesca_user_access_key` to generate new credentials if needed.

## Notes

- Uses OIDC credentials from the provider configuration.
- The `name` attribute forces replacement -- accounts cannot be renamed.
- `access_key` and `secret_key` are generated once at creation and preserved in state. They are not refreshed on subsequent reads.
