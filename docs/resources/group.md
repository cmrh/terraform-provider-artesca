---
page_title: "artesca_group Resource - artesca"
subcategory: "IAM"
description: |-
  Manages an IAM group within an ARTESCA account.
---

# artesca_group

Manages an IAM group within an ARTESCA account. Groups let you attach a set of permissions once and apply them to multiple users.

## Example

```hcl
resource "artesca_group" "engineers" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  name               = "engineers"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `name` | String | Yes | The group name (1–128 chars; alphanumeric and `+=,.@-`). Forces replacement. |

## Attributes Reference

| Name | Description |
|------|-------------|
| `group_id` | Unique IAM group ID. |
| `arn` | ARN of the group. |
| `path` | Group path (always `/`). |

## Notes

- All attributes force replacement; groups cannot be renamed in place.
- To add users to a group, use `artesca_group_membership`. To attach permissions, use `artesca_group_policy` (inline) or `artesca_group_policy_attachment` (managed).
