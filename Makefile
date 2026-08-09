.PHONY: build test fmt vet lint install-crd install-policies install-rbac run docker-build

MODULE := github.com/YOUR_USERNAME/homeostat
IMG    ?= homeostat:dev

build:
	go build -o bin/controller ./cmd/controller

test:
	go test ./... -race -coverprofile=coverage.out

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

install-crd:
	kubectl apply -f config/crd/bases/

install-policies:
	kubectl apply -f policies/

install-rbac:
	kubectl apply -f config/rbac/

run: build
	./bin/controller

docker-build:
	docker build -t $(IMG) .
