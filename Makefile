# Definitions
# support  x86、arm macos or x86 linux
DockerBuild = docker build
DockerRun = docker run
ifeq ($(shell uname -p),arm)
	DockerBuild = docker buildx build --platform=linux/amd64
	DockerRun = docker run --platform=linux/amd64
endif

# Definitions
IMAGE                   := registry.cn-hangzhou.aliyuncs.com/acs/gpushare-scheduler-extender
GIT_VERSION             := $(shell git rev-parse --short=7 HEAD)
COMMIT_ID 				:= $(shell git describe --match=NeVeRmAtCh --abbrev=99 --tags --always --dirty)
GOLANG_DOCKER_IMAGE     := golang:1.19

build-server:
	go build -o bin/gpushare-sche-extender ./cmd/main.go

build-image:
	${DockerBuild} -t ${IMAGE}:${GIT_VERSION} -f scripts/build/Dockerfile .

local-build-image:
	GOOS=linux GOARCH=amd64 go build -o bin/gpushare-sche-extender ./cmd/main.go
	${DockerBuild} -t ${IMAGE}:${GIT_VERSION} -f scripts/build/Dockerfile-local .
