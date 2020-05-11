################################################################################
# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2019 Intel Corporation
################################################################################


export GO111MODULE = on

.PHONY: build af nef lint test-unit-af test-unit-nef
TMP_DIR:=$(shell mktemp -d)
BUILD_DIR ?=dist
CERTS_DIR ?=/etc/certs

VER:=1.0

build: af nef

af:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/af" ./cmd/af
	cp ${BUILD_DIR}/af .

nef:
	mkdir -p "${BUILD_DIR}"
	GOOS=linux go build -o "${BUILD_DIR}/nef" ./cmd/nef
	cp ${BUILD_DIR}/nef .

clean:
	rm -rf "${BUILD_DIR}"
	rm -rf "${CERTS_DIR}"
	rm -rf extfile.cnf

help:
	@echo "Please use \`make <target>\` where <target> is one of"
	@echo "Building Packages:"
	@echo "  build                  to build all"
	@echo "  af                     to build the AF"
	@echo "  nef                    to build the NEF"
