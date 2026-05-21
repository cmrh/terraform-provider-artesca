---
page_title: "artesca_policy Resource - artesca"
subcategory: "IAM"
description: |-
  Manages an IAM managed policy within an ARTESCA account.
---

# artesca_policy

Manages a customer-managed IAM policy. Managed policies are referenced by ARN and can be attached to users, groups, and roles via the `*_policy_attachment` resources.

## Example

```hcl
resource "artesca_policy" "read_only_s3" {
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  name               = "read-only-s3"
  description        = "Read-only access to all buckets in this account"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:ListBucket"]
        Resource = ["arn:aws:s3:::*", "arn:aws:s3:::*/*"]
      }
    ]
  })
}

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
| `name` | String | Yes | Policy name (1–128 chars; alphanumeric and `+=,.@-`). Forces replacement. |
| `policy_document` | String | Yes | JSON policy document. Forces replacement on change. |
| `description` | String | No  | Free-text description. Forces replacement on change. |

## Attributes Reference

| Name | Description |
|------|-------------|
| `arn` | Policy ARN — use this when attaching to users, groups, or roles. |
| `policy_id` | Unique policy ID. |
| `path` | Policy path (always `/`). |
| `default_version_id` | Active policy version ID. |

## Notes

- All attributes force replacement on change. Update the policy by replacing the resource (Terraform will delete and re-create).
- ARTESCA supports policy versions internally, but this provider only manages the active version.
