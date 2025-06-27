{
  description = "Online Compiler API Gateway (Go + Kafka + WebSocket)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        packages.default = pkgs.buildGoModule {
          pname = "online-compiler-api-gateway";
          version = "0.1.0";
          src = ./.;
          vendorSha256 = null; # run `nix build` to get hash and replace this

          # if you want to build only main.go manually, set subPackages = [ "." ];
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.gopls
            pkgs.delve
            pkgs.kafka
            pkgs.git
          ];
        };
      }
    );
}
