// Unity build of vendored Dear ImGui + the GLFW/Vulkan backends. cgo only compiles
// C/C++ that lives in the package directory, so we pull the vendored translation
// units in here via #include. Include paths to third_party/imgui[/backends] are set
// by the #cgo CPPFLAGS in native.go.
//
// Keeping it one TU also means ImGui's globals are defined once and shared with
// smoke.cpp / app.cpp in the same package link unit.
#include "imgui.cpp"
#include "imgui_draw.cpp"
#include "imgui_tables.cpp"
#include "imgui_widgets.cpp"

// NOTE: the GLFW and Vulkan backends are compiled in their OWN translation units
// (impl_glfw.cpp / impl_vulkan.cpp), NOT included here. imgui_impl_glfw.cpp declares
// a stub `enum VkResult` for its surface helper when real Vulkan headers aren't
// present; pulling both into one TU collides with the real vulkan.h. Keeping them
// separate (as Dear ImGui's own examples do) avoids that.
