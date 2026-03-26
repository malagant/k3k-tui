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
            starship
          ];

          shellHook = ''
            # Merge k3k-tui starship config with user's existing config
            _user_config="''${STARSHIP_CONFIG:-$HOME/.config/starship.toml}"
            _merged="/tmp/k3k-tui-starship-$$.toml"

            {
              [ -f "$_user_config" ] && cat "$_user_config"
              echo ""
              cat "$PWD/.config/starship.toml"
            } > "$_merged"

            export STARSHIP_CONFIG="$_merged"
            trap "rm -f '$_merged'" EXIT

            # Init starship for current shell
            if [ -n "$ZSH_VERSION" ]; then
              eval "$(starship init zsh)"
            elif [ -n "$BASH_VERSION" ]; then
              eval "$(starship init bash)"
            elif [ -n "$FISH_VERSION" ]; then
              starship init fish | source
            fi

            echo "🚀 k3k-tui dev shell"
            echo "   go:       $(go version | awk '{print $3}')"
            echo "   starship: $(starship --version | head -1)"
            echo "   build:    go build -o k3k-tui ."
            echo "   run:      go run ."
            echo "   test:     go test ./..."
          '';
        };
      }
    );
}
