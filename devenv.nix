{ pkgs, ... }:

# devenv.sh dev environment for the Cycloid Terraform provider.
# One reproducible toolchain shared by local dev (`devenv shell` / `direnv allow`)
# and CI (the self-hosted [self-hosted, cycloid] runner enters this same shell).
#
# Goal: matching Go, lint, docs, and IaC tooling so a green local run == a green CI run.
{
  # Go must match go.mod (go 1.25.0). nixpkgs ships go_1_25 as the 1.25.x line.
  #
  # NOTE: we intentionally do NOT use `languages.go.enable` here. That module
  # force-builds a fixed set of IDE helpers (gopls, gotests, gomodifytags, impl,
  # iferr, go-tools) against the pinned Go. On the rolling nixpkgs some of those
  # tools' sources now require go >= 1.26, so building them with go_1_25 fails
  # (`go.mod requires go >= 1.26.0`) and breaks BOTH `devenv shell` and the CI
  # `nix develop`. CI/lint/vet/build/test only need the `go` binary plus the
  # explicitly-listed tools below, so we add go directly and skip the IDE set.
  packages = with pkgs; [
    # --- Go toolchain ---
    go_1_25 # matches go.mod (go 1.25.x); GOROOT set in enterShell

    # --- Go tooling ---
    golangci-lint # lint-build job: `golangci-lint run`
    gotools # goimports, etc.
    delve # interactive debugging

    # --- Terraform provider tooling ---
    terraform-plugin-docs # `tfplugindocs` — docs generation / drift check
    opentofu # OSS terraform for plan/apply against the dev override
    terraform # kept for parity with the registry build

    # --- Task runner / stack ---
    just # Justfile is the canonical command runner
    docker-compose # `docker compose` client (daemon provided by host/runner)

    # --- misc helpers used by the Justfile / bootstrap ---
    jq # used by the youdeploy-api healthcheck / scripts
    curl
    git
    gnumake # Makefile wraps just
  ];

  # `terraform` from nixpkgs is BSL-licensed; allow it explicitly so the shell builds.
  # (opentofu is the preferred runner; terraform is here only for registry parity.)
  env.NIXPKGS_ALLOW_UNFREE = "1";

  enterShell = ''
    echo "cycloid terraform-provider dev shell"
    echo "  go:            $(go version | awk '{print $3}')"
    echo "  golangci-lint: $(golangci-lint version --short 2>/dev/null || golangci-lint --version | head -1)"
    echo "  tfplugindocs:  $(command -v tfplugindocs >/dev/null && echo ok || echo MISSING)"
    echo "  just / tofu:   $(command -v just >/dev/null && echo ok) / $(tofu version | head -1 2>/dev/null)"
    echo
    echo "  just help     # list targets"
    echo "  just test-unit / just be-start / just test-acc"
  '';
}
