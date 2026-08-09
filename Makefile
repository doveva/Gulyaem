-include .env

POSTGRES_DB ?= gulyaem
POSTGRES_USER ?= gulyaem
POSTGRES_PASSWORD ?= gulyaem
POSTGRES_PORT ?= 5532
API_PORT ?= 8080
ENVIRONMENT ?= development
LOG_LEVEL ?= info
GEO_DATA_PATH ?= $(CURDIR)/data
GEO_TEST_AREA ?= spb-dense-center
GEO_CITY_CODE ?= spb
GEO_IMPORT_FILE ?=
NORMALIZATION_VERSION ?= stage1-segments-v1
MAX_SEGMENT_LENGTH_M ?= 0
CORS_ALLOWED_ORIGINS ?= http://localhost:5173,http://localhost:3000
VITE_API_URL ?= http://localhost:$(API_PORT)
VITE_MAP_STYLE_URL ?= https://tiles.openfreemap.org/styles/liberty
CGO_ENABLED ?= 0

DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
HTTP_ADDRESS ?= :$(API_PORT)

ifeq ($(strip $(GEO_TEST_AREA)),)
GEO_TEST_AREA := spb-dense-center
endif

ifeq ($(strip $(NORMALIZATION_VERSION)),)
NORMALIZATION_VERSION := stage1-segments-v1
endif

ifeq ($(strip $(MAX_SEGMENT_LENGTH_M)),)
MAX_SEGMENT_LENGTH_M := 0
endif

export DATABASE_URL HTTP_ADDRESS ENVIRONMENT LOG_LEVEL GEO_DATA_PATH GEO_TEST_AREA GEO_CITY_CODE
export NORMALIZATION_VERSION MAX_SEGMENT_LENGTH_M CORS_ALLOWED_ORIGINS VITE_API_URL VITE_MAP_STYLE_URL CGO_ENABLED

.PHONY: bootstrap db-up migrate geo-import api frontend up down logs check docs-check

bootstrap:
	cd backend && go mod download
	cd frontend && npm ci

db-up:
	docker compose up -d db

migrate:
	docker compose run --rm migrate

geo-import:
	@if [ -n "$(GEO_IMPORT_FILE)" ]; then \
		cd backend && go run ./cmd/geo-import --file "$(GEO_IMPORT_FILE)" --city-code "$(GEO_CITY_CODE)" --normalization-version "$(NORMALIZATION_VERSION)" --max-segment-length-m "$(MAX_SEGMENT_LENGTH_M)"; \
	else \
		cd backend && go run ./cmd/geo-import --fixture "$(GEO_TEST_AREA)" --normalization-version "$(NORMALIZATION_VERSION)" --max-segment-length-m "$(MAX_SEGMENT_LENGTH_M)"; \
	fi

api:
	cd backend && go run ./cmd/api

frontend:
	cd frontend && npm run dev

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api frontend

check:
	cd backend && GOCACHE=$(CURDIR)/.gocache go test ./... && GOCACHE=$(CURDIR)/.gocache go vet ./...
	cd frontend && npm run lint && npm test && npm run build
	python3 scripts/docs/docs.py index --check
	python3 scripts/docs/docs.py check

docs-check:
	python3 scripts/docs/docs.py index --check
	python3 scripts/docs/docs.py check
