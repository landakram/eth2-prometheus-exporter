{ nixpkgs ? import <nixpkgs> {} }:
let
  inherit (nixpkgs) pkgs;
in
with pkgs; stdenv.mkDerivation {
  name = "shell";
  buildInputs = [
    goreleaser
  ];
}

