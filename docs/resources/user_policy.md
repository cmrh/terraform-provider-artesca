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
  account_access_key = artesca_account.app.access_key
  account_secret_key = artesca_account.app.secret_key
  username           = artesca_user.operator.username
  policy_name        = "s3-read-write"
  policy_document    = jsonencode({
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
| `account_access_key` | String | Yes | Access key of the owning account. Sensitive. Forces replacement. |
| `account_secret_key` | String | Yes | Secret key of the owning account. Sensitive. Forces replacement. |
| `username` | String | Yes | IAM username to attach the policy to. Forces replacement. |
| `policy_name` | String | Yes | Name of the inline policy. Forces replacement. |
| `policy_document` | String | Yes | JSON policy document. Updated in-place on change. |

## Import

```bash
export ARTESCA_ACCOUNT_ACCESS_KEY="..."   # access key of the account that owns the resource
export ARTESCA_ACCOUNT_SECRET_KEY="..."
tofu import artesca_user_policy.operator_s3 <username>/<policy_name>
```

The import ID carries no credentials, so the provider reads the owning account's keys from `ARTESCA_ACCOUNT_ACCESS_KEY` / `ARTESCA_ACCOUNT_SECRET_KEY` during import. One account per import run. Keep `account_access_key` / `account_secret_key` in your configuration as usual.

## Notes

- Only `policy_document` can be updated in-place. Changing any other attribute forces replacement.
- The `policy_document` is URL-decoded when read back from the API.
