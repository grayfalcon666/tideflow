.PHONY: help up down migrate api worker test build clean

# Colors
GREEN  := \033[32m
YELLOW := \033[33m
NC     := \033[0m

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-12s$(NC) %s\n", $$1, $$2}'

# ==================== Infrastructure ====================
up: ## Start all infrastructure services
	docker compose up -d mysql redis rabbitmq elasticsearch kibana

down: ## Stop all infrastructure services
	docker compose down

# ==================== Backend ====================
migrate: ## Run database migrations
	cd backend && go run cmd/migrate/main.go

api: ## Start API server (http://localhost:8080)
	cd backend && go run cmd/api/main.go

worker: ## Start background worker
	cd backend && go run cmd/worker/main.go

test: ## Run all tests with race detection
	cd backend && go test -race -cover ./...

build: ## Build API binary
	cd backend && go build -o tideflow-api cmd/api/main.go

build-worker: ## Build Worker binary
	cd backend && go build -o tideflow-worker cmd/worker/main.go

clean: ## Clean build artifacts
	rm -f backend/tideflow-api backend/tideflow-worker
	rm -rf backend/uploads/*

# ==================== Development ====================
dev-api: ## Run API with hot reload (requires air)
	cd backend && air

fmt: ## Format Go code
	cd backend && go fmt ./... && goimports -w .

lint: ## Run linters
	cd backend && golint ./... && go vet ./...

# ==================== Database ====================
db-reset: ## Reset database (WARNING: deletes all data)
	docker compose exec mysql mysql -uroot -ppassword -e "DROP DATABASE IF EXISTS tideflow; CREATE DATABASE tideflow;"
	$(MAKE) migrate

# ==================== Utilities ====================
go-mod-tidy: ## Tidy Go modules
	cd backend && go mod tidy

go-get: ## Get a dependency (e.g., make go-get PKG=github.com/gin-gonic/gin)
	cd backend && go get $(PKG)

logs-api: ## Tail API logs
	docker compose logs -f api

logs-worker: ## Tail Worker logs
	docker compose logs -f worker

# ==================== Elasticsearch ====================
es-health: ## Check ES cluster health
	@curl -s http://localhost:9200/_cluster/health | jq .

es-indices: ## List ES indices
	@curl -s http://localhost:9200/_cat/indices?v

es-recreate-index: ## Delete and recreate videos_index
	@curl -X DELETE http://localhost:9200/videos_index 2>/dev/null; echo "deleted"
	@curl -X PUT http://localhost:9200/videos_index -H 'Content-Type: application/json' -d '{"settings":{"number_of_shards":3,"number_of_replicas":1,"analysis":{"analyzer":{"ik_max":{"type":"ik_max_word"},"ik_smart":{"type":"ik_smart"}}},"mappings":{"properties":{"video_id":{"type":"long"},"title":{"type":"text","analyzer":"ik_max","search_analyzer":"ik_smart"},"description":{"type":"text","analyzer":"ik_max","search_analyzer":"ik_smart"},"username":{"type":"text","analyzer":"ik_max","search_analyzer":"ik_smart"},"tags":{"type":"keyword"},"popularity":{"type":"long"},"create_time":{"type":"date","format":"strict_date_optional_time||epoch_millis"}}}}}' | jq .

es-refresh: ## Refresh videos_index
	@curl -X POST http://localhost:9200/videos_index/_refresh | jq .

es-stats: ## Get ES nodes stats
	@curl -s http://localhost:9200/_nodes/stats | jq .

.DEFAULT_GOAL := help