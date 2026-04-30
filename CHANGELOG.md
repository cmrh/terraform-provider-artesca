# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - Unreleased

### Added

- **Provider**: OIDC + management API authentication with auto-discovered instance ID
- **artesca_account**: Account management via management API with credential generation
- **artesca_bucket**: S3 bucket creation with versioning and location constraints
- **artesca_location**: Storage location management (AWS S3, Azure, GCP, Scality RING, etc.)
- **artesca_endpoint**: S3 data service endpoint mapping
- **artesca_replication**: Config-scoped overlay replication streams with server-managed versioning
- **artesca_user**: IAM user management within accounts
- **artesca_user_access_key**: IAM access key pair generation
- **artesca_user_policy**: Inline IAM policy attachment
- **artesca_bucket_workflow_expiration**: Object expiration lifecycle workflows
- **artesca_bucket_workflow_transition**: Object transition lifecycle workflows
- **artesca_bucket_workflow_replication**: Bucket-scoped replication workflows
- Import support for account, endpoint, location, and replication resources
- Acceptance tests for all 11 resources with CheckDestroy verification
- GPG-signed release artifacts with SHA256SUMS
