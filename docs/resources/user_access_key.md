---
page_title: "artesca_user_access_key Resource - artesca"
subcategory: "IAM"
description: |-
  Creates an IAM access key pair for a user within an ARTESCA account.
---

# artesca_user_access_key

Creates an IAM access key pair for a user within an ARTESCA account. The secret key is only available at creation and is preserved in state.

## Example

```hcl
resource "artesca_user_access_key" "operator_key" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = artesca_user.operator.username
}

output "operator_credentials" {
  value = {
    access_key = artesca_user_access_key.operator_key.access_key_id
    secret_key = artesca_user_access_key.operator_key.secret_access_key
  }
  sensitive = true
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | IAM username to create the key for. Forces replacement. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `access_key_id` | The access key ID. |
| `secret_access_key` | The secret access key. Sensitive. Only available at creation. |
| `status` | Key status (`Active` or `Inactive`). |

## Notes

- All arguments force replacement -- changing any value destroys the key and creates a new one.
- The `secret_access_key` is only returned at creation. It is preserved in state but cannot be re-read from the API.
- Import is not supported because the secret key cannot be retrieved after creation.
