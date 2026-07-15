.PHONY: test check web-build build verify

test:
	go test ./...
	cd web && npm test -- --run

check:
	go vet ./...
	cd web && npm run check

web-build:
	cd web && npm run build

build: web-build
	mkdir -p internal/httpapi/dist
	cp -R web/build/. internal/httpapi/dist/
	go build -o chronograph ./cmd/chronograph

verify: test check build
	go test -race ./...

