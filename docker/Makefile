build:
	@[ -d .build ] || mkdir .build
	CGO_ENABLED=0 go build -ldflags="-s -w" -o .build/ts-supervisor ../cmd/supervisor/main.go
	file  .build/ts-supervisor
	du -h .build/ts-supervisor
image: build
	docker build -t trafficstars-supervisor .

.PHONY: build