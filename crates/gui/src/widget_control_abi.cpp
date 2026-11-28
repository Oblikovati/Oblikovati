// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"
#include <cstddef>

// #3803: Construct ImVec2 inside C++ so ARM64 Windows never passes C++ vector classes across the Rust ABI.
extern "C" {
bool oblikovati_gui_begin_table(const char* id, int columns, int flags,
                                float outer_width, float outer_height, float inner_width) {
    return ImGui::BeginTable(id, columns, flags, ImVec2(outer_width, outer_height), inner_width);
}

bool oblikovati_gui_input_text_multiline(const char* label, char* buf, std::size_t buf_size,
                                         float width, float height, int flags,
                                         ImGuiInputTextCallback callback, void* user_data) {
    return ImGui::InputTextMultiline(label, buf, buf_size, ImVec2(width, height), flags, callback,
                                     user_data);
}

bool oblikovati_gui_selectable_sized(const char* label, bool selected, int flags,
                                     float width, float height) {
    return ImGui::Selectable(label, selected, flags, ImVec2(width, height));
}
}
