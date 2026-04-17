# artesca_user

Creates an IAM user within an ARTESCA account. Users can be assigned policies and access keys for day-to-day S3 operations.

## Example

```hcl
resource "artesca_user" "operator" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = "bucket-operator"
}

output "user_arn" {
  value = artesca_user.operator.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | IAM username. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `user_id` | Unique user identifier. |
| `arn` | ARN of the user. |
| `path` | IAM path of the user. |

## Import

```bash
tofu import artesca_user.operator bucket-operator
```

## Notes

- All arguments force replacement -- users cannot be renamed or moved between accounts.
- Delete the user's access keys and policies before deleting the user.
- Uses AWS SigV4 signing against the IAM API (endpoint derived from management endpoint).
