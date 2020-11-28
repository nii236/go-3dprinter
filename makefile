PACKAGE=goprint
DOCKER_CONTAINER=$(PACKAGE)-db
MIGRATE_VERSION=v4.12.2

BIN = $(CURDIR)/bin
SERVER = $(CURDIR)/server
WEB = $(CURDIR)/web

LOCAL_DEV_DB_USER=$(PACKAGE)
LOCAL_DEV_DB_PASS=dev
LOCAL_DEV_DB_HOST=localhost
LOCAL_DEV_DB_PORT=5432
LOCAL_DEV_DB_DATABASE=$(PACKAGE)
DB_CONNECTION_STRING="postgres://$(LOCAL_DEV_DB_USER):$(LOCAL_DEV_DB_PASS)@$(LOCAL_DEV_DB_HOST):$(LOCAL_DEV_DB_PORT)/$(LOCAL_DEV_DB_DATABASE)?sslmode=disable"


.PHONY: tools
tools:
	go generate -tags tools ./tools/...

.PHONY: serve
serve:
	../bin/air

.PHONY: db-drop
db-drop:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path ./migrations drop -f

.PHONY: db-migrate
db-migrate:
	$(BIN)/migrate -database $(DB_CONNECTION_STRING) -path ./migrations up

.PHONY: db-seed
db-seed:
	go run cmd/platform/main.go db --seed

.PHONY: db-setup
db-setup: 
	go run cmd/platform/main.go db --drop --migrate --seed

.PHONY: db-prepare
db-prepare: 
	docker exec -it goprint-db psql -U goprint


.PHONY: docker-start
docker-start:
	docker start $(DOCKER_CONTAINER) || docker run -d -p $(LOCAL_DEV_DB_PORT):5432 --name $(DOCKER_CONTAINER) -e POSTGRES_USER=$(PACKAGE) -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=$(PACKAGE) postgres:11-alpine

.PHONY: docker-stop
docker-stop:
	docker stop $(DOCKER_CONTAINER)

.PHONY: docker-remove
docker-remove:
	docker rm $(DOCKER_CONTAINER)

.PHONY: docker-setup
docker-setup:
	docker exec -it $(DOCKER_CONTAINER) psql -U $(PACKAGE) -c 'CREATE EXTENSION IF NOT EXISTS pg_trgm; CREATE EXTENSION IF NOT EXISTS pgcrypto; CREATE EXTENSION IF NOT EXISTS "uuid-ossp";'

.PHONY: sql
sql:
	$(BIN)/sqlboiler $(BIN)/sqlboiler-psql --wipe --tag db --config ./sqlboiler.toml --output db

.PHONY: gql
gql:
	cd $(SERVER) && go mod tidy
	cd $(SERVER)/graphql && go run github.com/99designs/gqlgen

.PHONY: bindata
bindata:
	cd $(SERVER) && go generate

.PHONY: generate
generate: bindata sql 

.PHONY: web-install
web-install:
	cd $(WEB) && npm install

.PHONY: go-mod-download
go-mod-download:
	cd $(SERVER) && go mod download

.PHONY: deps
deps: web-install go-mod-download

.PHONY: web-watch
web-watch:
	cd $(WEB) && npm start

.PHONY: destroy
destroy: docker-stop docker-remove

.PHONY: wait
wait: 
	sleep 5

.PHONY: init
init: docker-start wait docker-setup deps tools db-migrate generate db-seed
