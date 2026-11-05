// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"
#include <cstddef>

// #3803: Rust C aggregates and MSVC ARM64 C++ vector classes use different registers.
// Only scalars and byte pointers cross this boundary; C++ constructs its own vectors.
extern "C" {
void oblikovati_gui_window_position(float x, float y, int condition) {
    ImGui::SetNextWindowPos(ImVec2(x, y), condition);
}
void oblikovati_gui_window_size(float width, float height, int condition) {
    ImGui::SetNextWindowSize(ImVec2(width, height), condition);
}
void oblikovati_gui_cursor_position(float x, float y) {
    ImGui::SetCursorPos(ImVec2(x, y));
}
bool oblikovati_gui_button(const unsigned char* label, float width, float height) {
    return ImGui::Button(reinterpret_cast<const char*>(label), ImVec2(width, height));
}
bool oblikovati_gui_selectable(const unsigned char* label, bool selected, int flags) {
    return ImGui::Selectable(reinterpret_cast<const char*>(label), selected, flags, ImVec2(0, 0));
}
void oblikovati_gui_literal_text(const unsigned char* text, std::size_t length,
                                float x, float y, unsigned int color) {
    const char* begin = reinterpret_cast<const char*>(text);
    ImGui::GetWindowDrawList()->AddText(ImVec2(x, y), color, begin, begin + length);
}
}
