---
page_title: "artesca_user_policy Resource - artesca"
subcategory: "IAM"
description: |-
  Attaches an inline IAM policy to a user within an ARTESCA account.
---

# artesca_user_policy

Attaches an inline IAM policy to a user within an ARTESCA account. The policy document follows the standard AWS IAM policy JSON format.

## Example

```hcl
resource "artesca_user_policy" "operator_s3" {
  account_name = artesca_account.app.name
  username     = artesca_user.operator.username
  policy_name  = "s3-read-write"
  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "arn:aws:s3:::my-bucket/*"
      },
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = "arn:aws:s3:::my-bucket"
      }
    ]
  })
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `account_name` | String | Yes | Name of the account that owns the resource. Forces replacement. |
| `username` | String | Yes | IAM username to attach the policy to. Forces replacement. |
| `policy_name` | String | Yes | Name of the inline policy. Forces replacement. |
| `policy_document` | String | Yes | JSON policy document. Updated in-place on change. |

## Import

```bash
tofu import artesca_user_policy.operator_s3 <account_name>/<username>/<policy_name>
```

The import ID starts with the name of the account that owns the resource.

## Notes

- Only `policy_document` can be updated in-place. Changing any other attribute forces replacement.
- The `policy_document` is URL-decoded when read back from the API.
