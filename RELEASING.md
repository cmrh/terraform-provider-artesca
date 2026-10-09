# Releasing

This document describes how to cut a release of the ARTESCA Terraform Provider.

## Prerequisites

- Push access to the `cmrh/terraform-provider-artesca` repository
- GPG key registered as a GitHub Actions secret (see [GPG Setup](#gpg-setup) below)
- Test environment configured on the self-hosted runner (see [Test Environment](#test-environment) below)

## Release Flow

### 1. Prepare

- Ensure `main` is green: unit tests, formatting, and vet all pass in CI
- Run acceptance tests against real infrastructure: trigger the **Acceptance Tests** workflow manually via `Actions > Acceptance Tests > Run workflow`
- Update `CHANGELOG.md` — move items from `[Unreleased]` into a new version section

### 2. Internal / Pre-release

Tag with a pre-release suffix to build and test artifacts without publishing to any registry:

```bash
git tag v0.4.0-rc1
git push origin v0.4.0-rc1
```

This produces a full set of release artifacts (binaries, checksums, signed SHA256SUMS) but marks the GitHub release as **pre-release**. Both the Terraform and OpenTofu registries ignore pre-releases.

Valid pre-release suffixes: `-rc1`, `-beta.1`, `-dev.1` (any of `-rc`, `-beta`, `-dev`).

### 3. Public Release

Once the pre-release has been verified, tag the same commit without a suffix:

```bash
git tag v0.4.0
git push origin v0.4.0
```

This creates a full GitHub release that registries will discover and serve to users.

### 4. Verify

After the release workflow completes:

- [ ] GitHub release page has `.zip` archives for all 5 platforms
- [ ] `SHA256SUMS` file is present and contains all 5 checksums plus the manifest
- [ ] `SHA256SUMS.sig` file is present (GPG signature)
- [ ] Manifest JSON has correct version and `"protocols": ["6.0"]`
- [ ] Release is **not** marked as pre-release (for public releases)

Run `scripts/verify-release.sh <tag>` for an automated check of the same criteria plus zip-content sanity checks.

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **Patch** (`v0.4.1`): bug fixes, dependency updates
- **Minor** (`v0.5.0`): new resources, new attributes, non-breaking changes
- **Major** (`v1.0.0`, `v2.0.0`): breaking changes to resource schemas or provider configuration

All tags must include the patch number (use `v0.4.0`, not `v0.4`).

## What the Release Workflow Does

```
Tag push
  └─ test job: unit tests (-race), go vet, gofmt
       └─ build job (5 platforms in parallel):
            - Builds binary with version injected via -ldflags
            - Binary name: terraform-provider-artesca_v{VERSION}  (with v)
            - Archive name: terraform-provider-artesca_{VERSION}_{os}_{arch}.zip  (no v)
            - Generates per-archive SHA256 checksum
            - Uploads archive + checksum as an immutable within-run artifact
                └─ release job:
                     - Downloads all build artifacts
                     - Copies terraform-registry-manifest.json into the release set
                     - Aggregates a single SHA256SUMS covering zips + manifest
                     - Re-verifies every checksum against the downloaded artifacts
                     - Signs SHA256SUMS with the GPG key
                     - Publishes all assets to the GitHub release atomically
```

The build → release handoff via workflow artifacts (not through the mutable GitHub release) closes the window where a tampered asset could pick up a valid signature.

## GPG Setup

The release workflow signs checksums with a GPG key. This is required for publishing to the Terraform and OpenTofu registries.

### One-time setup

1. **Generate a key** (RSA 4096 — registries do not support ECC):

   ```bash
   gpg --full-generate-key
   # Select: (1) RSA and RSA
   # Key size: 4096
   # Expiry: 0 (does not expire) or 2y
   # Name/email: use the org identity
   ```

2. **Add secrets to GitHub** (Settings > Secrets and variables > Actions):

   ```bash
   # Private key — paste output as GPG_PRIVATE_KEY secret
   gpg --armor --export-secret-keys YOUR_EMAIL

   # Passphrase — paste as GPG_PASSPHRASE secret
   ```

3. **Register the public key with registries** (when ready for public publishing):

   ```bash
   gpg --armor --export YOUR_EMAIL
   ```

   - **Terraform Registry**: registry.terraform.io > Sign in > Settings > GPG Keys
   - **OpenTofu Registry**: registry.opentofu.org > Sign in > Settings > GPG Keys

### Key rotation

When rotating the GPG key:

1. Generate a new key
2. Update both GitHub secrets (`GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`)
3. Add the new public key to both registries (keep the old key — it's needed to verify past releases)

## Test Environment

The acceptance workflow (`.github/workflows/acceptance.yml`, run manually or on `v*` tags) runs on the self-hosted runner `cm-tf-provider-artesca` against a real ARTESCA cluster. It reads repository secrets and hardcodes `ARTESCA_INSECURE_SKIP_VERIFY="true"` and `TF_ACC_TERRAFORM_PATH=/usr/bin/tofu`.

### Repository secrets

| Secret | Exported as | Purpose |
|--------|-------------|---------|
| `ARTESCA_MANAGEMENT_ENDPOINT` | same name | Management API endpoint (OIDC-authenticated) |
| `ARTESCA_OIDC_URL` | same name | Control-plane ingress URL. Must produce tokens whose issuer matches the internal trust policies — retrieve via `salt-call metalk8s_network.get_control_plane_ingress_endpoint --out=json` |
| `ARTESCA_USERNAME` / `ARTESCA_PASSWORD` | same names | OIDC credentials for the management API |
| `ARTESCA_S3_ENDPOINT` | same name | Data-service endpoint for S3 operations |
| `RING_S3_*` | `TF_VAR_ring_s3_*` | RING backend for location, replication and workflow tests (source) |
| `DEST_RING_S3_*` | `TF_VAR_dest_ring_s3_*` | RING backend used as the replication destination |

### Running locally

Export the same variables (an env file sourced with `set -a` works), plus:

```bash
export TF_ACC_TERRAFORM_PATH="$(which tofu)"
export TF_ACC_PROVIDER_NAMESPACE="scality"
make testacc
```

### Notes

- `make testacc` sets `TF_ACC_PROVIDER_NAMESPACE` to the Makefile's `scality` namespace.
- Location, replication and workflow tests fail if the `TF_VAR_ring_s3_*` or `TF_VAR_dest_ring_s3_*` variables are not set.

## Local Development Builds

For local testing without a release:

```bash
# Build the binary in the repository root
make build

# Or use dev_overrides in ~/.terraformrc / tofu config:
provider_installation {
  dev_overrides {
    "registry.opentofu.org/cmrh/artesca" = "/path/to/built/binary/directory"
  }
  direct {}
}
```

## Registry Publishing

Both registries auto-discover releases from GitHub — no manual upload needed. Requirements:

- GitHub repository must be **public**
- Repository name must follow `terraform-provider-{name}` convention
- Release artifacts must include signed SHA256SUMS
- GPG public key must be registered with the registry

The provider is published under `cmrh/artesca` in both registries. Users use it as:

```hcl
terraform {
  required_providers {
    artesca = {
      source  = "cmrh/artesca"
      version = "~> 0.4.0"
    }
  }
}
```
