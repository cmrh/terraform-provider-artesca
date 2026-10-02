---
page_title: "artesca_group_membership Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches an IAM user to a group within an ARTESCA account.
---

# artesca_group_membership

Attaches a single IAM user to an IAM group.

## Example

```hcl
resource "artesca_group_membership" "alice_engineers" {
  account_name = artesca_account.app.name
  group_name   = artesca_group.engineers.name
  username     = artesca_user.alice.username
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `group_name` | String | Yes | The IAM group name. Forces replacement. |
| `username` | String | Yes | The IAM user to add to the group. Forces replacement. |

## Import

```bash
tofu import artesca_group_membership.alice_engineers <account_name>/<group_name>/<username>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- All attributes force replacement; memberships cannot be modified in place.
- Each `artesca_group_membership` manages exactly one user–group pairing. Use multiple resources to add multiple users to one group.
