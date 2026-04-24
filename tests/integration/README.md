# Integration Tests

Full create-plan-destroy lifecycle test for all provider resources against a real ARTESCA instance.

## Resources Tested

| Resource | Description |
|----------|-------------|
| `artesca_account` | Storage account with auto-generated credentials |
| `artesca_user` | IAM user within the account |
| `artesca_user_access_key` | Access key pair for the IAM user |
| `artesca_user_policy` | Inline policy attached to the IAM user |
| `artesca_location` (x2) | RING S3 storage locations (source + destination) |
| `artesca_bucket` (x2) | S3 buckets on each location |
| `artesca_endpoint` | DNS endpoint for the source bucket |
| `artesca_bucket_workflow_expiration` | Object expiration lifecycle rule |
| `artesca_bucket_workflow_transition` | Object transition lifecycle rule |
| `artesca_replication` | Cross-location replication stream |

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
salt-call pillar.get artesca:oidc:url
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

The GitHub Actions workflow (`.github/workflows/integration.yml`) runs on `self-hosted` runners. It reads credentials from GitHub repository secrets:

| Secret | Maps to |
|--------|---------|
| `ARTESCA_MANAGEMENT_ENDPOINT` | Provider management API URL |
| `ARTESCA_INSECURE_SKIP_VERIFY` | Skip TLS verification |
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
