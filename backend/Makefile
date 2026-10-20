.PHONY: run test build fmt vet migrate-up migrate-down
run:
	go run ./cmd/api
test:
	go test ./...
build:
	go build ./cmd/api
fmt:
	gofmt -w $$(find . -name '*.go' -type f)
vet:
	go vet ./...
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up
migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down
