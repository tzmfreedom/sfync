.PHONY: run
run: fmt
	go run main.go

.PHONY: fmt
fmt:
	gofmt -w .
