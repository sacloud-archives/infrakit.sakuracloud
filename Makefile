TEST?=$$(go list ./... | grep -v vendor)
VETARGS?=-all
GOFMT_FILES?=$$(find . -name '*.go' | grep -v vendor)
GOGEN_FILES?=$$(go list ./... | grep -v vendor)
BIN_NAME?=usacloud
CURRENT_VERSION = $(shell git log --merges --oneline | perl -ne 'if(m/^.+Merge pull request \#[0-9]+ from .+\/bump-version-([0-9\.]+)/){print $$1;exit}')

BUILD_LDFLAGS = "-s -w \
	  -X github.com/sacloud/infrakit.sakuracloud/version.Revision=`git rev-parse --short HEAD` \
	  -X github.com/sacloud/infrakit.sakuracloud/version.Version=$(CURRENT_VERSION)"

.PHONY: default
default: test vet

.PHONY: run
run:
	go run $(CURDIR)/plugin/instance/cmd/main.go $(ARGS)

.PHONY: clean
clean:
	rm -Rf build/*

.PHONY: deps
deps:
	go get -u github.com/kardianos/govendor; \
	go get -u github.com/golang/lint/golint

.PHONY: build build-x build-darwin build-windows build-linux
build: clean vet
	OS="`go env GOOS`" ARCH="`go env GOARCH`" ARCHIVE= BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

build-x: build-darwin build-windows build-linux

build-darwin: build/infrakit-instance-sakuracloud_darwin-amd64.zip

build-windows: build/infrakit-instance-sakuracloud_windows-386.zip build/infrakit-instance-sakuracloud_windows-amd64.zip

build-linux: build/infrakit-instance-sakuracloud_linux-386.zip build/infrakit-instance-sakuracloud_linux-amd64.zip

build/infrakit-instance-sakuracloud_darwin-amd64.zip:
	OS="darwin"  ARCH="amd64"     ARCHIVE=1 BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

build/infrakit-instance-sakuracloud_windows-386.zip:
	OS="windows" ARCH="386"     ARCHIVE=1 BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

build/infrakit-instance-sakuracloud_windows-amd64.zip:
	OS="windows" ARCH="amd64"     ARCHIVE=1 BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

build/infrakit-instance-sakuracloud_linux-386.zip:
	OS="linux"   ARCH="386" ARCHIVE=1 BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

build/infrakit-instance-sakuracloud_linux-amd64.zip:
	OS="linux"   ARCH="amd64" ARCHIVE=1 BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"

.PHONY: test
test: vet
	go test $(TEST) $(TESTARGS) -v -timeout=30m -parallel=4 ;

.PHONY: vet
vet: golint
	@echo "go tool vet $(VETARGS) ."
	@go tool vet $(VETARGS) $$(ls -d */ | grep -v vendor) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: golint
golint: fmt
	for pkg in $$(go list ./... | grep -v /vendor/ ) ; do \
        test -z "$$(golint $$pkg | grep -v 'should have comment' | tee /dev/stderr)" || RES=1; \
    done ;exit $$RES

.PHONY: fmt
fmt:
	gofmt -s -l -w $(GOFMT_FILES)

.PHONY: build-docs serve-docs
build-docs:
	sh -c "'$(CURDIR)/scripts/build_docs.sh'"

serve-docs:
	sh -c "'$(CURDIR)/scripts/serve_docs.sh'"

.PHONY: docker-run docker-test docker-build docker-rpm
docker-run:
	sh -c "$(CURDIR)/scripts/build_docker_image.sh" ; \
	sh -c "$(CURDIR)/scripts/run_on_docker.sh"

docker-test:
	sh -c "'$(CURDIR)/scripts/build_on_docker.sh' 'test'"

docker-build: clean
	sh -c "'$(CURDIR)/scripts/build_on_docker.sh' 'build-x'"

docker-image: clean
	OS="linux"   ARCH="amd64" ARCHIVE= BUILD_LDFLAGS=$(BUILD_LDFLAGS) sh -c "'$(CURDIR)/scripts/build.sh'"
	sh -c "$(CURDIR)/scripts/build_docker_bundle.sh"

integration-tests: docker-image
	@echo "+ $@"
	@cd ${CURDIR}/integration; ./instance.sh