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
  account_name = artesca_account.app.name
  name         = "read-only-s3"
  description  = "Read-only access to all buckets in this account"

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
  account_name = artesca_account.app.name
  username     = artesca_user.alice.username
  policy_arn   = artesca_policy.read_only_s3.arn
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
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

## Import

```bash
tofu import artesca_policy.read_only_s3 <account_name>/<arn>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Changes to `policy_document` made outside Terraform appear in the next plan. Whitespace and key-order differences are ignored.
- All attributes force replacement on change. Update the policy by replacing the resource (Terraform will delete and re-create).
- ARTESCA's IAM API supports policy versions; this provider creates and reads only the default version.
- Names are unique within the account regardless of case: `Alice` and `alice` can't both exist.
