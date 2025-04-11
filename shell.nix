{ pkgs ? import <nixpkgs> { } }:
with pkgs; 
mkShell {
    nativeBuildInputs = [
        glib
    ];

    buildInputs = [
        fastfetch
    ];
}