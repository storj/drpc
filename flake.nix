{
  inputs = {
    nixpkgs.url = "https://flakehub.com/f/NixOS/nixpkgs/*.tar.gz";
    flake-utils.url = "https://flakehub.com/f/numtide/flake-utils/*.tar.gz";
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
          vendorHash = "sha256-NMa9c+QIq9VEUQZqZ5X9fNbZDJT99q8XNCH2rRKyMzQ=";
        };

        devShell =
          let devtools = {
            staticcheck = buildGoModule {
              name = "staticcheck";
              src = fetchFromGitHub {
                owner = "dominikh";
                repo = "go-tools";
                rev = "2025.1.1";
                sha256 = "sha256-ekSOXaVSFdzM76tcj1hbtzhYw4fnFX3VkTnsGtJanXg=";
              };
              doCheck = false;
              subPackages = [ "cmd/staticcheck" ];
              vendorHash = "sha256-HssfBnSKdVZVgf4f0mwsGTwhiszBlE2HmDy7cvyvJ60=";
            };

            golangci-lint = buildGoModule rec {
              name = "golangci-lint";
              version = "2.1.1";
              src = fetchFromGitHub {
                owner = "golangci";
                repo = "golangci-lint";
                rev = "v${version}";
                hash = "sha256-yOsXQmOmhCHDR5Q6ngyk6Lax5SKm81sdHglgTxxMS9w=";
              };
              doCheck = false;
              subPackages = [ "cmd/golangci-lint" ];
              vendorHash = "sha256-g7VievbIxKCbdy00kv7JcOggvVnfMEzuw5cnLOc/oVc=";
            };

            ci = buildGoModule {
              name = "ci";
              src = fetchFromGitHub {
                owner = "storj";
                repo = "ci";
                rev = "79f1c57325dc26b2b2862a518f69d1bb3ac8333f";
                sha256 = "sha256-JDSZ/YTyzNNw16E3z9mk/DjswVSa19BbPnRVwQRye2M=";
              };
              doCheck = false;
              vendorHash = "sha256-W/ZVtwo1BUIvJ9bCR4p5lBa2edyQQaSmnWqwMe0bOhI=";
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
              doCheck = false;
              vendorHash = "sha256-KHd9zmNsmXmc2+NNtTnw/CSkmGwcBVYNrpEUmIoZi5Q=";
              modRoot = "./cmd/protoc-gen-go-grpc";
            };

            protoc-gen-go = buildGoModule {
              name = "protoc-gen-go";
              src = fetchFromGitHub {
                owner = "protocolbuffers";
                repo = "protobuf-go";
                rev = "v1.27.1";
                sha256 = "sha256-wkUvMsoJP38KMD5b3Fz65R1cnpeTtDcVqgE7tNlZXys=";
              };
              doCheck = false;
              vendorHash = null;
              modRoot = "./cmd/protoc-gen-go";
            };

            protoc-gen-gogo = buildGoPackage {
              name = "protoc-gen-gogo";
              src = fetchFromGitHub {
                owner = "gogo";
                repo = "protobuf";
                rev = "v1.3.2";
                sha256 = "sha256-CoUqgLFnLNCS9OxKFS7XwjE17SlH6iL1Kgv+0uEK2zU=";
              };
              doCheck = false;
              goPackagePath = "github.com/gogo/protobuf";
              subPackages = [ "./protoc-gen-gogo" ];
            };

            protoc-gen-twirp = buildGoPackage {
              name = "protoc-gen-twirp";
              src = fetchFromGitHub {
                owner = "twitchtv";
                repo = "twirp";
                rev = "v8.1.3";
                sha256 = "sha256-p3gHVHGBHakOOQnJAuMK7vZumNXN15mOABuEHUG0wNs=";
              };
              doCheck = false;
              goPackagePath = "github.com/twitchtv/twirp";
              subPackages = [ "./protoc-gen-twirp" ];
            };

            stringer = buildGoModule {
              name = "stringer";
              src = fetchFromGitHub {
                owner = "golang";
                repo = "tools";
                rev = "v0.3.0";
                sha256 = "sha256-UMEhFxODGQ20vkZPtscBpHhUDa6/+hnD85Z1yx0pQfQ=";
              };
              doCheck = false;
              vendorHash = "sha256-EQHYf4Q+XNjwG/KDoTA4m0mlBGxPkJSLUcO0VHFSpeA=";
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
              doCheck = false;
              goPackagePath = "github.com/robertkrimen/godocdown";
              subPackages = [ "./godocdown" ];
            };
          };
        in mkShell {
            buildInputs = [
              defaultPackage

              go_1_24
              protobuf
              graphviz
              bash
              gnumake

              devtools.protoc-gen-go-grpc
              devtools.protoc-gen-go
              devtools.protoc-gen-gogo
              devtools.protoc-gen-twirp
              devtools.staticcheck
              devtools.golangci-lint
              devtools.ci
              devtools.stringer
              devtools.godocdown
            ];
          };
      }
    );
}
