M 		= $(shell printf "\033[34;1m>>\033[0m")
GO 		= go
PATH 	:= $(GOBIN):$(PATH)
GOBIN 	?= $(PWD)/bin


.PHONY: build-proto
build-proto:
	buf generate --path internal/handlers/grpc/proto/user-manager/v1/


.PHONY: install-linter
install-linter: $(GOBIN)
	@GOBIN=$(GOBIN) $(GO) install \
		github.com/golangci/golangci-lint/cmd/golangci-lint

lint: install-linter ; $(info $(M) running linters...)
	@$(GOBIN)/golangci-lint run --timeout 5m0s ./...
