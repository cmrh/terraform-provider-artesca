#!/usr/bin/env bash
# State upgrade test: create every account-scoped resource with an older
# provider build, then plan with the current working tree. Passes only if the
# new provider upgrades the state and the plan shows no changes.
#
# Usage: tests/upgrade/run.sh [BASE_REF]   (default d7b13e1, the last commit
#        before account_name replaced account_access_key/account_secret_key)
#
# Reads ~/.scality-artesca.env (override with ENV_FILE): the ARTESCA_* provider
# settings plus RING_S3_* and DEST_RING_S3_* for the two storage locations.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="${ENV_FILE:-$HOME/.scality-artesca.env}"
BASE_REF="${1:-d7b13e1}"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: Environment file not found: $ENV_FILE"
  exit 1
fi
set -a
# shellcheck source=/dev/null
source "$ENV_FILE"
set +a
unset TF_ACC_PROVIDER_NAMESPACE
for v in RING_S3_ENDPOINT RING_S3_ACCESS_KEY RING_S3_SECRET_KEY RING_S3_BUCKET_NAME \
         DEST_RING_S3_ENDPOINT DEST_RING_S3_ACCESS_KEY DEST_RING_S3_SECRET_KEY DEST_RING_S3_BUCKET_NAME; do
  if [ -z "${!v:-}" ]; then
    echo "ERROR: $v not set in $ENV_FILE"
    exit 1
  fi
done

WORK="$(mktemp -d)"
OLD_BIN="$WORK/bin-old"
NEW_BIN="$WORK/bin-new"
RUN="$WORK/run"
mkdir -p "$OLD_BIN" "$NEW_BIN" "$RUN" "$WORK/src-old"

write_rc() { # write_rc <bin-dir> <rc-file>
  printf 'provider_installation {\n  dev_overrides {\n    "registry.opentofu.org/cmrh/artesca" = "%s"\n  }\n  direct {}\n}\n' "$1" > "$2"
}

# Destroy with the provider that matches the state's schema, then remove the
# work dir (it holds a tfvars file with the RING credentials).
STAGE=none
cleanup() {
  status=$?
  if [ "$STAGE" != none ]; then
    echo "==> Destroying test resources ($STAGE provider)..."
    if [ "$STAGE" = old ]; then cp "$RUN/v0.tf.src" "$RUN/main.tf"; rc="$WORK/old.tofurc"; else cp "$RUN/v1.tf.src" "$RUN/main.tf"; rc="$WORK/new.tofurc"; fi
    (cd "$RUN" && TF_CLI_CONFIG_FILE="$rc" tofu destroy -auto-approve -no-color >"$WORK/destroy.log" 2>&1) \
      && echo "    destroyed" || { echo "    DESTROY FAILED — see output below; resources may be left behind"; tail -20 "$WORK/destroy.log"; status=1; }
  fi
  rm -rf "$WORK"
  exit $status
}
trap cleanup EXIT

echo "==> Building base provider ($BASE_REF)..."
git -C "$REPO_ROOT" archive "$BASE_REF" | tar -x -C "$WORK/src-old"
(cd "$WORK/src-old" && go build -o "$OLD_BIN/terraform-provider-artesca" .)

echo "==> Building current provider (working tree)..."
(cd "$REPO_ROOT" && go build -o "$NEW_BIN/terraform-provider-artesca" .)

write_rc "$OLD_BIN" "$WORK/old.tofurc"
write_rc "$NEW_BIN" "$WORK/new.tofurc"

SUFFIX="$(date +%s | tail -c 7)"
sed "s/SUFFIX/$SUFFIX/g" "$SCRIPT_DIR/v0.tf.tmpl" > "$RUN/v0.tf.src"
sed "s/SUFFIX/$SUFFIX/g" "$SCRIPT_DIR/v1.tf.tmpl" > "$RUN/v1.tf.src"
(umask 077 && cat > "$RUN/terraform.tfvars" <<TFVARS
ring = { endpoint = "$RING_S3_ENDPOINT", access_key = "$RING_S3_ACCESS_KEY", secret_key = "$RING_S3_SECRET_KEY", bucket_name = "$RING_S3_BUCKET_NAME" }
dest_ring = { endpoint = "$DEST_RING_S3_ENDPOINT", access_key = "$DEST_RING_S3_ACCESS_KEY", secret_key = "$DEST_RING_S3_SECRET_KEY", bucket_name = "$DEST_RING_S3_BUCKET_NAME" }
TFVARS
)

cd "$RUN"
echo "==> Applying v0 configuration with $BASE_REF..."
cp v0.tf.src main.tf
STAGE=old
TF_CLI_CONFIG_FILE="$WORK/old.tofurc" tofu apply -auto-approve -no-color >"$WORK/apply.log" 2>&1 \
  || { tail -30 "$WORK/apply.log"; echo "FAIL: v0 apply failed"; exit 1; }
grep -E "Apply complete" "$WORK/apply.log"


echo "==> Planning v1 configuration with the current provider..."
cp v1.tf.src main.tf
STAGE=new
set +e
TF_CLI_CONFIG_FILE="$WORK/new.tofurc" tofu plan -detailed-exitcode -no-color >"$WORK/plan.log" 2>&1
code=$?
set -e
case $code in
  0) echo "    no changes" ;;
  2) grep -E "^  # |will be|must be replaced" "$WORK/plan.log" || true; echo "FAIL: plan after upgrade has changes"; exit 1 ;;
  *) tail -30 "$WORK/plan.log"; echo "FAIL: plan after upgrade errored"; exit 1 ;;
esac

echo "==> Persisting upgraded state and re-planning..."
TF_CLI_CONFIG_FILE="$WORK/new.tofurc" tofu apply -auto-approve -no-color >"$WORK/apply2.log" 2>&1 \
  || { tail -30 "$WORK/apply2.log"; echo "FAIL: apply after upgrade failed"; exit 1; }
python3 - <<'PY'
import json, sys
st = json.load(open("terraform.tfstate"))
bad = []
for r in st["resources"]:
    if r["mode"] != "managed" or r["type"] in ("artesca_account", "artesca_location"):
        continue
    for i in r["instances"]:
        a = i["attributes"]
        if i.get("schema_version") != 1 or not a.get("account_name") or "account_access_key" in a or "account_secret_key" in a:
            bad.append(f'{r["type"]}.{r["name"]} v{i.get("schema_version")} account_name={a.get("account_name")}')
if bad:
    print("FAIL: resources not upgraded:\n  " + "\n  ".join(bad))
    sys.exit(1)
print("    all account-scoped resources at schema v1 with account_name and no account keys")
PY
set +e
TF_CLI_CONFIG_FILE="$WORK/new.tofurc" tofu plan -detailed-exitcode -no-color >"$WORK/plan2.log" 2>&1
code=$?
set -e
[ $code -eq 0 ] || { tail -30 "$WORK/plan2.log"; echo "FAIL: second plan exit code $code"; exit 1; }
echo "    no changes"

echo "PASS: state upgrade from $BASE_REF"
