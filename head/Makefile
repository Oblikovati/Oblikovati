# The head is a separate cgo module (Vulkan + GLFW + vendored Dear ImGui). It needs a
# C/C++ toolchain, the Vulkan loader/headers, GLFW, and a Go >= 1.24 toolchain. The
# pinned GOTOOLCHAIN avoids re-resolving over the network on every invocation.
export CGO_ENABLED := 1
export GOTOOLCHAIN := go1.24.0

.PHONY: build smoke run shaders clean

# shaders recompiles the viewport GLSL to the embedded SPIR-V (needs glslangValidator
# from glslang-tools). Run after editing internal/native/shaders/*.vert|frag.
shaders:
	cd internal/native/shaders && \
		glslangValidator -V mesh.vert -o mesh.vert.spv && \
		glslangValidator -V mesh.frag -o mesh.frag.spv && \
		glslangValidator -V skybox.vert -o skybox.vert.spv && \
		glslangValidator -V skybox.frag -o skybox.frag.spv

# build compiles every head command (proves the cgo/Vulkan stack links).
build:
	go build ./...

# smoke opens a real GLFW+Vulkan window, renders a few Dear ImGui frames, and exits 0
# if the whole native stack initialized. Requires a display (X11/Wayland) + a Vulkan
# driver (hardware or lavapipe).
smoke:
	go run ./cmd/headsmoke -frames 5

# run launches the application. It points the add-in loader at ./addins (where
# `make install` in add-in/oblikovati-mcp-bridge drops the built library), so any
# installed add-in (e.g. the MCP bridge) loads at startup. Override with
# OBK_ADDINS_DIR, and OBK_MCP_ADDR to move the MCP endpoint off 127.0.0.1:7800.
run:
	OBK_ADDINS_DIR=$(CURDIR)/addins go run ./cmd/oblikovati-head

# run-watch launches the app under a relaunch loop with OBK_ADDIN_AUTORESTART=1, so a
# rebuilt add-in (`make install` in the bridge) makes the app exit cleanly and come
# straight back with the new library — the safe stand-in for in-process hot-reload (a
# Go c-shared add-in cannot be swapped in a live process). The head binary is built
# ONCE and the prebuilt binary is relaunched, so a .so swap restarts in ~1s (no cgo
# recompile). Re-run this target after changing head source. Ctrl-C twice to stop.
run-watch:
	go build -o oblikovati-head ./cmd/oblikovati-head
	@while true; do \
		OBK_ADDINS_DIR=$(CURDIR)/addins OBK_ADDIN_AUTORESTART=1 ./oblikovati-head || true; \
		echo "oblikovati-head exited; relaunching (Ctrl-C to stop)"; \
		sleep 1; \
	done

clean:
	go clean ./...
