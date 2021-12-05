BINARY_NAME=crawler

lint:
	golangci-lint run

test:
	go test ./...

test-v:
	go test ./... -v

test_coverage:
	go test ./... -coverprofile=coverage.out


build:
	go build -o ./bin/${BINARY_NAME} ./cmd/crawler/main.go

run-default:
	CONFIGPATH=config.json ./bin/${BINARY_NAME}