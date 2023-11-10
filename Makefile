all::

include Makefile.common

PROMTOOL_VERSION ?= 2.30.0
PROMTOOL_URL     ?= https://github.com/prometheus/prometheus/releases/download/v$(PROMTOOL_VERSION)/prometheus-$(PROMTOOL_VERSION).$(GO_BUILD_PLATFORM).tar.gz
PROMTOOL         ?= $(FIRST_GOPATH)/bin/promtool

DOCKER_IMAGE_NAME       ?= openmeteo-exporter
MACH                    ?= $(shell uname -m)

STATICCHECK_IGNORE =

PROMU_CONF := .promu.yml
PROMU := $(FIRST_GOPATH)/bin/promu --config $(PROMU_CONF)

.PHONY: build
build: promu openmeteo_exporter
openmeteo_exporter: *.go
	$(PROMU) build -v

fmt:
	@echo ">> Running fmt"
	gofmt -l -w -s .

crossbuild: promu
	@echo ">> Running crossbuild"
	GOARCH=amd64 $(PROMU) build --prefix=output/amd64
	GOARCH=arm64 $(PROMU) build --prefix=output/arm64
	ifeq ($(GOOS),windows)
	GOARCH=arm   $(PROMU) build --prefix=output/386
	else
	GOARCH=arm   $(PROMU) build --prefix=output/arm
	endif

clean:
	@echo ">> Running clean"
	rm -rf openmeteo_exporter output
