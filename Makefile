.PHONY: run test test-race vet check build smoke clean

run:
	go run ./cmd/server

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check: test vet

smoke:
	./scripts/smoke.sh http://localhost:8080

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/booking-os ./cmd/server

clean:
	rm -rf bin data/booking-os.json
