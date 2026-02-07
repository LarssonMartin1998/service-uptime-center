{
  description = "service-uptime-center, configure your service groups with required OK status intervals and notification systems when something doesn't go as planned.";

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
        go = pkgs.go;
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "service-uptime-center";
          version = "0.1";
          src = ./.;

          vendorHash = "sha256-XXgXzv6MARTUse1lf4RAaMp9xg8FfysaPMM7wq5zdlw=";

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
            echo "🦫 $(${go}/bin/go version) ready!"
          '';
        };
      }
    );
}
