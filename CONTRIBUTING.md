# Contributing

Thanks for your interest in improving the ARTESCA Terraform provider.

## Reporting issues

Open a GitHub issue with a clear description, the provider version, and the
shortest config that reproduces the problem. Include `TF_LOG=DEBUG` output if
the issue is a wire-level error.

For security issues, see [SECURITY.md](SECURITY.md) — please do not open a
public issue.

## Development setup

You'll need:

- Go 1.26+
- OpenTofu (preferred) or Terraform
- Access to an ARTESCA cluster for acceptance tests

Build and install the provider locally:

```bash
make install VERSION=0.4.1-dev
```

This drops the binary at
`~/.terraform.d/plugins/registry.terraform.io/cmrh/artesca/<VERSION>/<OS>_<ARCH>/`.

For interactive development, configure dev_overrides in `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "cmrh/artesca" = "/path/to/your/built/binary/dir"
  }
  direct {}
}
```

With dev_overrides active, `tofu init` is **not needed** (and won't work) —
go straight to `tofu plan`. No `.terraform.lock.hcl` is needed either.

## Tests

Unit tests run without infrastructure:

```bash
make test
```

Acceptance tests provision real ARTESCA resources and require credentials in
your environment:

```bash
export ARTESCA_MANAGEMENT_ENDPOINT=...
export ARTESCA_OIDC_URL=...
export ARTESCA_USERNAME=...
export ARTESCA_PASSWORD=...
export ARTESCA_S3_ENDPOINT=...
make testacc
```

Always add or extend acceptance tests when adding a resource or behavior.

## Style

- `make fmt` and `make lint` must pass.
- Comments: one short line max. Skip if the name is self-explanatory.
- For documentation, prefer "always do X to ensure Y" framing over "if you hit
  problem Z, fix it with X".

## Submitting changes

1. Fork the repo and create a topic branch.
2. Commit with descriptive messages.
3. Open a PR against `main` with a summary of the change and a test plan.
4. Make sure CI is green.

## Architectural conventions

The provider talks to three distinct API surfaces (Management, IAM, S3) — see
[ARCHITECTURE.md](ARCHITECTURE.md). Bucket sub-features (encryption, policy,
tagging) and workflows (expiration, transition, replication) are separate
resources, following the AWS-provider modular pattern. Don't propose folding
them back into `artesca_bucket` — for one-stop UX, write a wrapper module
instead.
