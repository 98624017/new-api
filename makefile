FRONTEND_DIR = ./web
BACKEND_DIR = .

.PHONY: all build-frontend start-backend sync-upstream-local

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && bun install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

sync-upstream-local:
	@bash scripts/sync_upstream_local.sh
