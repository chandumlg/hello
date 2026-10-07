# Nix (and Nix Flakes)

Nix is a purely functional package manager and build system that describes every dependency of a build or environment declaratively and content-addresses the results, so the same inputs produce the same outputs on any machine — ending "works on my machine" drift between laptops, CI, and production images.

## Why teams adopt it

Nix treats packages as immutable values stored in `/nix/store/<hash>-name`, where the hash covers all inputs (source, compiler, flags, transitive deps). Multiple versions coexist, upgrades are atomic, and rollbacks are instant. Flakes add a standard, lockfile-pinned entry point (`flake.nix` + `flake.lock`) so an entire toolchain is pinned to exact revisions.

Platform and staff engineers adopt it when:
- **Dev environments must be reproducible.** `nix develop` gives every engineer (and CI) identical compilers, CLIs, and system libraries, with no onboarding wiki of install steps.
- **CI and local builds should match.** The same flake that powers local dev runs in CI, and binary caches (Cachix, Attic, S3) make repeat builds fast.
- **Container images need to be minimal and deterministic.** `dockerTools` builds layered OCI images containing only the closure of your app, with no base-image drift.
- **Fleet or host configuration needs to be declarative.** NixOS, nix-darwin, and home-manager apply the same model to OS and dotfile config.
- **Polyglot monorepos need a hermetic build layer** without adopting Bazel's full model.

## Basic usage

**1. Try tools without installing them, and install Nix with flakes enabled:**
```bash
# Installer that enables flakes by default (Determinate Systems)
curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install

# Run a tool ephemerally, pinned via the nixpkgs registry
nix run nixpkgs#jq -- --version
nix shell nixpkgs#ripgrep nixpkgs#fd -c rg --version
```

**2. A reproducible dev shell for a project (`flake.nix`):**
```nix
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let pkgs = nixpkgs.legacyPackages.${system}; in {
        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_23 pkgs.golangci-lint pkgs.postgresql_16 pkgs.terraform ];
          shellHook = ''echo "dev shell ready"'';
        };
      });
}
```
```bash
git add flake.nix      # flakes only see git-tracked files
nix develop            # enter the shell; writes flake.lock
nix flake update       # deliberately bump pins
```
Pair with `direnv` and `use flake` in `.envrc` to enter the environment automatically on `cd`.

**3. Build a package and a minimal container image from the same flake:**
```nix
packages.default = pkgs.buildGoModule {
  pname = "myservice"; version = "0.1.0"; src = ./.;
  vendorHash = "sha256-AAAA...";   # nix prints the real hash on first failure
};
packages.image = pkgs.dockerTools.buildLayeredImage {
  name = "myservice";
  config.Cmd = [ "${self.packages.${system}.default}/bin/myservice" ];
};
```
```bash
nix build .#default && ./result/bin/myservice
nix build .#image && docker load < result
nix flake check        # run checks defined in the flake (tests, lint) in CI
```

## Pitfalls

- **Steep learning curve.** The Nix language is lazy, dynamically typed, and has poor error messages. Budget for a few champions; don't mandate it team-wide on day one.
- **Flakes are still "experimental"** in upstream Nix. The interface is widely used and effectively stable, but you must enable `experimental-features = nix-command flakes`, and the ecosystem has forks (Lix, Determinate Nix) with slightly different behavior.
- **Untracked files are invisible to flakes.** A new file not `git add`ed causes baffling "file not found" errors.
- **Hash churn.** `vendorHash`, `npmDepsHash`, and `cargoHash` must be updated when lockfiles change. Use `lib.fakeHash` and copy the printed value, or tools like `nix-update`.
- **Non-Nix-friendly binaries.** Prebuilt binaries expect `/lib64/ld-linux`; use `autoPatchelfHook`, `nix-ld`, or build from source. Language ecosystems with install-time downloads (npm, pip) need the sandbox-friendly wrappers (`buildNpmPackage`, `uv2nix`, `poetry2nix`, `crane`).
- **Disk growth.** Old generations and store paths accumulate; schedule `nix-collect-garbage --delete-older-than 14d` and set up a binary cache so CI doesn't rebuild the world.
- **Pin nixpkgs deliberately.** Tracking `nixos-unstable` gets you fresh packages but occasional breakage; use a stable release branch for production-facing builds and update via PRs reviewed like any dependency bump.
- **macOS caveats.** The sandbox is weaker and some packages lag Linux; Apple Silicon support is good but not identical.
