{
  description = "k3k-tui - Terminal UI for managing k3k virtual clusters";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = "0.1.0";
      in
      {
        packages = {
          default = self.packages.${system}.k3k-tui;

          k3k-tui = pkgs.buildGoModule {
            pname = "k3k-tui";
            inherit version;
            src = ./.;

            # First build will fail with the correct hash — replace this value:
            vendorHash = pkgs.lib.fakeHash;

            ldflags = [
              "-s" "-w"
              "-X main.version=v${version}"
              "-X main.commitHash=${self.shortRev or "dirty"}"
              "-X main.buildTime=1970-01-01T00:00:00Z"
            ];

            meta = with pkgs.lib; {
              description = "Terminal UI for managing k3k (Kubernetes-in-Kubernetes) virtual clusters";
              homepage = "https://github.com/malagant/k3k-tui";
              license = licenses.mit;
              maintainers = [ ];
              mainProgram = "k3k-tui";
            };
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls
            gotools
            go-tools      # staticcheck
            delve         # debugger
            kubectl
            k9s
          ];

          shellHook = ''
            echo "🚀 k3k-tui dev shell"
            echo "   go version: $(go version | awk '{print $3}')"
            echo "   build:      go build -o k3k-tui ."
            echo "   run:        go run ."
            echo "   test:       go test ./..."
          '';
        };
      }
    );
}
