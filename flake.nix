{
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
          vendorHash = "sha256-MfdImlAHGHO1rxQN1YjvnCWpiTHRA1p1K9XO8nl/qY8=";
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
          devShell = pkgs.mkShell {
            packages = with pkgs; [
              gopls
              gotools
              go_1_25
            ];
          };
        };
    in
    flake-utils.lib.eachDefaultSystem (system: flakeForSystem nixpkgs system);
}
