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
| `email` | String | No | Account email address. Changing it forces replacement. Not readable from the API. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `id` | Account ID. |
| `arn` | Account root ARN (`arn:aws:iam::<id>:root`). |
| `canonical_id` | Canonical ID of the account. |
| `access_key` | S3 access key. Sensitive. Only available at creation. |
| `secret_key` | S3 secret key. Sensitive. Only available at creation. |

## Import

```bash
tofu import artesca_account.team_a team-a
```

After import, `access_key`, `secret_key`, and `email` are null — none of them can be read back from the API. If your configuration sets `email`, the next apply adopts it into state without replacing the account. To get new account keys, create them in the ARTESCA UI; `artesca_user_access_key` creates keys for IAM users, not for the account.

## Notes

- Uses OIDC credentials from the provider configuration.
- The `name` attribute forces replacement -- accounts cannot be renamed.
- `access_key` and `secret_key` are generated once at creation and preserved in state. They are not refreshed on subsequent reads.
