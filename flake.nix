{
  description = "TrueNAS Terraform Provider development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
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
            # Go toolchain (matches go.mod 1.24)
            go_1_24

            # Linting & formatting
            golangci-lint

            # OpenTofu (Terraform-compatible, open-source)
            opentofu

            # QEMU (for VM-based acceptance tests)
            qemu
            socat

          ];

          shellHook = ''
            echo "terraform-provider-truenas dev shell"
          '';
        };
      }
    );
}
