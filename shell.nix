# Development shell for the CoreDNS ConsulKV plugin.
# Usage: nix-shell   (then `go build ./...`, `go test ./...`)
{ pkgs ? import <nixpkgs> { } }:

pkgs.mkShell {
  packages = with pkgs; [
    go
  ];

  shellHook = ''
    export GOPATH="$PWD/.gopath"
    export PATH="$GOPATH/bin:$PATH"
    echo "Go dev shell: $(go version)"
  '';
}
