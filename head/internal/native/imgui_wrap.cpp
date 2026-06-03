// Thin C wrappers over the Dear ImGui widgets the head's chrome uses. Each is a
// near-1:1 pass-through so the chrome LAYOUT lives in Go (head/ui), reading the model
// each frame (ADR-0004/0009). Keeping the binding here — rather than rendering a
// Go-described tree in C++ — is what lets Go own the UI composition. Add wrappers as
// the chrome grows; resist putting logic here.
#include <cstring> // strcmp (theme color-name lookup)
#include "imgui.h"
#include "imgui_internal.h" // SeparatorEx (vertical ribbon-panel divider)

extern "C" {

int  obk_ig_begin_main_menu_bar(void)        { return ImGui::BeginMainMenuBar() ? 1 : 0; }
void obk_ig_end_main_menu_bar(void)          { ImGui::EndMainMenuBar(); }
int  obk_ig_begin_menu(const char* label)    { return ImGui::BeginMenu(label) ? 1 : 0; }
void obk_ig_end_menu(void)                   { ImGui::EndMenu(); }
int  obk_ig_menu_item(const char* label)     { return ImGui::MenuItem(label) ? 1 : 0; }

void obk_ig_set_next_window_pos(float x, float y)  { ImGui::SetNextWindowPos(ImVec2(x, y)); }
void obk_ig_set_next_window_size(float w, float h) { ImGui::SetNextWindowSize(ImVec2(w, h)); }
void obk_ig_set_next_window_size_first_use(float w, float h) {
    ImGui::SetNextWindowSize(ImVec2(w, h), ImGuiCond_FirstUseEver);
}
int  obk_ig_begin(const char* name)          { return ImGui::Begin(name) ? 1 : 0; }
void obk_ig_end(void)                        { ImGui::End(); }
void obk_ig_text(const char* s)              { ImGui::TextUnformatted(s); }
int  obk_ig_button(const char* label)        { return ImGui::Button(label) ? 1 : 0; }
void obk_ig_same_line(void)                  { ImGui::SameLine(); }
void obk_ig_separator(void)                  { ImGui::Separator(); }
void obk_ig_separator_text(const char* s)    { ImGui::SeparatorText(s); }
// begin_group/end_group bracket a layout group so a whole ribbon panel (its button
// row + title) lays out as one item and panels can sit SameLine side-by-side.
void obk_ig_begin_group(void)                { ImGui::BeginGroup(); }
void obk_ig_end_group(void)                  { ImGui::EndGroup(); }
// separator_vertical draws a vertical divider between two horizontally-laid panels.
void obk_ig_separator_vertical(void)         { ImGui::SeparatorEx(ImGuiSeparatorFlags_Vertical); }
// mouse_double_clicked reports a double-click of the given button this frame (used to
// re-open a dimension for editing).
int  obk_ig_mouse_double_clicked(int button) { return ImGui::IsMouseDoubleClicked(button) ? 1 : 0; }

int  obk_ig_begin_tab_bar(const char* id)    { return ImGui::BeginTabBar(id) ? 1 : 0; }
void obk_ig_end_tab_bar(void)                { ImGui::EndTabBar(); }
int  obk_ig_begin_tab_item(const char* l)    { return ImGui::BeginTabItem(l) ? 1 : 0; }
// begin_tab_item_ex optionally forces this tab selected this frame (ImGui's
// SetSelected flag) — used to auto-switch to/from the contextual Sketch tab.
int  obk_ig_begin_tab_item_ex(const char* l, int setSelected) {
    ImGuiTabItemFlags f = setSelected ? ImGuiTabItemFlags_SetSelected : 0;
    return ImGui::BeginTabItem(l, nullptr, f) ? 1 : 0;
}
void obk_ig_end_tab_item(void)               { ImGui::EndTabItem(); }
// selectable is a clickable browser row; returns 1 on click. selected draws it
// highlighted (the current selection).
int  obk_ig_selectable(const char* label, int selected) {
    return ImGui::Selectable(label, selected != 0) ? 1 : 0;
}

int  obk_ig_collapsing_header(const char* l) { return ImGui::CollapsingHeader(l) ? 1 : 0; }
int  obk_ig_tree_node(const char* label)     { return ImGui::TreeNode(label) ? 1 : 0; }
void obk_ig_tree_pop(void)                   { ImGui::TreePop(); }
void obk_ig_bullet_text(const char* s)       { ImGui::BulletText("%s", s); }
void obk_ig_set_item_tooltip(const char* s)  { ImGui::SetItemTooltip("%s", s); }

// begin_popup_context_item opens (on right-click of the just-drawn item) and begins a
// context-menu popup keyed by id; returns 1 while the popup is open so the caller fills
// it with menu_items, then calls end_popup. set_scroll_here_y scrolls the current window
// so the last item is centered — used to reveal the browser node synced to the selection.
int  obk_ig_begin_popup_context_item(const char* id) { return ImGui::BeginPopupContextItem(id) ? 1 : 0; }
void obk_ig_end_popup(void)                  { ImGui::EndPopup(); }
void obk_ig_set_scroll_here_y(void)          { ImGui::SetScrollHereY(0.5f); }

// Preference widgets: a float/int field and a checkbox. Each returns 1 when the value
// changed this frame and writes the new value back through the pointer.
int  obk_ig_input_float(const char* label, float* v) { return ImGui::InputFloat(label, v) ? 1 : 0; }
int  obk_ig_input_double(const char* label, double* v) { return ImGui::InputDouble(label, v) ? 1 : 0; }
int  obk_ig_input_int(const char* label, int* v)     { return ImGui::InputInt(label, v) ? 1 : 0; }
int  obk_ig_input_text(const char* label, char* buf, int buf_size) { return ImGui::InputText(label, buf, (size_t)buf_size) ? 1 : 0; }
int  obk_ig_checkbox(const char* label, int* v) {
    bool b = (*v != 0);
    bool changed = ImGui::Checkbox(label, &b);
    *v = b ? 1 : 0;
    return changed ? 1 : 0;
}

void obk_ig_begin_disabled(int disabled)     { ImGui::BeginDisabled(disabled != 0); }
void obk_ig_end_disabled(void)               { ImGui::EndDisabled(); }

// --- Theming (ADR-0021) -----------------------------------------------------------
// obk_col_index maps an ImGui color NAME to its ImGuiCol_ enum so the Go side never
// hardcodes the numeric index (it shifts between ImGui versions). Only the slots the
// theme drives are listed; an unknown name returns -1 (ignored). Several names may share
// a token on the Go side (e.g. an accent token sets TabSelected, CheckMark, SliderGrab).
static int obk_col_index(const char* name) {
    struct { const char* n; int c; } table[] = {
        {"Text", ImGuiCol_Text}, {"TextDisabled", ImGuiCol_TextDisabled},
        {"WindowBg", ImGuiCol_WindowBg}, {"ChildBg", ImGuiCol_ChildBg},
        {"PopupBg", ImGuiCol_PopupBg}, {"Border", ImGuiCol_Border},
        {"FrameBg", ImGuiCol_FrameBg}, {"FrameBgHovered", ImGuiCol_FrameBgHovered},
        {"FrameBgActive", ImGuiCol_FrameBgActive}, {"TitleBg", ImGuiCol_TitleBg},
        {"TitleBgActive", ImGuiCol_TitleBgActive}, {"TitleBgCollapsed", ImGuiCol_TitleBgCollapsed},
        {"MenuBarBg", ImGuiCol_MenuBarBg}, {"ScrollbarBg", ImGuiCol_ScrollbarBg},
        {"ScrollbarGrab", ImGuiCol_ScrollbarGrab}, {"ScrollbarGrabHovered", ImGuiCol_ScrollbarGrabHovered},
        {"ScrollbarGrabActive", ImGuiCol_ScrollbarGrabActive}, {"CheckMark", ImGuiCol_CheckMark},
        {"SliderGrab", ImGuiCol_SliderGrab}, {"SliderGrabActive", ImGuiCol_SliderGrabActive},
        {"Button", ImGuiCol_Button}, {"ButtonHovered", ImGuiCol_ButtonHovered},
        {"ButtonActive", ImGuiCol_ButtonActive}, {"Header", ImGuiCol_Header},
        {"HeaderHovered", ImGuiCol_HeaderHovered}, {"HeaderActive", ImGuiCol_HeaderActive},
        {"Separator", ImGuiCol_Separator}, {"SeparatorHovered", ImGuiCol_SeparatorHovered},
        {"SeparatorActive", ImGuiCol_SeparatorActive}, {"Tab", ImGuiCol_Tab},
        {"TabHovered", ImGuiCol_TabHovered}, {"TabSelected", ImGuiCol_TabSelected},
        {"TabDimmed", ImGuiCol_TabDimmed}, {"TabDimmedSelected", ImGuiCol_TabDimmedSelected},
        {"TextSelectedBg", ImGuiCol_TextSelectedBg},
    };
    for (auto& e : table) {
        if (strcmp(e.n, name) == 0) return e.c;
    }
    return -1;
}

void obk_ig_set_style_color(const char* name, float r, float g, float b, float a) {
    int idx = obk_col_index(name);
    if (idx < 0) return;
    ImGui::GetStyle().Colors[idx] = ImVec4(r, g, b, a);
}

int  obk_ig_color_edit4(const char* label, float* rgba) {
    return ImGui::ColorEdit4(label, rgba) ? 1 : 0;
}
int  obk_ig_begin_combo(const char* label, const char* preview) {
    return ImGui::BeginCombo(label, preview) ? 1 : 0;
}
void obk_ig_end_combo(void)                  { ImGui::EndCombo(); }

// Table verbs (the Parameters dialog grid). A fixed flag set gives the dialog its
// bordered, row-striped, resizable, vertically scrolling grid; the Go side owns layout.
int  obk_ig_begin_table(const char* id, int columns, float outer_w, float outer_h) {
    ImGuiTableFlags flags = ImGuiTableFlags_Borders | ImGuiTableFlags_RowBg |
                            ImGuiTableFlags_Resizable | ImGuiTableFlags_ScrollY;
    return ImGui::BeginTable(id, columns, flags, ImVec2(outer_w, outer_h)) ? 1 : 0;
}
void obk_ig_table_setup_column(const char* label)      { ImGui::TableSetupColumn(label); }
void obk_ig_table_setup_scroll_freeze(int cols, int rows) { ImGui::TableSetupScrollFreeze(cols, rows); }
void obk_ig_table_headers_row(void)                    { ImGui::TableHeadersRow(); }
void obk_ig_table_next_row(void)                       { ImGui::TableNextRow(); }
int  obk_ig_table_next_column(void)                    { return ImGui::TableNextColumn() ? 1 : 0; }
void obk_ig_end_table(void)                            { ImGui::EndTable(); }
void obk_ig_set_next_item_width(float w)               { ImGui::SetNextItemWidth(w); }
void obk_ig_push_id_int(int id)                        { ImGui::PushID(id); }
void obk_ig_pop_id(void)                               { ImGui::PopID(); }
int  obk_ig_is_item_deactivated_after_edit(void)       { return ImGui::IsItemDeactivatedAfterEdit() ? 1 : 0; }

// want_capture_mouse reports whether ImGui consumed the pointer this frame, so the Go
// loop knows when NOT to forward a click to the 3D viewport (picking).
int  obk_ig_want_capture_mouse(void)         { return ImGui::GetIO().WantCaptureMouse ? 1 : 0; }

// Viewport navigation input. invisible_button reserves an input-capturing region (it
// accepts any mouse button, so a middle-/right-drag still makes the item "active");
// the Go side reads is_item_active/hovered + mouse deltas/wheel + Shift to drive the
// camera (orbit/pan/zoom). get/set_cursor_pos let Go draw the viewport image back over
// the button. Layout stays in Go; this only exposes the primitives.
int  obk_ig_invisible_button(const char* id, float w, float h) {
    return ImGui::InvisibleButton(id, ImVec2(w, h),
        ImGuiButtonFlags_MouseButtonLeft | ImGuiButtonFlags_MouseButtonRight |
        ImGuiButtonFlags_MouseButtonMiddle) ? 1 : 0;
}
int  obk_ig_is_item_active(void)             { return ImGui::IsItemActive() ? 1 : 0; }
int  obk_ig_is_item_hovered(void)            { return ImGui::IsItemHovered() ? 1 : 0; }
int  obk_ig_mouse_down(int button)           { return ImGui::IsMouseDown((ImGuiMouseButton)button) ? 1 : 0; }
int  obk_ig_is_item_clicked(int button)      { return ImGui::IsItemClicked((ImGuiMouseButton)button) ? 1 : 0; }
void obk_ig_item_rect_min(float* x, float* y) {
    ImVec2 p = ImGui::GetItemRectMin();
    *x = p.x;
    *y = p.y;
}
void obk_ig_mouse_pos(float* x, float* y) {
    ImVec2 p = ImGui::GetIO().MousePos;
    *x = p.x;
    *y = p.y;
}
int  obk_ig_key_shift(void)                  { return ImGui::GetIO().KeyShift ? 1 : 0; }
int  obk_ig_key_ctrl(void)                   { return ImGui::GetIO().KeyCtrl ? 1 : 0; }
// escape_pressed fires once on the frame Esc is pressed (cancel the active tool).
int  obk_ig_escape_pressed(void)             { return ImGui::IsKeyPressed(ImGuiKey_Escape) ? 1 : 0; }
float obk_ig_mouse_wheel(void)               { return ImGui::GetIO().MouseWheel; }
float obk_ig_delta_time(void)                { return ImGui::GetIO().DeltaTime; }
void obk_ig_mouse_delta(float* dx, float* dy) {
    ImVec2 d = ImGui::GetIO().MouseDelta;
    *dx = d.x;
    *dy = d.y;
}
void obk_ig_get_cursor_pos(float* x, float* y) {
    ImVec2 p = ImGui::GetCursorPos();
    *x = p.x;
    *y = p.y;
}
void obk_ig_set_cursor_pos(float x, float y)  { ImGui::SetCursorPos(ImVec2(x, y)); }

// image draws a previously-rendered texture (the 3D viewport color image) at the given
// size; content_region_avail reports the free space in the current window so the
// viewport can be rendered at panel resolution.
void obk_ig_image(unsigned long long tex, float w, float h) {
    ImGui::Image((ImTextureID)tex, ImVec2(w, h));
}
// image_button draws a clickable icon (a ribbon button) tinted by (r,g,b,a) so a theme
// recolors the monochrome glyph without re-rasterizing it; the str_id keeps each
// button's ImGui id distinct. Returns 1 if clicked this frame.
int obk_ig_image_button(const char* id, unsigned long long tex, float w, float h,
                        float r, float g, float b, float a) {
    return ImGui::ImageButton(id, (ImTextureID)tex, ImVec2(w, h),
                              ImVec2(0, 0), ImVec2(1, 1), ImVec4(0, 0, 0, 0),
                              ImVec4(r, g, b, a)) ? 1 : 0;
}
void obk_ig_content_region_avail(float* w, float* h) {
    ImVec2 a = ImGui::GetContentRegionAvail();
    *w = a.x;
    *h = a.y;
}

} // extern "C"
