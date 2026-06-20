// The "head" is the windowed application: a Vulkan 1.3 viewport + Dear ImGui chrome
// (ADR-0004/0005). Separate module so its cgo build never touches the pure-Go,
// headless-tested core. Dear ImGui is vendored (third_party/imgui, MIT) and compiled
// here with our own imconfig, so the ABI is fully under our control; the core is
// consumed via the replace directive below.
module oblikovati.org/head

go 1.22

require (
	github.com/srwiley/oksvg v0.0.0-20221011165216-be6e8873101c
	github.com/srwiley/rasterx v0.0.0-20220730225603-2ab79fcdd4ef
	oblikovati.org v0.0.0
	oblikovati.org/api v0.80.1
)

require golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // icon glyph normalization (x/image/draw)

require (
	github.com/yuin/gopher-lua v1.1.1 // indirect
	golang.org/x/net v0.0.0-20211118161319-6a13c67c3ce4 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	oblikovati.org/motor-designer v0.0.0
)

// The core stays a relative replace: head is a submodule of this same repo, so
// `../` is the repo root regardless of where the repo is checked out. The
// Apache-2.0 api contract (sibling repo ../../Oblikovati.API) is resolved via the
// go.work workspace at the app repo root instead of a committed replace.
replace oblikovati.org => ../
