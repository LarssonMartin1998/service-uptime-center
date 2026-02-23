{
  description = "service-uptime-center with NixOS module";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "service-uptime-center";
          version = "0.1";
          src = ./.;
          vendorHash = "sha256-o61u+DK+UwID1SAoxy5kKto7oz7tFV6EAE5xcuBk32A=";
          doCheck = true;
          checkPhase = ''
            go test ./...
          '';
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            go
            delve
            gotools
          ];
          shellHook = ''
            echo "🦫 $(${pkgs.go}/bin/go version) ready!"
          '';
        };
      }
    )
    // {
      nixosModules.default = import ./service-uptime-center.nix;
    };
}
