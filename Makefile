vendor: deps
	-rm -fR vendor
	govendor init
	govendor add +external

deps:
	go get -u github.com/kardianos/govendor

.PHONY: fmt
fmt: ## Run formatting code
	@echo "Fix formatting"
	@gofmt -w ${GO_FMT_FLAGS} $$(go list -f "{{ .Dir }}" ./...); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi

.PHONY: build