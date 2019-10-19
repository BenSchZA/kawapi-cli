with import <nixpkgs> { };

stdenv.mkDerivation {
  name = "go";
  buildInputs = [ go gcc ];
  shellHook = "";
}
