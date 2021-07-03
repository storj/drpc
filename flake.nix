{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      with nixpkgs.legacyPackages.${system}; rec {
        defaultPackage = buildGoModule rec {
          name = "protoc-gen-go-drpc";
          src = builtins.path {
            path = ./.;
            name = "${name}-src";
            filter = (path: type: builtins.elem path (builtins.map toString [
              ./cmd
              ./cmd/protoc-gen-go-drpc
              ./cmd/protoc-gen-go-drpc/main.go
              ./go.mod
              ./go.sum
            ]));
          };
          subPackages = [ "cmd/protoc-gen-go-drpc" ];
          vendorSha256 = "sha256-gE5b0cmq4lHEY1Ar0dCERbFLRvptNESZqija4Ruw9z0=";
        };

        devShell =
          let devtools = {
            staticcheck = buildGoModule {
              name = "staticcheck";
              src = fetchFromGitHub {
                owner = "dominikh";
                repo = "go-tools";
                rev = "v0.2.0";
                sha256 = "sha256-QhTjzrERhbhCSkPzyLQwFyxrktNoGL9ris+XfE7n5nQ=";
              };
              doCheck = false;
              subPackages = [ "cmd/staticcheck" ];
              vendorSha256 = "sha256-EjCOMdeJ0whp2pHZvm4VV2K78UNKzl98Z/cQvGhWSyY=";
            };

            ci = buildGoModule {
              name = "ci";
              src = fetchFromGitHub {
                owner = "storj";
                repo = "ci";
                rev = "e92f7f42d44a515670339331b652aa8a7516c390";
                sha256 = "sha256-n2Rytcuaffy9ftzDT1Nrmi2RWiyWDOf6B4qvIWiVz7M=";
              };
              vendorSha256 = "sha256-6D452YbnkunAfD/M69VmwGDxENmVS72NKj92FTemJR0=";
              doCheck = false;
              allowGoReference = true; # until check-imports stops needing this
              subPackages = [
                "check-copyright"
                "check-large-files"
                "check-imports"
                "check-atomic-align"
                "check-errs"
              ];
            };

            protoc-gen-go-grpc = buildGoModule {
              name = "protoc-gen-go-grpc";
              src = fetchFromGitHub {
                owner = "grpc";
                repo = "grpc-go";
                rev = "v1.36.0";
                sha256 = "sha256-sUDeWY/yMyijbKsXDBwBXLShXTAZ4445I4hpP7bTndQ=";
              };
              vendorSha256 = "sha256-KHd9zmNsmXmc2+NNtTnw/CSkmGwcBVYNrpEUmIoZi5Q=";
              doCheck = false;
              modRoot = "./cmd/protoc-gen-go-grpc";
            };

            protoc-gen-go = buildGoModule {
              name = "protoc-gen-go";
              src = fetchFromGitHub {
                owner = "protocolbuffers";
                repo = "protobuf-go";
                rev = "v1.26.0";
                sha256 = "sha256-n2LHI8DXQFFWhTPOFCegBgwi/0tFvRE226AZfRW8Bnc=";
              };
              vendorSha256 = "sha256-pQpattmS9VmO3ZIQUFn66az8GSmB4IvYhTTCFn6SUmo=";
              doCheck = false;
              modRoot = "./cmd/protoc-gen-go";
            };

            stringer = buildGoModule {
              name = "stringer";
              src = fetchFromGitHub {
                owner = "golang";
                repo = "tools";
                rev = "v0.1.4";
                sha256 = "sha256-7iQZvA6uUjZLP3/dxaM9y9jomSwEoaUgGclnciF8rh4=";
              };
              vendorSha256 = "sha256-PRC59obp0ptooFuWhg2ruihEfJ0wKeMyT9xcLjoZyCo=";
              doCheck = false;
              subPackages = [ "cmd/stringer" ];
            };

            godocdown = buildGoPackage {
              name = "godocdown";
              src = fetchFromGitHub {
                owner = "robertkrimen";
                repo = "godocdown";
                rev = "0bfa0490548148882a54c15fbc52a621a9f50cbe";
                sha256 = "sha256-5gGun9CTvI3VNsMudJ6zjrViy6Zk00NuJ4pZJbzY/Uk=";
              };
              goPackagePath = "github.com/robertkrimen/godocdown";
              subPackages = [ "./godocdown" ];
            };
          };
        in mkShell {
            buildInputs = [
              defaultPackage

              go
              golangci-lint
              protobuf
              graphviz
              bash
              gnumake

              devtools.protoc-gen-go-grpc
              devtools.protoc-gen-go
              devtools.staticcheck
              devtools.ci
              devtools.stringer
              devtools.godocdown
            ];
          };
      }
    );
}
