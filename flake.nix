{
  description = "CoreDNS ConsulKV plugin - Go development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = with pkgs; [
            go
          ];

          shellHook = ''
            export GOPATH="$PWD/.gopath"
            export PATH="$GOPATH/bin:$PATH"
            echo "Go dev shell: $(go version)"
          '';
        };
      });
    };
}
