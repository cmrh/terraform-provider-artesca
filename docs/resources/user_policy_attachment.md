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
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = artesca_user.alice.username
  policy_arn         = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | The IAM user to attach the policy to. Forces replacement. |
| `policy_arn` | String | Yes | The ARN of the managed policy. Forces replacement. |

## Notes

- Each resource manages exactly one user–policy pairing.
- Use `artesca_user_policy` instead if you need an inline (per-user) policy.
