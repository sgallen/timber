GOCACHE := $(CURDIR)/.cache/go-build
GOMODCACHE := $(CURDIR)/.cache/go-mod
GOENV := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE)

.PHONY: fmt
fmt:
	mkdir -p $(GOCACHE) $(GOMODCACHE)
	$(GOENV) gofmt -w cmd/timber/main.go internal/cli/root.go internal/cli/root_test.go internal/timber/paths.go internal/timber/project.go internal/timber/project_test.go

.PHONY: test
test:
	mkdir -p $(GOCACHE) $(GOMODCACHE)
	$(GOENV) go test ./...


.PHONY: test-race
test-race:
	mkdir -p $(GOCACHE) $(GOMODCACHE)
	$(GOENV) go test -race ./...

.PHONY: build
build:
	mkdir -p $(GOCACHE) $(GOMODCACHE)
	mkdir -p bin
	$(GOENV) go build -o ./bin/timber ./cmd/timber

.PHONY: tidy
tidy:
	mkdir -p $(GOCACHE) $(GOMODCACHE)
	$(GOENV) go mod tidy
