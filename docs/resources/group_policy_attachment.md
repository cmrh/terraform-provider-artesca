---
page_title: "artesca_group_policy_attachment Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches a managed IAM policy to a group.
---

# artesca_group_policy_attachment

Attaches a managed IAM policy (created via `artesca_policy`) to an IAM group. All users in the group inherit the permissions.

## Example

```hcl
resource "artesca_group_policy_attachment" "engineers_read" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  group_name         = artesca_group.engineers.name
  policy_arn         = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `group_name` | String | Yes | The IAM group to attach the policy to. Forces replacement. |
| `policy_arn` | String | Yes | The ARN of the managed policy. Forces replacement. |

## Notes

- Each resource manages exactly one group–policy pairing.
- Use `artesca_group_policy` instead if you need an inline (per-group) policy.
