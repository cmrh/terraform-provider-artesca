---
page_title: "artesca_policy_document Data Source - artesca"
subcategory: "Identity"
description: |-
  Builds an IAM policy document as JSON.
---

# Data Source: artesca_policy_document

Builds an IAM policy document as JSON from HCL blocks, for `artesca_policy`, `artesca_user_policy`, `artesca_group_policy`, `artesca_role` (trust policy) and `artesca_bucket_policy`. It works like `aws_iam_policy_document`. The document is computed locally; the data source makes no API calls and needs no `account_name`.

## Example — managed policy

```hcl
data "artesca_policy_document" "read_data" {
  statement {
    sid       = "ReadData"
    actions   = ["s3:GetObject", "s3:ListBucket"]
    resources = [
      "arn:aws:s3:::${artesca_bucket.data.name}",
      "arn:aws:s3:::${artesca_bucket.data.name}/*",
    ]
  }
}

resource "artesca_policy" "read_data" {
  account_name    = artesca_account.app.name
  name            = "read-data"
  policy_document = data.artesca_policy_document.read_data.json
}
```

## Example — role trust policy

```hcl
data "artesca_policy_document" "trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${artesca_account.app.id}:root"]
    }
  }
}

resource "artesca_role" "deployer" {
  account_name                = artesca_account.app.name
  name                        = "deployer"
  assume_role_policy_document = data.artesca_policy_document.trust.json
}
```

## Example — combining documents

```hcl
data "artesca_policy_document" "combined" {
  source_policy_documents   = [data.artesca_policy_document.read_data.json]
  override_policy_documents = [file("${path.module}/deny-deletes.json")]

  statement {
    sid       = "Tagging"
    actions   = ["s3:PutBucketTagging"]
    resources = ["arn:aws:s3:::${artesca_bucket.data.name}"]
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `version` | String | No | Policy language version: `2012-10-17` (default) or `2008-10-17`. |
| `statement` | Block | No | A policy statement; repeatable. See below. |
| `source_policy_documents` | List of String | No | Policy documents (JSON) whose statements come first. A Sid may appear in only one of them; a `statement` block with the same Sid replaces it. |
| `override_policy_documents` | List of String | No | Policy documents (JSON) applied last, in order: a statement replaces the statement with its Sid; one without a Sid, or with a new Sid, is appended. |

### Statement Block

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `sid` | String | No | Statement ID. Must be unique among `statement` blocks. |
| `effect` | String | No | `Allow` (default) or `Deny`. |
| `actions` | Set of String | No | Actions, such as `s3:GetObject`. Conflicts with `not_actions`. |
| `not_actions` | Set of String | No | Actions the statement doesn't apply to. Conflicts with `actions`. |
| `resources` | Set of String | No | Resource ARNs. Conflicts with `not_resources`. |
| `not_resources` | Set of String | No | Resource ARNs the statement doesn't apply to. Conflicts with `resources`. |
| `principals` | Block | No | Principals (bucket policies and trust policies); repeatable. `type` (such as `AWS` or `Service`) and `identifiers` (set of ARNs). Blocks of the same type are merged. `type = "*"` with `identifiers = ["*"]` is everyone and must be the only block. Conflicts with `not_principals`. |
| `not_principals` | Block | No | Same shape as `principals`. Conflicts with `principals`. |
| `condition` | Block | No | Repeatable. `test` (operator, such as `StringEquals`), `variable` (condition key, such as `aws:SourceIp`) and `values` (list; any one may match). |

## Attributes Exported

| Name | Description |
|------|-------------|
| `json` | The policy document as indented JSON. A list with one value is written as a plain string, and sets are sorted, so the output is stable. |

## Notes

- Account principals in ARTESCA are written `arn:aws:iam::<account_id>:root`; `artesca_account.arn` has that form.
- ARTESCA validates the document when it is used. Bucket policies can't use the server-side-encryption condition keys.
