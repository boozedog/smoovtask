{
  description = "smoovtask – task management for Claude Code";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      eachSystem = nixpkgs.lib.genAttrs [
        "aarch64-linux"
        "x86_64-linux"
        "aarch64-darwin"
        "x86_64-darwin"
      ];
    in
    {
      packages = eachSystem (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          st = pkgs.buildGoModule {
            pname = "st";
            version = "0.1.0-dev";
            src = ./.;
            subPackages = [ "cmd/st" ];
            vendorHash = "sha256-3N8MPD+DI5ZcrhQ1vGCMCJqzj1oKt6cEKtKJd1zmhdg=";
          };
          default = self.packages.${system}.st;
        }
      );
    };
}
