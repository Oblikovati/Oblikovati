// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"
#include "imgui_internal.h"
#include <cstddef>

// #3803: Rust C aggregates and MSVC ARM64 C++ vector classes use different registers.
// Only scalars and byte pointers cross this boundary; C++ constructs its own vectors.
extern "C" {
void oblikovati_gui_fit_dialog(const unsigned char* title, float x, float y, float width, float height) {
    const ImGuiWindow* previous = ImGui::FindWindowByName(reinterpret_cast<const char*>(title));
    const ImVec2 position = previous ? previous->Pos : ImVec2(x, y);
    const ImVec2 size = previous ? previous->SizeFull : ImVec2(width, height);
    const ImVec2 display = ImGui::GetIO().DisplaySize;
    const ImVec2 fitted_size(ImClamp(size.x, 1.0f, ImMax(1.0f, display.x - 32.0f)),
                             ImClamp(size.y, 1.0f, ImMax(1.0f, display.y - 32.0f)));
    const ImVec2 fitted_position(ImClamp(position.x, 0.0f, ImMax(0.0f, display.x - fitted_size.x - 16.0f)),
                                 ImClamp(position.y, 0.0f, ImMax(0.0f, display.y - fitted_size.y - 16.0f)));
    if (!previous || position.x != fitted_position.x || position.y != fitted_position.y)
        ImGui::SetNextWindowPos(fitted_position, ImGuiCond_Always);
    if (!previous || size.x != fitted_size.x || size.y != fitted_size.y)
        ImGui::SetNextWindowSize(fitted_size, ImGuiCond_Always);
}

void oblikovati_gui_window_position(float x, float y, int condition) {
    ImGui::SetNextWindowPos(ImVec2(x, y), condition);
}
void oblikovati_gui_window_size(float width, float height, int condition) {
    ImGui::SetNextWindowSize(ImVec2(width, height), condition);
}
void oblikovati_gui_cursor_position(float x, float y) {
    ImGui::SetCursorPos(ImVec2(x, y));
}
bool oblikovati_gui_button(const unsigned char* label, float width, float height, bool selected) {
    if (selected) ImGui::PushStyleColor(ImGuiCol_Button, ImGui::GetStyleColorVec4(ImGuiCol_ButtonActive));
    const bool clicked = ImGui::Button(reinterpret_cast<const char*>(label), ImVec2(width, height));
    if (selected) ImGui::PopStyleColor();
    return clicked;
}
bool oblikovati_gui_begin_scroll_region(const unsigned char* identity, float width, float height) {
    return ImGui::BeginChild(reinterpret_cast<const char*>(identity), ImVec2(width, height), false,
                             ImGuiWindowFlags_HorizontalScrollbar);
}
void oblikovati_gui_dummy(float width, float height) {
    ImGui::Dummy(ImVec2(width, height));
}
bool oblikovati_gui_selectable(const unsigned char* label, bool selected, int flags) {
    return ImGui::Selectable(reinterpret_cast<const char*>(label), selected, flags, ImVec2(0, 0));
}
void oblikovati_gui_literal_text(const unsigned char* text, std::size_t length,
                                float x, float y, unsigned int color, bool background) {
    const char* begin = reinterpret_cast<const char*>(text);
    // #3839: annotations need the same scalar ABI even before a window is active.
    ImDrawList* draw = background ? ImGui::GetBackgroundDrawList() : ImGui::GetWindowDrawList();
    draw->AddText(ImVec2(x, y), color, begin, begin + length);
}
}
