{
  description = "weftlo — Go development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            gopls
            gotools
            go-tools
            golangci-lint
            delve
            gnumake
          ];

          # Intentionally do NOT override GOPATH into the project tree:
          # tools like `gofmt -l .` and the Makefile's `fmt-check` would
          # otherwise scan the module cache and report thousands of files.
          # Go modules already provide per-project isolation via go.mod.
        };
      });
}
