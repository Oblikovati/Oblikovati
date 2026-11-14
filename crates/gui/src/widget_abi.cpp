// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"
#include "imgui_internal.h"
#include <cstddef>
#include <cstdint>

// #3803: Rust C aggregates and MSVC ARM64 C++ vector classes use different registers.
// Only scalars and byte pointers cross this boundary; C++ constructs its own vectors.
namespace {
void draw_ribbon_icon(const char* label, std::uint32_t texture, ImVec2 icon, bool large, bool show_caption) {
    const ImVec2 lo = ImGui::GetItemRectMin(), hi = ImGui::GetItemRectMax();
    const char* end = ImGui::FindRenderedTextEnd(label);
    const ImVec2 text = ImGui::CalcTextSize(label, end);
    const ImVec2 at(!show_caption || large ? (lo.x + hi.x - icon.x) * 0.5f : lo.x + 2.0f,
                    !show_caption ? (lo.y + hi.y - icon.y) * 0.5f :
                    large ? lo.y + 2.0f : (lo.y + hi.y - icon.y) * 0.5f);
    const ImVec2 caption(large ? (lo.x + hi.x - text.x) * 0.5f : at.x + icon.x + 3.0f,
                         large ? hi.y - text.y - 2.0f : (lo.y + hi.y - text.y) * 0.5f);
    ImDrawList* draw = ImGui::GetWindowDrawList();
    draw->PushClipRect(lo, hi, true);
    draw->AddImage(reinterpret_cast<ImTextureID>(static_cast<std::uintptr_t>(texture)), at,
                   ImVec2(at.x + icon.x, at.y + icon.y), ImVec2(0,0), ImVec2(1,1), ImGui::GetColorU32(ImVec4(1,1,1,1)));
    if (show_caption) draw->AddText(caption, ImGui::GetColorU32(ImGuiCol_Text), label, end);
    draw->PopClipRect();
}
}
extern "C" {
bool oblikovati_gui_icon_button(const unsigned char* bytes, std::uint32_t texture,
                                float width, float height, float icon_width, float icon_height, bool large, bool show_caption) {
    const char* label = reinterpret_cast<const char*>(bytes);
    // Normal Button owns identity, navigation, disabled state, hit testing and last-item state.
    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0,0,0,0));
    const bool clicked = ImGui::Button(label, ImVec2(width, height));
    ImGui::PopStyleColor();
    if (ImGui::IsItemVisible()) draw_ribbon_icon(label, texture, ImVec2(icon_width, icon_height), large, show_caption);
    return clicked;
}

void oblikovati_gui_fit_dialog(const unsigned char* title, float x, float y, float width, float height) {
    const ImGuiWindow* previous = ImGui::FindWindowByName(reinterpret_cast<const char*>(title));
    // Dock nodes own their members' bounds; fitting an individual member would undo that layout.
    if (previous && previous->DockId != 0)
        return;
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
