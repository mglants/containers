{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  packages = with pkgs; [
    nodejs_22
  ];

  shellHook = ''
    echo "Nix shell ready for RW Temporary Subscription App"
    echo "Node: $(node --version)"
    echo "npm: $(npm --version)"
  '';
}
