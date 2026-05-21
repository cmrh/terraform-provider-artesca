---
page_title: "artesca_role_policy_attachment Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches a managed IAM policy to a role.
---

# artesca_role_policy_attachment

Attaches a managed IAM policy (created via `artesca_policy`) to an IAM role.

~> **Note:** ARTESCA does not implement inline role policies (`PutRolePolicy` returns `NotImplemented`). Attaching a managed policy via this resource is the only way to grant permissions to a role.

## Example

```hcl
resource "artesca_role_policy_attachment" "deployer_s3" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  role_name          = artesca_role.deployer.name
  policy_arn         = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `role_name` | String | Yes | The IAM role to attach the policy to. Forces replacement. |
| `policy_arn` | String | Yes | The ARN of the managed policy. Forces replacement. |

## Notes

- Each resource manages exactly one role–policy pairing.
