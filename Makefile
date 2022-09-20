# Copyright 2021 The KubeEdge Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

GOPATH?=$(shell go env GOPATH)
IMAGE_REPO ?= kubeedge
ARCH ?= amd64
IMAGE_TAG ?= $(shell git describe --tags)
GO_LDFLAGS='$(shell hack/make-rules/version.sh)'

# make all builds both agent and server binaries

BINARIES=edgemesh-agent \
         edgemesh-gateway

# the env PLATFORMS defines to generate linux images for amd 64-bit, arm 64-bit and armv7 architectures
# the full list of PLATFORMS is linux/amd64,linux/arm64,linux/arm/v7
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7
COMPONENTS=agent \
           gateway

.EXPORT_ALL_VARIABLES:
OUT_DIR ?= _output/local

define ALL_HELP_INFO
# Build code.
#
# Args:
#   WHAT: binary names to build. support: $(BINARIES)
#         the build will produce executable files under $(OUT_DIR)
#         If not specified, "everything" will be built.
#
# Example:
#   make
#   make all
#   make all HELP=y
#   make all WHAT=edgemesh-agent
#   make all WHAT=edgemesh-agent GOLDFLAGS="" GOGCFLAGS="-N -l"
#     Note: Specify GOLDFLAGS as an empty string for building unstripped binaries, specify GOGCFLAGS
#     to "-N -l" to disable optimizations and inlining, this will be helpful when you want to
#     use the debugging tools like delve. When GOLDFLAGS is unspecified, it defaults to "-s -w" which strips
#     debug information, see https://golang.org/cmd/link for other flags.
endef

.PHONY: all
ifeq ($(HELP),y)
all: clean
	@echo "$$ALL_HELP_INFO"
else
all: verify-golang
	EDGEMESH_OUTPUT_SUBPATH=$(OUT_DIR) hack/make-rules/build.sh $(WHAT)
endif

.PHONY: docker-cross-build
ifeq ($(HELP),y)
docker-cross-build:
	@echo "docker cross build for $${COMPONENTS} in platform $${PLATFORMS}"
else
docker-cross-build:
	hack/make-rules/cross-build.sh
endif

define VERIFY_HELP_INFO
# verify golang,vendor
#
# Example:
# make verify
endef
.PHONY: verify
ifeq ($(HELP),y)
verify:
	@echo "$$VERIFY_HELP_INFO"
else
verify:verify-golang verify-vendor verify-vendor-licenses
endif

.PHONY: verify-golang
verify-golang:
	hack/verify-golang.sh
.PHONY: verify-vendor
verify-vendor:
	hack/verify-vendor.sh
.PHONY: verify-vendor-licenses
verify-vendor-licenses:
	hack/verify-vendor-licenses.sh


define LINT_HELP_INFO
# run golang lint check.
#
# Example:
#   make lint
#   make lint HELP=y
endef
.PHONY: lint
ifeq ($(HELP),y)
lint:
	@echo "$$LINT_HELP_INFO"
else
lint:
	hack/make-rules/lint.sh
endif

define E2E_HELP_INFO
# e2e test.
#
# Example:
#   make e2e
#   make e2e HELP=y
#
endef
.PHONY: e2e
ifeq ($(HELP),y)
e2e:
	@echo "$$E2E_HELP_INFO"
else
e2e:
#	This has been commented temporarily since there is an issue of CI using same master for all PRs, which is causing failures when run parallelly
	tests/e2e/scripts/execute.sh
endif


define CLEAN_HELP_INFO
# Clean up the output of make.
#
# Example:
#   make clean
#   make clean HELP=y
#
endef
.PHONY: clean
ifeq ($(HELP),y)
clean:
	@echo "$$CLEAN_HELP_INFO"
else
clean:
	hack/make-rules/clean.sh
endif


define LOADBALANCE_HELP_INFO
# Load balance test.
#
# Example:
#   make lb
#   make lb HELP=y
#
endef
.PHONY: lb
ifeq ($(HELP),y)
lb:
	@echo "$$LOADBALANCE_HELP_INFO"
else
lb:
	tests/loadbalance/execute.sh
endif


.PHONY: images agentimage gatewayimage
images: agentimage gatewayimage
agentimage gatewayimage:
	docker build --build-arg GO_LDFLAGS=${GO_LDFLAGS} -t kubeedge/edgemesh-${@:image=}:${IMAGE_TAG} -f build/${@:image=}/Dockerfile .


.PHONY: push push-all push-multi-platform-images
push-all: push-multi-platform-images

# push target pushes edgemesh-built images
push: images
	for target in $(COMPONENTS); do docker push ${IMAGE_REPO}/edgemesh-$$target:${IMAGE_TAG}; done

# push multi-platform images
push-multi-platform-images:
	bash hack/make-rules/push.sh
