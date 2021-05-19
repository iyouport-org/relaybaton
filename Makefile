PROJECT="relaybaton"

# Constants
MK_FILE_PATH = $(lastword $(MAKEFILE_LIST))
PRJ_DIR      = $(abspath $(dir $(MK_FILE_PATH)))
GOPATH_ENV  ?= $(shell go env GOPATH)
GOROOT = $(GOPATH_ENV)/src/github.com/cloudflare/go

ANDROID_ARM_CC = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi16-clang
ANDROID_ARM_CXX = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/armv7a-linux-androideabi16-clang++
ANDROID_ARM_STRIP = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/arm-linux-androideabi-strip
ANDROID_ARM64_CC = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android21-clang
ANDROID_ARM64_CXX = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android21-clang++
ANDROID_ARM64_STRIP = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android-strip
ANDROID_X86_CC = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/i686-linux-android16-clang
ANDROID_X86_CXX = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/i686-linux-android16-clang++
ANDROID_X86_STRIP = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/i686-linux-android-strip
ANDROID_X86_64_CC = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/x86_64-linux-android21-clang
ANDROID_X86_64_CXX = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/x86_64-linux-android21-clang++
ANDROID_X86_64_STRIP = /opt/android-ndk/toolchains/llvm/prebuilt/linux-x86_64/bin/x86_64-linux-android-strip

ifeq ($(OS),Windows_NT)
	MAKESCRIPT := make.bat
else
	MAKESCRIPT := make.bash
endif

###############
#
# Build targets
#
##############################

core: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/relaybaton $(PRJ_DIR)/cmd/cli/main.go

desktop: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) $(GOROOT)/bin/go build -buildmode=c-archive -o $(PRJ_DIR)/bin/core.a $(PRJ_DIR)/cmd/desktop/core.go

core_android_arm: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 CC=$(ANDROID_ARM_CC) CXX=$(ANDROID_ARM_CXX) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/librelaybaton.so -trimpath -ldflags="-s -w -buildid=" $(PRJ_DIR)/cmd/cli/main.go
	$(ANDROID_ARM_STRIP) $(PRJ_DIR)/bin/librelaybaton.so

core_android_arm64: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) CGO_ENABLED=1 GOOS=android GOARCH=arm64 CC=$(ANDROID_ARM64_CC) CXX=$(ANDROID_ARM64_CXX) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/librelaybaton.so -trimpath -ldflags="-s -w -buildid=" $(PRJ_DIR)/cmd/cli/main.go
	$(ANDROID_ARM64_STRIP) $(PRJ_DIR)/bin/librelaybaton.so

core_android_x86: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) CGO_ENABLED=1 GOOS=android GOARCH=386 CC=$(ANDROID_X86_CC) CXX=$(ANDROID_X86_CXX) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/librelaybaton.so -trimpath -ldflags="-s -w -buildid=" $(PRJ_DIR)/cmd/cli/main.go
	$(ANDROID_X86_STRIP) $(PRJ_DIR)/bin/librelaybaton.so

core_android_x86_64: $(GOROOT)/pkg $(PRJ_DIR)/vendor
	GOROOT=$(GOROOT) CGO_ENABLED=1 GOOS=android GOARCH=amd64 CC=$(ANDROID_X86_64_CC) CXX=$(ANDROID_X86_64_CXX) $(GOROOT)/bin/go build -o $(PRJ_DIR)/bin/librelaybaton.so -trimpath -ldflags="-s -w -buildid=" $(PRJ_DIR)/cmd/cli/main.go
	$(ANDROID_X86_64_STRIP) $(PRJ_DIR)/bin/librelaybaton.so

###############
#
# Build GOROOT
#
##############################

$(PRJ_DIR)/vendor:
	go mod vendor

$(GOROOT)/pkg: $(GOROOT)
	cd $(GOPATH_ENV)/src/github.com/cloudflare/go/src/ && \
	./$(MAKESCRIPT) && \
	cd $(PRJ_DIR)

$(GOROOT):
	git clone https://github.com/cloudflare/go.git $(GOPATH_ENV)/src/github.com/cloudflare/go
