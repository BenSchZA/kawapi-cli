{ pkgs ? import <nixpkgs> {} }:

with pkgs;

let
  elixir = beam.packages.erlangR22.elixir_1_9;
in
mkShell {
  buildInputs = [ elixir erlangR22 protobuf ];
  shellHook = ''
    protoc --elixir_out=plugins=grpc:. *.proto
    # protoc -I ../helloworld --go_out=plugins=grpc:../helloworld ../helloworld/helloworld.proto
  '';
}
