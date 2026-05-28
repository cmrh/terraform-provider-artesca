# Releasing

How to cut a release of the ARTESCA Terraform provider.

The flow is **tag → CI builds → verify → promote or roll a new RC → optionally submit to OpenTofu Registry**. The intent is that every release goes through the same automated checks, so cutting one is a low-stakes operation.

## Versioning

We follow [Semantic Versioning](https://semver.org/):

- **Patch** (`v0.X.Y` → `v0.X.Y+1`) — bug fixes only.
- **Minor** (`v0.X.0` → `v0.X+1.0`) — new resources, new data sources, new optional fields, anything additive.
- **Major** (`v0.X.0` → `v1.0.0` later) — incompatible changes to existing resource schemas, removed fields, renamed resources. Avoid until the surface is stable.

Pre-releases use the `-rc.N` suffix: `v0.4.0-rc1`, `v0.4.0-rc2`, etc. Pre-releases are marked as such by the release workflow automatically (by detecting the `-` in the tag).

## Step-by-step

### 1. Update `CHANGELOG.md`

Move the `[Unreleased]` block to a versioned heading with today's date:

```markdown
## [Unreleased]

(blank — for future entries)

## [0.4.0] - 2026-05-28

### Added
- …
```

Open a PR with the changelog update. Merge it before tagging.

### 2. Cut a release candidate first

For anything bigger than a single bugfix patch, **start with an RC** so the artifacts can be verified end-to-end before the canonical version exists:

```sh
git tag v0.4.0-rc1
git push origin v0.4.0-rc1
```

Pushing a tag matching `v*` triggers `.github/workflows/release.yml`, which:

1. Runs the test job (unit tests, vet, gofmt, lint).
2. Builds binaries for all five platforms (`linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`).
3. Generates the SHA256SUMS file and GPG-signs it.
4. Writes the protocol manifest.
5. Uploads everything to the GitHub release for the tag.

For an `-rc*` tag, the release is automatically marked **prerelease** in the GitHub UI.

### 3. Verify the release

After the workflow finishes, run:

```sh
scripts/verify-release.sh v0.4.0-rc1
```

To also check the GPG signature, point at the public key file:

```sh
scripts/verify-release.sh v0.4.0-rc1 --gpg-key .github/RELEASE_KEY.asc
```

The script checks:

- All five platforms have a `.zip` + `.zip.sha256`.
- Filenames follow the registry convention (no `v` prefix on the version part of any filename).
- The aggregated `SHA256SUMS` file matches each zip's actual sha256.
- The GPG signature on `SHA256SUMS.sig` verifies against the provided public key.
- `manifest.json` is valid JSON with the right version, protocols, and platform list.
- Each zip contains exactly one binary, named correctly.

Exit code 0 means every check passed. Any failure prints a `Failures:` summary at the end.

### 4. Roll forward

- **All checks passed** → tag the canonical version once you're ready (`v0.4.0`), let CI build, re-run the verify script against the new tag.
- **Some checks failed** → fix in a regular PR, merge to `main`, then tag `v0.4.0-rc2` and repeat.

### 5. Submit to OpenTofu Registry (first publication only)

This step is only needed once per provider; subsequent versions get picked up automatically.

1. Make sure your GPG public key is uploaded to the Scality publisher account on `registry.opentofu.org`.
2. Open the [Provider Submission issue form](https://github.com/opentofu/registry/issues/new?assignees=&labels=provider%2Csubmission&projects=&template=provider.yml&title=Provider%3A+).
3. Fill in the repository field: `scality/terraform-provider-artesca`.
4. Wait for the OpenTofu Registry team to review and approve.

Subsequent tagged releases will be ingested without further action.

## Troubleshooting

**`verify-release.sh` flags legacy `v`-prefixed assets.** The `release.yml` workflow may have regressed. Check that `ARCHIVE_NAME`, `BINARY_NAME`, and the SHA256SUMS section all use `VERSION_NO_V` (not `VERSION`) for filenames.

**`verify-release.sh` flags missing assets.** The build matrix may have failed for one platform; check the workflow run in the GitHub Actions UI. Common culprits: cross-compilation breakage, runner availability.

**GPG signature verification fails.** Either the wrong public key was provided to `--gpg-key`, or the signing key in the CI secrets has rotated without the public key being updated.

**Manifest has the wrong protocols.** The `release.yml` writes `["6.0"]` literally. If the provider ever needs a different protocol version, update the workflow.

**A tag was pushed by mistake.** Delete it locally and on the remote: `git tag -d v0.X.Y && git push origin :refs/tags/v0.X.Y`. The GitHub release can be deleted from the UI. (This is allowed for pre-releases; avoid for canonical releases that have already been ingested by the registry.)

## What does NOT need to happen for a release

- No manual edits to `manifest.json` — the workflow generates it from the tag.
- No manual SHA256SUMS — the workflow aggregates per-platform sha256 files.
- No manual GPG signing — the workflow imports the key from secrets and signs.
- No README rewrite — the docs sources in `docs/` and `README.md` are the source of truth; they get rebuilt by the registry on ingestion.
