.PHONY: build ui test vet fmt check clean corpus-tidy

BINARY := pathwise

build:
	go build -o $(BINARY) ./cmd/pathwise

# Rebuild the dashboard SPA into internal/server/assets, which `go build` then
# embeds. The built assets are committed, so `make build` needs no Node; run
# `make ui` only after changing anything under frontend/.
ui:
	cd frontend && npm install && npm run build

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# The gate to run before committing.
check: vet test

# The corpus app is its own module and is not built by ./... -- resolve its
# dependencies explicitly when it changes.
corpus-tidy:
	cd testdata/apps/basic-gin && go mod tidy

clean:
	rm -f $(BINARY)
	rm -rf .pathwise
