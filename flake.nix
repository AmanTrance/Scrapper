{
  description = "Cron Scraper";

  inputs = {
    nixpkgs = { 
      url = "github:nixos/nixpkgs?ref=nixos-24.11";
    };
  };

  outputs = { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
      ];

      forAllSystems = (f: nixpkgs.lib.genAttrs systems (system: f {
        pkgs = import nixpkgs { inherit system; };
      }));
    in
    {
      packages = forAllSystems ({ pkgs }: {
        default = pkgs.buildGoModule {
          name = "scraper";
          src = self;
          vendorHash = "sha256-EEzNj22RvQF8mO0R/stydHltbRzHkzPixnH7+r9y5+o=";
          subPackages = [ 
            "./" 
          ];
        };
      });
    };
}