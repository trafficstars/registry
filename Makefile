vendor: deps
	-rm -fR vendor
	govendor init
	govendor add +external

deps:
	go get -u github.com/kardianos/govendor

.PHONY: build