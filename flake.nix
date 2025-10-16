{
  description = "Dev environment for Horsa";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };


outputs = { self, nixpkgs, ... }:
    let
      pkgs = import nixpkgs { system = "x86_64-linux"; };
    in
    {
      devShells.x86_64-linux.default = pkgs.mkShell {
        packages = with pkgs; [
          go_1_25

          nodejs_20
          nodePackages.npm

          git
        ];

        env = {
          GOFLAGS = "-buildvcs=false";
          PATH = "$HOME/go/bin:${pkgs.go_1_25}/bin:$PATH";
        };

        shellHook = ''
          export PATH="$HOME/go/bin:$PATH"
        '';
      };
    };
}