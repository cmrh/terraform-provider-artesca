---
page_title: "artesca_user_policy_attachment Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches a managed IAM policy to a user.
---

# artesca_user_policy_attachment

Attaches a managed IAM policy (created via `artesca_policy`) to an IAM user.

## Example

```hcl
resource "artesca_user_policy_attachment" "alice_read" {
  account_name = artesca_account.app.name
  username     = artesca_user.alice.username
  policy_arn   = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `username` | String | Yes | The IAM user to attach the policy to. Forces replacement. |
| `policy_arn` | String | Yes | The ARN of the managed policy. Forces replacement. |

## Import

```bash
tofu import artesca_user_policy_attachment.alice_read <account_name>/<username>/<policy_arn>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Each resource manages exactly one user–policy pairing.
- Use `artesca_user_policy` instead if you need an inline (per-user) policy.
