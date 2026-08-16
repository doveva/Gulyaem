-include .env

POSTGRES_DB ?= gulyaem
POSTGRES_USER ?= gulyaem
POSTGRES_PASSWORD ?= gulyaem
POSTGRES_PORT ?= 5532
API_PORT ?= 8080
ENVIRONMENT ?= development
LOG_LEVEL ?= info
GEO_DATA_PATH ?= $(CURDIR)/data
GEO_TEST_AREA ?= spb-stage1-validation
GEO_CITY_CODE ?= spb
GEO_IMPORT_FILE ?=
NORMALIZATION_VERSION ?= stage1-segments-v1
MAX_SEGMENT_LENGTH_M ?= 0
DISTRICT_TEST_AREA ?= spb-administrative-districts
DISTRICT_IMPORT_FILE ?=
DISTRICT_NORMALIZATION_VERSION ?= stage1-districts-v1
CORS_ALLOWED_ORIGINS ?= http://localhost:5173,http://localhost:3000
VITE_API_URL ?= http://localhost:$(API_PORT)
VITE_MAP_STYLE_URL ?= https://tiles.openfreemap.org/styles/liberty
CGO_ENABLED ?= 0
VALHALLA_PORT ?= 8002
GRAPHHOPPER_PORT ?= 8989
OSRM_PORT ?= 5001
VALHALLA_URL ?= http://localhost:$(VALHALLA_PORT)
ROUTING_ENGINE_VERSION ?= 3.7.0
ROUTING_DATASET_METADATA_PATH ?= $(CURDIR)/.routing/valhalla/routing-dataset.json
DEVELOPMENT_ACTOR_ID ?= 01900000-0000-7000-8000-000000000003
CITY_ID ?= 01900000-0000-7000-8000-000000000001
GRAPHHOPPER_URL ?= http://localhost:$(GRAPHHOPPER_PORT)
OSRM_URL ?= http://localhost:$(OSRM_PORT)

DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
HTTP_ADDRESS ?= :$(API_PORT)

ifeq ($(strip $(GEO_TEST_AREA)),)
GEO_TEST_AREA := spb-stage1-validation
endif

ifeq ($(strip $(NORMALIZATION_VERSION)),)
NORMALIZATION_VERSION := stage1-segments-v1
endif

ifeq ($(strip $(MAX_SEGMENT_LENGTH_M)),)
MAX_SEGMENT_LENGTH_M := 0
endif

export DATABASE_URL HTTP_ADDRESS ENVIRONMENT LOG_LEVEL GEO_DATA_PATH GEO_TEST_AREA GEO_CITY_CODE DISTRICT_TEST_AREA
export NORMALIZATION_VERSION MAX_SEGMENT_LENGTH_M CORS_ALLOWED_ORIGINS VITE_API_URL VITE_MAP_STYLE_URL CGO_ENABLED
export DISTRICT_NORMALIZATION_VERSION
export VALHALLA_PORT GRAPHHOPPER_PORT OSRM_PORT VALHALLA_URL GRAPHHOPPER_URL OSRM_URL
export ROUTING_DATASET_METADATA_PATH ROUTING_ENGINE_VERSION
export DEVELOPMENT_ACTOR_ID

.PHONY: bootstrap db-up migrate geo-import district-import exploration-rebuild api frontend up down logs check docs-check \
	routing-prepare routing-images routing-up routing-benchmark routing-down routing-reset routing-spike \
	stage1-e2e stage1-validate stage1-freeze-check stage3-coverage-validate

bootstrap:
	cd backend && go mod download
	cd frontend && npm ci
	cd frontend && npx playwright install chromium

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

district-import:
	@if [ -n "$(DISTRICT_IMPORT_FILE)" ]; then \
		cd backend && go run ./cmd/district-import --file "$(DISTRICT_IMPORT_FILE)" --city-code "$(GEO_CITY_CODE)" --normalization-version "$(DISTRICT_NORMALIZATION_VERSION)"; \
	else \
		cd backend && go run ./cmd/district-import --fixture "$(DISTRICT_TEST_AREA)" --normalization-version "$(DISTRICT_NORMALIZATION_VERSION)"; \
	fi

exploration-rebuild:
	cd backend && go run ./cmd/exploration-rebuild --actor "$(DEVELOPMENT_ACTOR_ID)" --city "$(CITY_ID)"

api:
	cd backend && go run ./cmd/api

frontend:
	cd frontend && npm run dev

up:
	docker compose up -d --force-recreate routing-prepare valhalla routing-metadata
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f api frontend

check:
	cd backend && GOCACHE=$(CURDIR)/.gocache go test ./... && GOCACHE=$(CURDIR)/.gocache go vet ./...
	cd frontend && npm run lint && npm test && npm run build
	./scripts/routing/prepare_test.sh
	python3 scripts/docs/docs.py index --check
	python3 scripts/docs/docs.py check

docs-check:
	python3 scripts/docs/docs.py index --check
	python3 scripts/docs/docs.py check

stage1-e2e:
	cd frontend && npm run test:e2e

stage1-validate:
	python3 scripts/validation/stage1.py \
		--root "$(CURDIR)" \
		--api-url "$(VITE_API_URL)" \
		--output "data/validation/spb-stage1/report.json"

stage1-freeze-check: check stage1-e2e stage1-validate

stage3-coverage-validate:
	python3 scripts/validation/coverage_v2.py \
		--root "$(CURDIR)" \
		--api-url "$(VITE_API_URL)" \
		--output "data/validation/spb-stage3-coverage-v2/report.json"

routing-prepare:
	./scripts/routing/prepare.sh

routing-images:
	docker compose --profile routing-spike pull valhalla osrm osrm-prepare
	docker compose --profile routing-spike build graphhopper

routing-up: routing-images routing-prepare
	./scripts/routing/up.sh

routing-benchmark:
	cd backend && GOCACHE=$(CURDIR)/.gocache go run ./cmd/routing-spike \
		--root "$(CURDIR)" \
		--setup-metrics "$(CURDIR)/.routing/setup-metrics.json" \
		--output "$(CURDIR)/frontend/public/routing-spike/comparison.json"

routing-down:
	docker compose --profile routing-spike stop valhalla graphhopper osrm

routing-reset:
	./scripts/routing/reset.sh

routing-spike: routing-up routing-benchmark
