// Oblikovati's compile-time configuration for vendored Dear ImGui.
//
// Deliberately minimal: we do NOT inject the cimgui-go interop hooks (MyVec2 /
// MyMatrix44 / goImguiAssertHandler) that ship in upstream-via-cimgui-go's
// imconfig.h, because we compile ImGui ourselves and bind it with our own thin
// cgo layer. Keeping this clean is what makes our build ABI-self-consistent
// (the whole reason the head vendors ImGui instead of linking cimgui-go's
// prebuilt archive). See third_party/imgui/PROVENANCE.md.
#pragma once

// Convenience operators on ImVec2/ImVec4 (used by the Vulkan backend helpers).
#define IMGUI_DEFINE_MATH_OPERATORS
