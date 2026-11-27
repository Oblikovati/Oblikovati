// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"

extern "C" void oblikovati_gui_document_tab_underline() {
    const float left = ImGui::GetItemRectMin().x;
    const float right = ImGui::GetItemRectMax().x;
    const ImVec2 position = ImGui::GetWindowPos();
    const ImVec2 size = ImGui::GetWindowSize();
    const float bottom = position.y + size.y - 1.0f;
    const ImU32 color = ImGui::ColorConvertFloat4ToU32(ImGui::GetStyleColorVec4(ImGuiCol_NavHighlight));
    ImDrawList* draw = ImGui::GetWindowDrawList();
    // Reserve the bottom of the strip even when ImGui's tab row is shorter.
    draw->PushClipRect(position, ImVec2(position.x + size.x, bottom), false);
    draw->AddRectFilled(ImVec2(left, bottom - 2.0f), ImVec2(right, bottom), color);
    draw->PopClipRect();
}
