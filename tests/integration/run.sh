#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="${ENV_FILE:-$HOME/.scality-artesca.env}"
TFVARS_FILE="${TFVARS_FILE:-$SCRIPT_DIR/integration.tfvars}"

# --- Load credentials ---

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: Environment file not found: $ENV_FILE"
  echo "Copy .env.example to $ENV_FILE and fill in credentials."
  exit 1
fi

if [ ! -f "$TFVARS_FILE" ]; then
  echo "ERROR: Variables file not found: $TFVARS_FILE"
  echo "Copy tests/integration/integration.tfvars.example to integration.tfvars and fill in values."
  exit 1
fi

# shellcheck source=/dev/null
source "$ENV_FILE"

# --- Build provider ---

echo "==> Building provider..."
cd "$REPO_ROOT"
go build -o terraform-provider-artesca .

# --- Set up dev override ---

TERRAFORMRC=$(mktemp)
trap 'rm -f "$TERRAFORMRC"' EXIT

cat > "$TERRAFORMRC" <<EOF
provider_installation {
  dev_overrides {
    "${TF_ACC_PROVIDER_NAMESPACE}/artesca" = "$REPO_ROOT"
  }
  direct {}
}
EOF

export TF_CLI_CONFIG_FILE="$TERRAFORMRC"

TOFU="${TF_ACC_TERRAFORM_PATH:-tofu}"

# --- Run integration test ---

cd "$SCRIPT_DIR"

cleanup() {
  echo "==> Cleaning up: running tofu destroy..."
  "$TOFU" destroy -auto-approve -var-file="$TFVARS_FILE" 2>&1 || echo "WARNING: destroy failed during cleanup"
}

echo "==> Running tofu apply..."
if ! "$TOFU" apply -auto-approve -var-file="$TFVARS_FILE"; then
  echo "ERROR: apply failed, attempting cleanup..."
  cleanup
  exit 1
fi

echo "==> Running tofu plan (drift check)..."
PLAN_EXIT=0
"$TOFU" plan -detailed-exitcode -var-file="$TFVARS_FILE" || PLAN_EXIT=$?

if [ "$PLAN_EXIT" -eq 2 ]; then
  echo "ERROR: Drift detected after apply — plan shows changes."
  cleanup
  exit 1
elif [ "$PLAN_EXIT" -ne 0 ]; then
  echo "ERROR: Plan command failed (exit code $PLAN_EXIT)."
  cleanup
  exit 1
fi

echo "==> Running tofu destroy..."
"$TOFU" destroy -auto-approve -var-file="$TFVARS_FILE"

echo "==> Integration tests passed."
