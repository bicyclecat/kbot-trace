APP:=kbot-trace
REGISTRY=docker.io
REPOSITORY=bicyclecat
GIT_REPOSITORY=bicyclecat
VERSION=$(shell git describe --tags --abbrev=0)-$(shell git rev-parse --short HEAD)

TARGETOS=linux
TARGETARCH=amd64

format:
	gofmt -s -w ./

lint:
	golint

test:
	go test -v

get-dependencies:
	go get

build: format get-dependencies
	CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o ${APP} -ldflags "-X="github.com/${GIT_REPOSITORY}/${APP}/cmd.appVersion=${VERSION}

image:
	docker build . -t ${REGISTRY}/${REPOSITORY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH} --build-arg TARGETOS=${TARGETOS} --build-arg TARGETARCH=${TARGETARCH}

push:
	docker push ${REGISTRY}/${REPOSITORY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}

clean:
	rm -rf kbot
	docker rmi ${REGISTRY}/${REPOSITORY}/${APP}:${VERSION}-${TARGETOS}-${TARGETARCH}