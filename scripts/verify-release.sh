#!/usr/bin/env bash
#
# Verify a GitHub release of the ARTESCA Terraform provider against the
# OpenTofu Registry artifact conventions.
#
# Usage:
#   scripts/verify-release.sh <tag> [--gpg-key <path>]
#
# Examples:
#   scripts/verify-release.sh v0.4.0-rc1
#   scripts/verify-release.sh v0.4.0 --gpg-key .github/RELEASE_KEY.asc
#
# Checks:
#   1. Tag exists as a GitHub release with assets.
#   2. All five platforms have a .zip and a .zip.sha256 asset.
#   3. Archive filenames use no 'v' prefix; binary inside each zip uses the 'v' prefix.
#   4. SHA256SUMS file lists every zip and matches each zip's actual sha256.
#   5. SHA256SUMS.sig verifies against the provided GPG public key (optional).
#   6. manifest.json is valid JSON with the expected shape.
#   7. Each zip contains exactly one binary with the expected name.

set -euo pipefail

NAME="terraform-provider-artesca"
EXPECTED_PLATFORMS=(
  "linux_amd64"
  "linux_arm64"
  "darwin_amd64"
  "darwin_arm64"
  "windows_amd64"
)

# --- arg parsing ---

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <tag> [--gpg-key <path>]"
  exit 2
fi

TAG="$1"
shift
GPG_KEY=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --gpg-key) GPG_KEY="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

# Strip leading 'v' from the tag to get the version-as-used-in-filenames.
VERSION_NO_V="${TAG#v}"

# --- result tracking ---

PASS_COUNT=0
FAIL_COUNT=0
FAILURES=()

pass() { echo "  ✓ $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
fail() { echo "  ✗ $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); FAILURES+=("$1"); }

# --- workspace ---

# Resolve the GitHub repo from the current git checkout so we can call gh from
# a temp directory. Override with REPO env var if needed.
if [ -z "${REPO:-}" ]; then
  REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
fi
if [ -z "$REPO" ]; then
  echo "ERROR: could not determine GitHub repo. Run from inside the repo or set REPO=owner/name." >&2
  exit 2
fi

WORK=$(mktemp -d -t verify-release-XXXXXX)
trap 'rm -rf "$WORK"' EXIT

echo "==> Verifying release $TAG"
echo "    Repo: $REPO"
echo "    Expected version-in-filenames: $VERSION_NO_V"
echo "    Workspace: $WORK"
echo

# --- step 1: release exists ---

echo "==> Step 1: release exists"
if ! gh release view "$TAG" --repo "$REPO" --json tagName >/dev/null 2>&1; then
  fail "release $TAG not found on GitHub"
  echo
  echo "==> Aborting (no release to inspect)"
  exit 1
fi
pass "release $TAG exists"
echo

# --- step 2: download all assets ---

echo "==> Step 2: downloading release assets"
cd "$WORK"
gh release download "$TAG" --repo "$REPO" 2>&1 | sed 's/^/    /'
echo "    files downloaded:"
ls -1 | sed 's/^/      /'
echo

# --- step 3: filename convention + coverage ---

echo "==> Step 3: filename convention + platform coverage"
for plat in "${EXPECTED_PLATFORMS[@]}"; do
  zip_name="${NAME}_${VERSION_NO_V}_${plat}.zip"
  sha_name="${zip_name}.sha256"
  if [ -f "$zip_name" ]; then pass "$zip_name present"; else fail "$zip_name missing"; fi
  if [ -f "$sha_name" ]; then pass "$sha_name present"; else fail "$sha_name missing"; fi
done

# Detect any artifact that still has the legacy 'v' prefix.
if ls "${NAME}_v"* 2>/dev/null 1>&2; then
  for f in "${NAME}_v"*; do
    fail "legacy 'v'-prefixed asset present: $f"
  done
fi

SHA_FILE="${NAME}_${VERSION_NO_V}_SHA256SUMS"
SIG_FILE="${SHA_FILE}.sig"
MANIFEST_FILE="${NAME}_${VERSION_NO_V}_manifest.json"

[ -f "$SHA_FILE" ]      && pass "$SHA_FILE present"      || fail "$SHA_FILE missing"
[ -f "$SIG_FILE" ]      && pass "$SIG_FILE present"      || fail "$SIG_FILE missing"
[ -f "$MANIFEST_FILE" ] && pass "$MANIFEST_FILE present" || fail "$MANIFEST_FILE missing"
echo

# --- step 4: SHA256SUMS internal consistency ---

echo "==> Step 4: SHA256SUMS internal consistency"
if [ -f "$SHA_FILE" ]; then
  for plat in "${EXPECTED_PLATFORMS[@]}"; do
    zip_name="${NAME}_${VERSION_NO_V}_${plat}.zip"
    if [ ! -f "$zip_name" ]; then continue; fi
    actual=$(sha256sum "$zip_name" | awk '{print $1}')
    listed=$(awk -v f="$zip_name" '$2 == f {print $1}' "$SHA_FILE")
    if [ -z "$listed" ]; then
      fail "$zip_name: not listed in SHA256SUMS"
    elif [ "$actual" != "$listed" ]; then
      fail "$zip_name: sha mismatch (actual=$actual, listed=$listed)"
    else
      pass "$zip_name: sha matches SHA256SUMS"
    fi
  done
fi
echo

# --- step 5: GPG signature (optional) ---

echo "==> Step 5: GPG signature"
if [ -z "$GPG_KEY" ]; then
  echo "  - skipped (no --gpg-key provided)"
elif [ ! -f "$GPG_KEY" ]; then
  fail "GPG key file not found: $GPG_KEY"
elif [ ! -f "$SHA_FILE" ] || [ ! -f "$SIG_FILE" ]; then
  fail "cannot verify signature (SHA256SUMS or .sig missing)"
else
  TMP_GPG_HOME=$(mktemp -d -t verify-gpg-XXXXXX)
  trap 'rm -rf "$WORK" "$TMP_GPG_HOME"' EXIT
  if gpg --homedir "$TMP_GPG_HOME" --import "$GPG_KEY" >/dev/null 2>&1; then
    if gpg --homedir "$TMP_GPG_HOME" --verify "$SIG_FILE" "$SHA_FILE" >/dev/null 2>&1; then
      pass "SHA256SUMS signature verifies against $GPG_KEY"
    else
      fail "SHA256SUMS signature does not verify against $GPG_KEY"
    fi
  else
    fail "could not import GPG key $GPG_KEY"
  fi
fi
echo

# --- step 6: manifest.json shape ---

echo "==> Step 6: manifest.json"
if [ -f "$MANIFEST_FILE" ]; then
  if jq empty "$MANIFEST_FILE" >/dev/null 2>&1; then
    pass "manifest is valid JSON"
    manifest_version=$(jq -r '.version' "$MANIFEST_FILE")
    if [ "$manifest_version" = "$VERSION_NO_V" ]; then
      pass "manifest version = $VERSION_NO_V"
    else
      fail "manifest version mismatch (got $manifest_version, want $VERSION_NO_V)"
    fi
    protocols=$(jq -r '.protocols // [] | join(",")' "$MANIFEST_FILE")
    if [ -n "$protocols" ]; then
      pass "manifest protocols = [$protocols]"
    else
      fail "manifest missing protocols"
    fi
    platforms=$(jq '.platforms // [] | length' "$MANIFEST_FILE")
    if [ "$platforms" = "${#EXPECTED_PLATFORMS[@]}" ]; then
      pass "manifest lists $platforms platforms"
    else
      fail "manifest lists $platforms platforms, want ${#EXPECTED_PLATFORMS[@]}"
    fi
  else
    fail "manifest is not valid JSON"
  fi
fi
echo

# --- step 7: zip contents ---

echo "==> Step 7: zip contents"
for plat in "${EXPECTED_PLATFORMS[@]}"; do
  zip_name="${NAME}_${VERSION_NO_V}_${plat}.zip"
  if [ ! -f "$zip_name" ]; then continue; fi

  entries=$(unzip -Z1 "$zip_name" 2>/dev/null | wc -l)
  if [ "$entries" != "1" ]; then
    fail "$zip_name: expected 1 file inside, got $entries"
    continue
  fi

  inside=$(unzip -Z1 "$zip_name")
  # Registry convention: the binary inside the zip carries the 'v' prefix
  # that `terraform init` and `tofu init` look for; the archive filename
  # itself uses the unprefixed version.
  expected="${NAME}_${TAG}"
  if [[ "$plat" == windows_* ]]; then
    expected="${expected}.exe"
  fi
  if [ "$inside" = "$expected" ]; then
    pass "$zip_name: contains $inside"
  else
    fail "$zip_name: contains $inside (want $expected)"
  fi
done
echo

# --- summary ---

TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo "==> Summary: $PASS_COUNT/$TOTAL checks passed"
if [ "$FAIL_COUNT" -gt 0 ]; then
  echo
  echo "Failures:"
  for f in "${FAILURES[@]}"; do
    echo "  - $f"
  done
  exit 1
fi
echo "All checks passed."
