{
  pkgs,
  lib,
  config,
  ...
}:

# Development environment for Eudaimonia: a Go node + React client.
# `devenv shell` gives you the toolchain; `devenv up` runs Postgres, LiveKit,
# the backend (with live reload) and the Vite dev server together.
# Production images are built from backend/Dockerfile and frontend/Dockerfile.
{
  name = "eudaimonia";

  # Optional overrides live in a git-ignored .env (see .env.example).
  dotenv.enable = true;

  # Development defaults. They match the fallbacks compiled into the backend
  # (backend/internal/config) and the defaults used by compose.yaml.
  # Lower priority than dotenv's lib.mkDefault so values in .env win.
  env = lib.mapAttrs (_: v: lib.mkOverride 1100 v) {
    BACKEND_ADDR = ":8080";
    NODE_NAME = "dev-node";
    LOG_LEVEL = "debug";
    DATABASE_URL = "postgres://eudaimonia:eudaimonia@127.0.0.1:${toString config.services.postgres.port}/eudaimonia?sslmode=disable";
    LIVEKIT_URL = "ws://127.0.0.1:7880";
    LIVEKIT_API_KEY = "devkey";
    LIVEKIT_API_SECRET = "secret";
  };

  # https://devenv.sh/packages/
  packages = with pkgs; [
    git
    sqlc # typed SQL -> Go
    goose # migrations (see backend/migrations)
    golangci-lint
    air # live reload for the backend
    livekit # livekit-server binary for local voice
    livekit-cli # lk: token generation, load testing
    hadolint # Dockerfile linter
    nixfmt-rfc-style
  ];

  # https://devenv.sh/languages/
  languages = {
    go = {
      enable = true;
      package = pkgs.go_1_27;
    };
    javascript = {
      enable = true;
      package = pkgs.nodejs_26; # Active LTS
      directory = "${config.devenv.root}/frontend";
      npm.enable = true;
      npm.install.enable = true; # keep frontend/node_modules in sync on shell entry
    };
    typescript.enable = true;
    nix.enable = true;
  };

  # https://devenv.sh/services/
  services.postgres = {
    enable = true;
    package = pkgs.postgresql_18;
    listen_addresses = "127.0.0.1";
    port = 5432;
    initialDatabases = [
      {
        name = "eudaimonia";
        user = "eudaimonia";
        pass = "eudaimonia";
      }
    ];
  };

  # https://devenv.sh/processes/  (started together by `devenv up`)
  processes = {
    backend = {
      exec = "cd ${config.devenv.root}/backend && air";
      process-compose.depends_on.postgres.condition = "process_healthy";
    };
    frontend.exec = "cd ${config.devenv.root}/frontend && npm run dev";
    # Dev mode uses the devkey/secret pair above and binds 127.0.0.1:7880.
    livekit.exec = "livekit-server --dev --bind 127.0.0.1";
  };

  # https://devenv.sh/scripts/
  scripts = {
    db-migrate.exec = ''goose -dir "$DEVENV_ROOT/backend/migrations" postgres "$DATABASE_URL" up "$@"'';
    db-status.exec = ''goose -dir "$DEVENV_ROOT/backend/migrations" postgres "$DATABASE_URL" status'';
    backend-test.exec = ''cd "$DEVENV_ROOT/backend" && gofmt -l . && go vet ./... && go test ./... "$@"'';
    frontend-build.exec = ''cd "$DEVENV_ROOT/frontend" && npm run build'';
    lint-docker.exec = ''hadolint "$DEVENV_ROOT/backend/Dockerfile" "$DEVENV_ROOT/frontend/Dockerfile"'';
    stack-up.exec = ''docker compose -f "$DEVENV_ROOT/compose.yaml" up --build "$@"'';
  };

  # https://devenv.sh/git-hooks/
  git-hooks.hooks = {
    gofmt.enable = true;
    golangci-lint = {
      enable = true;
      files = "^backend/.*\\.go$";
      pass_filenames = false;
      entry = toString (
        pkgs.writeShellScript "golangci-lint-backend" ''
          # Works from a plain git commit too, where Go is not on PATH.
          export PATH=${config.languages.go.package}/bin:$PATH
          cd "$(${lib.getExe pkgs.git} rev-parse --show-toplevel)/backend"
          exec ${lib.getExe pkgs.golangci-lint} run ./...
        ''
      );
    };
    prettier = {
      enable = true;
      files = "^frontend/.*\\.(ts|tsx|css|html|json)$";
      excludes = [ "^frontend/package-lock\\.json$" ];
    };
    nixfmt.enable = true;
    hadolint.enable = true;
  };

  enterShell = ''
    echo "eudaimonia dev shell"
    echo "  go $(go version | cut -d' ' -f3)  node $(node --version)  postgres ${config.services.postgres.package.version}"
    echo "  devenv up        start postgres, livekit, backend (air) and vite"
    echo "  backend-test     gofmt + vet + test the Go node"
    echo "  frontend-build   type-check and bundle the React app"
    echo "  stack-up         docker compose up --build (needs Docker)"
  '';

  # https://devenv.sh/tests/  (`devenv test`)
  enterTest = ''
    backend-test
    frontend-build
    lint-docker
  '';
}
