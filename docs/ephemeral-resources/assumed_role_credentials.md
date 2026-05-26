---
page_title: "artesca_assumed_role_credentials Ephemeral Resource - artesca"
subcategory: "IAM"
description: |-
  Calls sts:AssumeRole and returns temporary credentials. Values are never written to Terraform state.
---

# Ephemeral Resource: artesca_assumed_role_credentials

Calls ARTESCA's STS `AssumeRole` and returns temporary credentials (access key ID, secret access key, session token, expiration). Values exist only for the duration of the current Terraform operation — they are **never written to state**, never appear in plan output, and never leak via state backends.

~> **Requires Terraform 1.10+ or OpenTofu 1.10+.** Older versions don't understand the `ephemeral` block.

The typical use case is bridging ARTESCA-issued temporary credentials into another provider block (AWS, etc.) or into an external tool that runs as part of the same apply.

## Example — Use the assumed role to drive the AWS provider

```hcl
ephemeral "artesca_assumed_role_credentials" "deploy" {
  access_key        = artesca_user_access_key.ci_bot.access_key_id
  secret_key        = artesca_user_access_key.ci_bot.secret_access_key
  role_arn          = artesca_role.deployer.arn
  role_session_name = "tf-deploy-${formatdate("YYYYMMDDhhmmss", timestamp())}"
  duration_seconds  = 3600
}

provider "aws" {
  access_key = ephemeral.artesca_assumed_role_credentials.deploy.access_key_id
  secret_key = ephemeral.artesca_assumed_role_credentials.deploy.secret_access_key
  token      = ephemeral.artesca_assumed_role_credentials.deploy.session_token

  endpoints {
    s3 = "https://s3.my-company.com"
  }
}
```

## Example — Verify via check block

Ephemeral values can't be exposed as root-module outputs, but you can assert on them with `check` blocks:

```hcl
ephemeral "artesca_assumed_role_credentials" "deploy" {
  access_key        = var.caller_access_key
  secret_key        = var.caller_secret_key
  role_arn          = "arn:aws:iam::123456789012:role/deployer"
  role_session_name = "smoke-test"
}

check "assume_role_succeeded" {
  assert {
    condition     = ephemeral.artesca_assumed_role_credentials.deploy.assumed_role_arn != ""
    error_message = "AssumeRole did not return an assumed-role ARN"
  }
}
```

## Argument Reference

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `access_key` | String | Yes | Access key of the IAM user calling `AssumeRole`. Sensitive. |
| `secret_key` | String | Yes | Secret key of the IAM user calling `AssumeRole`. Sensitive. |
| `role_arn` | String | Yes | ARN of the role to assume. |
| `role_session_name` | String | Yes | Session name. Recorded in the assumed-role ARN; useful for audit log correlation. |
| `duration_seconds` | Int64 | No  | Session lifetime. Defaults to STS's default (3600s / 1 hour) when omitted. |
| `external_id` | String | No  | External ID, if the role's trust policy requires one. |
| `policy` | String | No  | Inline session policy (JSON). Further restricts the assumed-role permissions. |

## Attributes Exported

| Name | Description |
|------|-------------|
| `access_key_id` | Temporary access key ID. Sensitive. |
| `secret_access_key` | Temporary secret access key. Sensitive. |
| `session_token` | Session token. Must be sent alongside the access key / secret on every signed request. Sensitive. |
| `expiration` | When the returned credentials expire (RFC3339). |
| `assumed_role_id` | Session identifier (`RoleId:SessionName`). |
| `assumed_role_arn` | ARN of the assumed-role session (`arn:aws:sts::<id>:assumed-role/<role>/<session>`). |

## Notes

- The provider's `s3_endpoint` must be configured (the STS endpoint is derived from it by replacing the leading `s3.` with `sts.`).
- Credentials are re-fetched on every Terraform operation that references the ephemeral block — there's no caching across runs.
- Ephemeral values cannot be assigned to root-module `output` blocks. Use them directly in other provider configurations or in `check` assertions.
