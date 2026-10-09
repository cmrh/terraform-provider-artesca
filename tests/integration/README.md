# Integration Tests

Full create-plan-destroy lifecycle test for all provider resources against a real ARTESCA instance.

## Resources Tested

| Resource | Description |
|----------|-------------|
| `artesca_account` | Storage account |
| `artesca_user`, `artesca_user_access_key`, `artesca_user_policy` | IAM user with an access key and an inline policy |
| `artesca_group`, `artesca_group_policy`, `artesca_group_membership` | IAM group with an inline policy and the user as a member |
| `artesca_policy` and its user, group and role attachments | Managed policy attached three ways |
| `artesca_role` | IAM role |
| `artesca_location` (x2) | RING S3 storage locations (source + destination) |
| `artesca_bucket` (x2) | S3 buckets on each location |
| `artesca_bucket_policy`, `artesca_bucket_tagging` | Policy and tags on the source bucket |
| `artesca_endpoint` | Endpoint for the source location |
| `artesca_bucket_workflow_expiration`, `artesca_bucket_workflow_transition` | Lifecycle rules on the source bucket |
| `artesca_bucket_workflow_replication` | Replication rule from the source to the destination bucket |

## Prerequisites

- Access to a running ARTESCA instance
- Two RING S3 connectors (source and destination) with pre-created buckets
- Go toolchain (version in `go.mod`)
- OpenTofu (`tofu`) installed

## Setup

### 1. Provider environment

Copy the example env file and fill in your ARTESCA credentials:

```bash
cp .env.example ~/.scality-artesca.env
# Edit ~/.scality-artesca.env with real values
```

### 2. RING S3 variables

Copy the example tfvars and fill in your RING S3 connector details:

```bash
cp tests/integration/integration.tfvars.example tests/integration/integration.tfvars
# Edit integration.tfvars with real values
```

### 3. OIDC URL

The `ARTESCA_OIDC_URL` must produce tokens whose `iss` claim matches the issuer configured in the storage-manager-role trust policy. On the ARTESCA device:

```bash
salt-call metalk8s_network.get_control_plane_ingress_endpoint --out=json
```

Use the URL returned by that command.

## Running Locally

```bash
tests/integration/run.sh
```

The script builds the provider, sets up a dev override, then runs `tofu apply`, `tofu plan` (drift check), and `tofu destroy`.

Override file paths with environment variables:

```bash
ENV_FILE=/path/to/env TFVARS_FILE=/path/to/vars tests/integration/run.sh
```

## Running in CI

The GitHub Actions workflow (`.github/workflows/integration.yml`) runs on `self-hosted` runners. It reads credentials from GitHub repository secrets (`ARTESCA_INSECURE_SKIP_VERIFY` is hardcoded to `"true"`):

| Secret | Maps to |
|--------|---------|
| `ARTESCA_MANAGEMENT_ENDPOINT` | Provider management API URL |
| `ARTESCA_OIDC_URL` | OIDC token endpoint |
| `ARTESCA_USERNAME` | OIDC username |
| `ARTESCA_PASSWORD` | OIDC password |
| `ARTESCA_S3_ENDPOINT` | S3 API endpoint |
| `RING_S3_ENDPOINT` | Source RING S3 connector URL |
| `RING_S3_ACCESS_KEY` | Source RING S3 access key |
| `RING_S3_SECRET_KEY` | Source RING S3 secret key |
| `RING_S3_BUCKET_NAME` | Source RING S3 bucket |
| `DEST_RING_S3_ENDPOINT` | Destination RING S3 connector URL |
| `DEST_RING_S3_ACCESS_KEY` | Destination RING S3 access key |
| `DEST_RING_S3_SECRET_KEY` | Destination RING S3 secret key |
| `DEST_RING_S3_BUCKET_NAME` | Destination RING S3 bucket |

## Cleanup

If a run fails mid-apply, resources may be left behind. Clean up manually:

```bash
# Source the env file for credentials
source ~/.scality-artesca.env

# Destroy with tofu (if state file exists)
cd tests/integration
tofu destroy -auto-approve -var-file=integration.tfvars

# Or delete individual resources via the management API
curl -k -X DELETE \
  -H "X-Authentication-Token: $TOKEN" \
  "$ARTESCA_MANAGEMENT_ENDPOINT/api/v1/config/$INSTANCE_ID/location/inttest-ring-loc"
```
