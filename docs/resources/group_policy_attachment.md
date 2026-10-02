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
  account_name = artesca_account.app.name
  group_name   = artesca_group.engineers.name
  policy_arn   = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `group_name` | String | Yes | The IAM group to attach the policy to. Forces replacement. |
| `policy_arn` | String | Yes | The ARN of the managed policy. Forces replacement. |

## Import

```bash
tofu import artesca_group_policy_attachment.engineers_read <account_name>/<group_name>/<policy_arn>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Each resource manages exactly one group–policy pairing.
- Use `artesca_group_policy` instead if you need an inline (per-group) policy.
