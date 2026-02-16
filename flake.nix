{
  description = "vpngate - VPN server connector";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      vpngate =
        pkgs:
        pkgs.buildGo125Module rec {
          name = "vpngate";
          src = ./.;
          vendorHash = "sha256-FNpeIIIrINm/3neCkuX/kFWWGCCEN8Duz1iSFAki+54=";
          nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.makeWrapper ];
          env.CGO_ENABLED = 0;
          doCheck = false;
        };

      flakeForSystem =
        nixpkgs: system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          vg = vpngate pkgs;
        in
        {
          packages = {
            default = vg;
            vpngate = vg;
          };
          devShells.default = pkgs.mkShell {
            name = "vpngate-dev";
            description = "Development environment for vpngate";
            packages = with pkgs; [
              go_1_26
              gopls
              gotools
              golangci-lint
            ];
            shellHook = ''
              echo "Welcome to the vpngate dev environment"
              go version
            '';
          };
        };
    in
    flake-utils.lib.eachDefaultSystem (system: flakeForSystem nixpkgs system);
}
