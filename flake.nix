{
  description = "vpngate - VPN server connector";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);

      vpngate =
        pkgs:
        pkgs.buildGo126Module {
          name = "vpngate";
          src = ./.;
          vendorHash = "sha256-z6b7BPfcmt2PrCQvDL1vyvbCqVE4Uauo4uI6RLiuf5o=";
          nativeBuildInputs = pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.makeWrapper ];
          env.CGO_ENABLED = 0;
          doCheck = false;
        };
    in
    {
      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          vg = vpngate pkgs;
        in
        {
          default = vg;
          vpngate = vg;
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            name = "vpngate-dev";
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
        }
      );
    };
}
