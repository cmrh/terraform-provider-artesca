HOSTNAME=registry.opentofu.org
NAMESPACE=scality
NAME=artesca
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

default: build

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

test:
	go test ./... -v

testacc:
	TF_ACC=1 TF_ACC_TERRAFORM_PATH=$$(which tofu) TF_ACC_PROVIDER_NAMESPACE=${NAMESPACE} go test ./internal/provider/ -v -timeout 60m

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

.PHONY: default build install test testacc lint fmt
