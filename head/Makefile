# The head is a separate cgo module (Vulkan + GLFW + vendored Dear ImGui). It needs a
# C/C++ toolchain, the Vulkan loader/headers, GLFW, and the Go toolchain named by the
# workspace go.work (currently go 1.26.4). GOTOOLCHAIN=local pins the build to the
# installed toolchain instead of re-resolving/downloading one over the network: the dev
# machine's Go already satisfies the go.work floor, and a hard version pin (e.g. go1.24.0)
# is both stale against go.work and would trigger a network fetch when the installed build
# is a custom variant (e.g. go1.26.4-X).
export CGO_ENABLED := 1
export GOTOOLCHAIN := local

HOST_NATIVE_LIB_DIR := /run/host/usr/lib/x86_64-linux-gnu
HOST_PKG_CONFIG_DIR := $(HOST_NATIVE_LIB_DIR)/pkgconfig
LOCAL_NATIVE_LIB_DIR := $(abspath $(CURDIR)/../.local-native-lib)

ifneq ($(wildcard $(HOST_PKG_CONFIG_DIR)/glfw3.pc),)
export PKG_CONFIG_PATH := $(HOST_PKG_CONFIG_DIR)$(if $(PKG_CONFIG_PATH),:$(PKG_CONFIG_PATH))
export PKG_CONFIG_SYSROOT_DIR := /run/host
export LD_LIBRARY_PATH := $(LOCAL_NATIVE_LIB_DIR)$(if $(LD_LIBRARY_PATH),:$(LD_LIBRARY_PATH))
endif

.PHONY: build smoke run run-watch lint test shaders native-libs clean

# native-libs exposes only the host GLFW/Vulkan runtime libs to Flatpak builds.
# Do not put the whole host libdir on LD_LIBRARY_PATH; it can load host libc into
# the SDK process and break cgo.
native-libs:
	@if [ -d "$(HOST_NATIVE_LIB_DIR)" ]; then \
		mkdir -p "$(LOCAL_NATIVE_LIB_DIR)"; \
		ln -sfn "$(HOST_NATIVE_LIB_DIR)/libglfw.so" "$(LOCAL_NATIVE_LIB_DIR)/libglfw.so"; \
		ln -sfn "$(HOST_NATIVE_LIB_DIR)/libglfw.so.3" "$(LOCAL_NATIVE_LIB_DIR)/libglfw.so.3"; \
		ln -sfn "$(HOST_NATIVE_LIB_DIR)/libvulkan.so" "$(LOCAL_NATIVE_LIB_DIR)/libvulkan.so"; \
		ln -sfn "$(HOST_NATIVE_LIB_DIR)/libvulkan.so.1" "$(LOCAL_NATIVE_LIB_DIR)/libvulkan.so.1"; \
	fi

# shaders recompiles the viewport GLSL to the embedded SPIR-V (needs glslangValidator
# from glslang-tools). Run after editing internal/native/shaders/*.vert|frag.
shaders:
	cd internal/native/shaders && \
		glslangValidator -V mesh.vert -o mesh.vert.spv && \
		glslangValidator -V mesh.frag -o mesh.frag.spv && \
		glslangValidator -V point.vert -o point.vert.spv && \
		glslangValidator -V point.frag -o point.frag.spv && \
		glslangValidator -V wideline.vert -o wideline.vert.spv && \
		glslangValidator -V wideline.frag -o wideline.frag.spv && \
		glslangValidator -V skybox.vert -o skybox.vert.spv && \
		glslangValidator -V skybox.frag -o skybox.frag.spv && \
		glslangValidator --target-env vulkan1.3 -V raytrace.comp -o raytrace.comp.spv && \
		glslangValidator --target-env vulkan1.3 -V swtrace.comp -o swtrace.comp.spv

# build compiles every head command (proves the cgo/Vulkan stack links).
build: native-libs
	go build ./...

lint: native-libs
	golangci-lint run ./...

test: native-libs
	go test ./...

# smoke opens a real GLFW+Vulkan window, renders a few Dear ImGui frames, and exits 0
# if the whole native stack initialized. Requires a display (X11/Wayland) + a Vulkan
# driver (hardware or lavapipe).
smoke: native-libs
	go run ./cmd/headsmoke -frames 5

# run launches the application. It points the add-in loader at ./addins (where
# `make install` in add-in/oblikovati-mcp-bridge drops the built library), so any
# installed add-in (e.g. the MCP bridge) loads at startup. Override with
# OBK_ADDINS_DIR, and OBK_MCP_ADDR to move the MCP endpoint off 127.0.0.1:7800.
run: native-libs
	OBK_ADDINS_DIR=$(CURDIR)/addins go run ./cmd/oblikovati-head

# run-watch launches the app under a relaunch loop with OBK_ADDIN_AUTORESTART=1, so a
# rebuilt add-in (`make install` in the bridge) makes the app exit cleanly and come
# straight back with the new library — the safe stand-in for in-process hot-reload (a
# Go c-shared add-in cannot be swapped in a live process). The head binary is built
# ONCE and the prebuilt binary is relaunched, so a .so swap restarts in ~1s (no cgo
# recompile). Re-run this target after changing head source. Ctrl-C twice to stop.
run-watch: native-libs
	go build -o oblikovati-head ./cmd/oblikovati-head
	@while true; do \
		OBK_ADDINS_DIR=$(CURDIR)/addins OBK_ADDIN_AUTORESTART=1 ./oblikovati-head || true; \
		echo "oblikovati-head exited; relaunching (Ctrl-C to stop)"; \
		sleep 1; \
	done

clean:
	go clean ./...
