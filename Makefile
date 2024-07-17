.SILENT: main
.PHONY: main lint import server test
.DEFAULT_GOAL := main
main: # don't change this line; first line is the default target in make <= 3.79 despite .DEFAULT_GOAL
	echo "commands:"
	echo "lint, migrate, import, server, test"
lint:
	./scripts/ci-lint.sh
migrate:
	tern migrate -m ./migrations
import:
	go run github.com/henvic/vio/cmd/import
server:
	go run github.com/henvic/vio/cmd/server
test:
	go test -race -v ./...