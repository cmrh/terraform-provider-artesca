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
  account_name = artesca_account.app.name
  name         = "engineers"
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `name` | String | Yes | The group name (1–128 chars; alphanumeric and `+=,.@-`). Forces replacement. |

## Attributes Reference

| Name | Description |
|------|-------------|
| `group_id` | Unique IAM group ID. |
| `arn` | ARN of the group. |
| `path` | Group path (always `/`). |

## Import

```bash
tofu import artesca_group.engineers <account_name>/<name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- All attributes force replacement; groups cannot be renamed in place.
- To add users to a group, use `artesca_group_membership`. To attach permissions, use `artesca_group_policy` (inline) or `artesca_group_policy_attachment` (managed).
