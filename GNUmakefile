default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run
	terraform fmt -check -recursive examples/

generate:
	go tool tfplugindocs generate -provider-name zabbix

fmt:
	gofmt -s -w -e .
	terraform fmt -recursive examples/

unit-tests:
	go test -v -coverprofile=coverage.out -covermode=atomic -timeout=120s -parallel=10 ./...

testacc-up:
	docker compose up -d
	scripts/testacc-bootstrap.sh > .testacc.env

testacc:
	@set -a && [ -f .testacc.env ] && . ./.testacc.env; set +a && \
		TF_ACC=1 go test -v -coverprofile=coverage.out -covermode=atomic -timeout 120m ./...

testacc-down:
	docker compose down -v
	rm -f .testacc.env

acc-tests:
	$(MAKE) testacc-up && $(MAKE) testacc; STATUS=$$?; $(MAKE) testacc-down; exit $$STATUS

.PHONY: fmt lint unit-tests testacc testacc-up testacc-down acc-tests build install generate
