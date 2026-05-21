---
page_title: "artesca_role Resource - artesca"
subcategory: "IAM"
description: |-
  Manages an IAM role within an ARTESCA account.
---

# artesca_role

Manages an IAM role within an ARTESCA account. Roles are used with STS (AssumeRole) to grant temporary credentials.

~> **Limitation:** ARTESCA does not implement `UpdateAssumeRolePolicy` or inline role policies (`PutRolePolicy`, etc.). The trust policy is set at creation only — changing `assume_role_policy_document` forces the role to be replaced. To grant permissions to a role, use `artesca_policy` + `artesca_role_policy_attachment`.

## Example

```hcl
resource "artesca_role" "deployer" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  name               = "deployer"
  description        = "Used by CI pipelines for deploy operations"

  assume_role_policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::123456789012:user/ci-bot" }
        Action    = "sts:AssumeRole"
      }
    ]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `name` | String | Yes | The role name (1–64 chars; alphanumeric and `+=,.@-`). Forces replacement. |
| `assume_role_policy_document` | String | Yes | Trust-policy JSON. Forces replacement on change. |
| `description` | String | No  | Free-text description. Forces replacement on change. |

## Attributes Reference

| Name | Description |
|------|-------------|
| `role_id` | Unique IAM role ID. |
| `arn` | ARN of the role. |
| `path` | Role path (always `/`). |
