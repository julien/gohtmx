.PHONY: test coverage coverage-html

test:
	@go test ./... -count 1 -cover -covermode atomic -coverprofile coverage -race -v

coverage: test
	@go tool cover -func coverage

coverage-html: test
	@go tool cover -html coverage
