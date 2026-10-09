---
page_title: "artesca_group_policy Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches an inline IAM policy to a group within an ARTESCA account.
---

# artesca_group_policy

Attaches an inline IAM policy to a group. All users in the group inherit the permissions. For sharing a policy across multiple principals, prefer `artesca_policy` + `artesca_group_policy_attachment`.

## Example

```hcl
resource "artesca_group_policy" "engineers_s3" {
  account_name = artesca_account.app.name
  group_name   = artesca_group.engineers.name
  policy_name  = "s3-read-write"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject"]
        Resource = "arn:aws:s3:::shared-bucket/*"
      }
    ]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `group_name` | String | Yes | The IAM group name. Forces replacement. |
| `policy_name` | String | Yes | Name of the inline policy. Forces replacement. |
| `policy_document` | String | Yes | JSON policy document. Updated in-place on change. |

## Import

```bash
tofu import artesca_group_policy.engineers_s3 <account_name>/<group_name>/<policy_name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Changes to `policy_document` made outside Terraform appear in the next plan. Whitespace and key-order differences are ignored.
- Only `policy_document` can be updated in-place. Other attributes force replacement.
