# ==============================
# FAMS Backend Makefile
# ==============================

.PHONY: infra-up infra-down infra-reset healthcheck \
        test-unit test-integration test-e2e test-all \
        lint clean

# ---- Infrastructure ----
infra-up:
	./scripts/infra-up.sh

infra-down:
	./scripts/infra-down.sh

infra-reset:
	./scripts/infra-reset.sh

healthcheck:
	./scripts/healthcheck.sh

# ---- Testing ----
test-unit:
	go test ./pkg/... ./service/... -short -v -cover

test-integration:
	go test ./tests/integration/... -tags=integration -v

test-e2e:
	go test ./tests/e2e/... -tags=e2e -v

test-all: test-unit test-integration test-e2e

# ---- Code Quality ----
lint:
	golangci-lint run ./...

# ---- Clean ----
clean:
	docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/docker-compose-env.yml down -v
	rm -rf data/
