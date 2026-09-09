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

.PHONY: build smoke run run-watch lint test deps-check display-check shaders native-libs clean

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

# HEAD_SESSION_DISPLAY is the display the in-window GLFW/Vulkan tests need. With none,
# the tests run under a PRIVATE Xvfb server — the recipe CI already proves works
# (.github/workflows/ci.yml, step "Test head (xvfb + lavapipe)"). With neither a session
# display nor xvfb-run, deps-check refuses: ~40 in-window tests call t.Skipf("no
# window/GPU available") and the packages still print `ok`, which is the exact
# "green means less than you think" failure #3526 exists to remove.
HEAD_SESSION_DISPLAY := $(or $(DISPLAY),$(WAYLAND_DISPLAY))
XVFB_RUN := $(if $(HEAD_SESSION_DISPLAY),,$(shell command -v xvfb-run 2>/dev/null))
TEST_UNDER_DISPLAY := $(if $(XVFB_RUN),$(XVFB_RUN) -a)
HEAD_RUN_DISPLAY := $(if $(HEAD_SESSION_DISPLAY),session display $(HEAD_SESSION_DISPLAY),a private Xvfb display)

# VULKAN_ICDS counts the driver manifests the Vulkan loader can see. `pkg-config --exists
# vulkan` proves the LOADER is installed and can never prove a DRIVER exists; with zero
# ICDs every in-window test skips while its package still reports `ok`. Recursively
# expanded (`=`, not `:=`) so it is read where it is printed, after native-libs has run.
VULKAN_ICDS = $(if $(VK_ICD_FILENAMES),1+,$(words $(wildcard /usr/share/vulkan/icd.d/*.json \
  /usr/local/share/vulkan/icd.d/*.json /etc/vulkan/icd.d/*.json)))

# NATIVE_PKGS are the pkg-config names the cgo build links against (see
# internal/native: `#cgo pkg-config: glfw3 vulkan`). deps-check exists so the repo-root
# `make gate` can say WHICH native dep is missing and exit non-zero, instead of letting
# the head module drop out of the gate unnoticed (#3526): "nothing touched head" has to
# be a measurement, not an assertion.
NATIVE_PKGS := glfw3 vulkan

deps-check: native-libs
	@missing=""; \
	for tool in cc pkg-config; do \
		command -v $$tool >/dev/null 2>&1 || missing="$$missing $$tool"; \
	done; \
	if command -v pkg-config >/dev/null 2>&1; then \
		for pkg in $(NATIVE_PKGS); do \
			pkg-config --exists $$pkg || missing="$$missing $$pkg(pkg-config)"; \
		done; \
	else \
		missing="$$missing $(NATIVE_PKGS)(unprobed: no pkg-config)"; \
	fi; \
	if [ -n "$$missing" ]; then \
		echo "head: MISSING native deps:$$missing"; \
		echo "head: Debian/Ubuntu: sudo apt-get install build-essential pkg-config libglfw3-dev libvulkan-dev mesa-vulkan-drivers"; \
		echo "head: macOS: brew install glfw vulkan-loader vulkan-headers molten-vk pkg-config"; \
		echo "head: the cgo Vulkan module cannot be built or tested here, so the gate is NOT satisfied"; \
		exit 1; \
	fi; \
	echo "head: native deps OK (glfw3 $$(pkg-config --modversion glfw3), vulkan $$(pkg-config --modversion vulkan))"; \
	echo "head: what this proves: the C toolchain, GLFW, and the Vulkan LOADER are installed."; \
	echo "head: what it cannot prove: that a Vulkan DRIVER initialises. Only 'make smoke' does."; \
	if [ "$(VULKAN_ICDS)" = "0" ]; then \
		echo "head: WARNING zero Vulkan ICD manifests - every in-window test will skip and its"; \
		echo "head: package will still print ok. Install mesa-vulkan-drivers for the lavapipe"; \
		echo "head: software ICD, which is what CI uses."; \
	fi

# display-check is SEPARATE from deps-check because the two failures are different: no
# native deps means the module cannot be built at all, while no display means only the
# in-window tests cannot run. `make gate` demands both (it passes HEAD_ALLOW_NO_DISPLAY=
# on the command line, which beats any environment value). A developer on a headless box
# can still run the ~15 non-window packages with HEAD_ALLOW_NO_DISPLAY=1, and the run
# says out loud which coverage that costs.
display-check:
	@if [ -n "$$DISPLAY" ] || [ -n "$$WAYLAND_DISPLAY" ] || command -v xvfb-run >/dev/null 2>&1; then \
		echo "head: display $(if $(HEAD_SESSION_DISPLAY),$(HEAD_SESSION_DISPLAY),a private Xvfb via $(XVFB_RUN)), Vulkan ICD manifests $(VULKAN_ICDS)"; \
	elif [ -n "$(HEAD_ALLOW_NO_DISPLAY)" ]; then \
		echo "head: HEAD_ALLOW_NO_DISPLAY=$(HEAD_ALLOW_NO_DISPLAY): running WITHOUT a display."; \
		echo "head: every in-window GLFW/Vulkan test SKIPS and its package still prints ok, so"; \
		echo "head: this run does NOT cover the window path. It does not satisfy the gate."; \
	else \
		echo "head: MISSING a display: no DISPLAY, no WAYLAND_DISPLAY and no xvfb-run"; \
		echo "head: Debian/Ubuntu: sudo apt-get install xvfb   (CI runs: xvfb-run -a go test ./...)"; \
		echo "head: or run the gate from a desktop session"; \
		echo "head: without one the in-window GLFW/Vulkan tests SKIP and their packages still print ok"; \
		echo "head: to run only the non-window packages anyway: make test HEAD_ALLOW_NO_DISPLAY=1"; \
		exit 1; \
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
		glslangValidator --target-env vulkan1.3 -V swtrace.comp -o swtrace.comp.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace.rgen -o pathtrace.rgen.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace.rmiss -o pathtrace.rmiss.spv && \
		glslangValidator --target-env vulkan1.3 -V shadow.rmiss -o shadow.rmiss.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace.rchit -o pathtrace.rchit.spv && \
		glslangValidator --target-env vulkan1.3 -V swpathtrace.comp -o swpathtrace.comp.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace_realistic.rgen -o pathtrace_realistic.rgen.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace_realistic.rmiss -o pathtrace_realistic.rmiss.spv && \
		glslangValidator --target-env vulkan1.3 -V pathtrace_realistic.rchit -o pathtrace_realistic.rchit.spv && \
		glslangValidator --target-env vulkan1.3 -V swpathtrace_realistic.comp -o swpathtrace_realistic.comp.spv

# build compiles every head command (proves the cgo/Vulkan stack links).
build: native-libs
	go build ./...

lint: native-libs
	golangci-lint run ./...

# -p 1 runs one PACKAGE at a time. Several packages here open a real GLFW window and a
# real Vulkan device, and under a live compositor they contend for the window server:
# a parallel `go test ./...` failed TestWaitEventsBlocksWhenIdle (a timing assertion)
# and TestInWindowToolFaceHighlightGeneralizes in 2 of 3 runs, while 3 serial runs and
# every isolated run were green. It is not this wave: at the wave base (44adce61, clean
# worktree) a parallel run HUNG 10 minutes inside CreateWindow and timed out
# (TestSWScenePathtraceMatchesCPUOracle) — a third instability of the same family. The cost is ~15 s on a ~2 min suite; a gate that flakes
# is a gate people learn to ignore. **#3529 is the issue that removes this line** — fixing
# the two tests, or giving every run a private display, is the root fix; serialisation is
# the symptom fix.
#
# deps-check is a prerequisite so `make -C head test` refuses the same way `make gate`
# does, and the last line reports what the run could NOT cover instead of leaving that in
# a warning printed 28 minutes earlier (#3526).
test: native-libs deps-check display-check
	$(TEST_UNDER_DISPLAY) go test -p 1 ./...
	@if [ "$(VULKAN_ICDS)" = "0" ]; then \
		echo "head: ZERO Vulkan ICD manifests - every in-window test SKIPPED, so the ok lines"; \
		echo "head: above do NOT mean the window path was covered. Install mesa-vulkan-drivers."; \
	else \
		echo "head: tests ran on $(HEAD_RUN_DISPLAY) with $(VULKAN_ICDS) Vulkan ICD manifest(s)"; \
	fi

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
