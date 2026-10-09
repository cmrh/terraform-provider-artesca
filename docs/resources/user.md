---
page_title: "artesca_user Resource - artesca"
subcategory: "IAM"
description: |-
  Creates an IAM user within an ARTESCA account for S3 operations.
---

# artesca_user

Creates an IAM user within an ARTESCA account. Users can be assigned policies and access keys for day-to-day S3 operations.

## Example

```hcl
resource "artesca_user" "operator" {
  account_name = artesca_account.app.name
  username     = "bucket-operator"
}

output "user_arn" {
  value = artesca_user.operator.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `username` | String | Yes | IAM username. Must be 1-64 characters, alphanumeric and `+=,.@-`. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | Unique user identifier. |
| `arn` | ARN of the user. |
| `path` | IAM path of the user. |

## Import

```bash
tofu import artesca_user.operator <account_name>/<username>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- All arguments force replacement -- users cannot be renamed or moved between accounts.
- Delete the user's access keys and policies before deleting the user.
- Names are unique within the account regardless of case: `Alice` and `alice` can't both exist.
- Uses AWS SigV4 signing against the IAM API (endpoint derived from management endpoint).
