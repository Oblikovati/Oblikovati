// Thin C wrappers over the Dear ImGui widgets the head's chrome uses. Each is a
// near-1:1 pass-through so the chrome LAYOUT lives in Go (head/ui), reading the model
// each frame (ADR-0004/0009). Keeping the binding here — rather than rendering a
// Go-described tree in C++ — is what lets Go own the UI composition. Add wrappers as
// the chrome grows; resist putting logic here.
#include <cstring> // strcmp (theme color-name lookup)
#include <cstdio>  // snprintf (bounds-safe key-name packing)
#include "imgui.h"
#include "imgui_internal.h" // SeparatorEx (vertical ribbon-panel divider)

extern "C" {

int  obk_ig_begin_main_menu_bar(void)        { return ImGui::BeginMainMenuBar() ? 1 : 0; }
void obk_ig_end_main_menu_bar(void)          { ImGui::EndMainMenuBar(); }
int  obk_ig_begin_menu(const char* label)    { return ImGui::BeginMenu(label) ? 1 : 0; }
void obk_ig_end_menu(void)                   { ImGui::EndMenu(); }
int  obk_ig_menu_item(const char* label)     { return ImGui::MenuItem(label) ? 1 : 0; }
// menu_item_ex adds a right-aligned shortcut hint and an enabled flag (a disabled item
// greys out and cannot be clicked). A "" shortcut passes NULL so none is drawn.
int  obk_ig_menu_item_ex(const char* label, const char* shortcut, int enabled) {
    const char* sc = (shortcut && shortcut[0]) ? shortcut : nullptr;
    return ImGui::MenuItem(label, sc, false, enabled != 0) ? 1 : 0;
}

void obk_ig_set_next_window_pos(float x, float y)  { ImGui::SetNextWindowPos(ImVec2(x, y)); }
// Center the next window on the main viewport. A (0.5,0.5) pivot anchors the window's
// midpoint to the viewport midpoint, so auto-sized content centers without us knowing
// the window's size, and GetCenter() (not ImVec2(w/2,h/2)) keeps it correct off a
// non-zero viewport origin under multi-monitor. Re-applied each frame (ImGuiCond_Always)
// so a resized main window keeps the prompt centred (Oblikovati#1474).
void obk_ig_center_next_window(void) {
    ImVec2 center = ImGui::GetMainViewport()->GetCenter();
    ImGui::SetNextWindowPos(center, ImGuiCond_Always, ImVec2(0.5f, 0.5f));
}
void obk_ig_set_next_window_size(float w, float h) { ImGui::SetNextWindowSize(ImVec2(w, h)); }
void obk_ig_set_next_window_size_first_use(float w, float h) {
    ImGui::SetNextWindowSize(ImVec2(w, h), ImGuiCond_FirstUseEver);
}
int  obk_ig_begin(const char* name)          { return ImGui::Begin(name) ? 1 : 0; }
int  obk_ig_begin_closable(const char* name, int* open) {
    bool p_open = (*open != 0);
    int visible = ImGui::Begin(name, &p_open) ? 1 : 0;
    *open = p_open ? 1 : 0;
    return visible;
}
void obk_ig_end(void)                        { ImGui::End(); }
void obk_ig_text(const char* s)              { ImGui::TextUnformatted(s); }
// "%s" guard: the text is user/add-in supplied and may contain '%' — never pass it as the format.
void obk_ig_text_wrapped(const char* s)      { ImGui::TextWrapped("%s", s); }
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

// begin_ribbon_band pins a full-width, fixed-height band to the top of the main
// viewport (under the menu bar) via BeginViewportSideBar — the side-bar claims its
// slice of the viewport work area, so the dockspace created afterwards lays out below
// it. The band cannot be moved, resized, collapsed, or docked: the ribbon is window
// chrome, not a palette. Pair with obk_ig_end regardless of the return value.
int  obk_ig_begin_ribbon_band(const char* name, float height) {
    // HorizontalScrollbar (not NoScrollbar) so a too-narrow window shows a scrollbar at the band
    // bottom and the mouse wheel scrolls the overflowing buttons into view (Oblikovati#1471). The
    // caller reserves the scrollbar's height so the panel-name strip is never hidden behind it.
    ImGuiWindowFlags flags = ImGuiWindowFlags_HorizontalScrollbar | ImGuiWindowFlags_NoSavedSettings;
    return ImGui::BeginViewportSideBar(name, ImGui::GetMainViewport(), ImGuiDir_Up,
        height, flags) ? 1 : 0;
}
// scroll_max_x reports the current window's maximum horizontal scroll — >0 exactly when its content
// overflows the window width, i.e. the ribbon's horizontal scrollbar is showing (#1471).
float obk_ig_scroll_max_x(void) { return ImGui::GetScrollMaxX(); }
// scrollbar_size is the style thickness of a scrollbar, so the ribbon can grow its band height by
// exactly that to seat the horizontal scrollbar without covering content (#1471).
float obk_ig_scrollbar_size(void) { return ImGui::GetStyle().ScrollbarSize; }

// Cursor/layout probes for the ribbon's manual placement (screen space): the panel
// name strip is pinned at a fixed band Y and centered under its button block, which
// needs the block's rect, the text width, and a screen-space cursor write.
void obk_ig_get_cursor_screen_pos(float* x, float* y) {
    ImVec2 p = ImGui::GetCursorScreenPos();
    *x = p.x; *y = p.y;
}
void obk_ig_set_cursor_screen_pos(float x, float y) { ImGui::SetCursorScreenPos(ImVec2(x, y)); }
void obk_ig_item_rect_max(float* x, float* y) {
    ImVec2 p = ImGui::GetItemRectMax();
    *x = p.x; *y = p.y;
}
float obk_ig_calc_text_width(const char* s)  { return ImGui::CalcTextSize(s).x; }
float obk_ig_frame_height(void)              { return ImGui::GetFrameHeight(); }
float obk_ig_text_line_height(void)          { return ImGui::GetTextLineHeight(); }
// style_metrics reports the paddings/spacings the Go layout math needs (frame padding,
// item spacing, window padding) so pixel constants in Go stay in sync with the style.
void obk_ig_style_metrics(float* fpx, float* fpy, float* isx, float* isy, float* wpx, float* wpy) {
    const ImGuiStyle& st = ImGui::GetStyle();
    *fpx = st.FramePadding.x; *fpy = st.FramePadding.y;
    *isx = st.ItemSpacing.x;  *isy = st.ItemSpacing.y;
    *wpx = st.WindowPadding.x; *wpy = st.WindowPadding.y;
}
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
// begin_tab_item_closable is the same tab primitive with ImGui's close affordance.
// It writes open=0 on the frame the close button is clicked.
int  obk_ig_begin_tab_item_closable(const char* l, int setSelected, int* open) {
    bool p_open = (*open != 0);
    ImGuiTabItemFlags f = setSelected ? ImGuiTabItemFlags_SetSelected : 0;
    bool visible = ImGui::BeginTabItem(l, &p_open, f);
    *open = p_open ? 1 : 0;
    return visible ? 1 : 0;
}
void obk_ig_end_tab_item(void)               { ImGui::EndTabItem(); }
// selectable is a clickable browser row; returns 1 on click. selected draws it
// highlighted (the current selection).
int  obk_ig_selectable(const char* label, int selected) {
    return ImGui::Selectable(label, selected != 0) ? 1 : 0;
}

// Drag-and-drop: reorder a list by dragging a row onto another (#1222). The payload is a single
// int (the dragged row index); SetDragDropPayload copies it internally so a local is safe.
int  obk_ig_begin_drag_drop_source(void) { return ImGui::BeginDragDropSource() ? 1 : 0; }
void obk_ig_set_drag_drop_payload_int(const char* type, int v) {
    ImGui::SetDragDropPayload(type, &v, sizeof(int));
}
void obk_ig_end_drag_drop_source(void)   { ImGui::EndDragDropSource(); }
int  obk_ig_begin_drag_drop_target(void) { return ImGui::BeginDragDropTarget() ? 1 : 0; }
// Writes *out and returns 1 when a payload of this type was dropped on the current target.
int  obk_ig_accept_drag_drop_payload_int(const char* type, int* out) {
    const ImGuiPayload* pl = ImGui::AcceptDragDropPayload(type);
    if (pl && pl->DataSize == (int)sizeof(int)) { *out = *(const int*)pl->Data; return 1; }
    return 0;
}
void obk_ig_end_drag_drop_target(void)   { ImGui::EndDragDropTarget(); }

int  obk_ig_collapsing_header(const char* l) { return ImGui::CollapsingHeader(l) ? 1 : 0; }
void obk_ig_set_next_item_open(int open, int first_use) {
    ImGui::SetNextItemOpen(open != 0, first_use != 0 ? ImGuiCond_FirstUseEver : ImGuiCond_Always);
}
int  obk_ig_tree_node(const char* label)     { return ImGui::TreeNode(label) ? 1 : 0; }
// tree_node_selectable is an expandable AND clickable row: OpenOnArrow keeps the label
// click free for selection (the caller reads IsItemClicked), the arrow toggles open, and
// SpanAvailWidth makes the whole row the hit target. Selected draws it highlighted.
int  obk_ig_tree_node_selectable(const char* label, int selected) {
    ImGuiTreeNodeFlags flags = ImGuiTreeNodeFlags_OpenOnArrow | ImGuiTreeNodeFlags_SpanAvailWidth;
    if (selected != 0) flags |= ImGuiTreeNodeFlags_Selected;
    return ImGui::TreeNodeEx(label, flags) ? 1 : 0;
}
void obk_ig_tree_pop(void)                   { ImGui::TreePop(); }
void obk_ig_bullet_text(const char* s)       { ImGui::BulletText("%s", s); }

// ImGuiListClipper virtualizes a uniform-height row list: only the rows inside the scroll viewport
// are submitted (M34-F3, the large model browser). The render loop is single-threaded and these
// flat leaf lists never nest a clipper inside another, so one static instance is reused per frame.
static ImGuiListClipper g_obk_clipper;
void obk_ig_clipper_begin(int items)         { g_obk_clipper.Begin(items, -1.0f); }
void obk_ig_clipper_include_item(int item)   { g_obk_clipper.IncludeItemByIndex(item); }
int  obk_ig_clipper_step(void)               { return g_obk_clipper.Step() ? 1 : 0; }
int  obk_ig_clipper_display_start(void)      { return g_obk_clipper.DisplayStart; }
int  obk_ig_clipper_display_end(void)        { return g_obk_clipper.DisplayEnd; }
void obk_ig_clipper_end(void)                { g_obk_clipper.End(); }
void obk_ig_indent(float w)                  { ImGui::Indent(w); }
void obk_ig_unindent(float w)                { ImGui::Unindent(w); }
void obk_ig_set_item_tooltip(const char* s)  { ImGui::SetItemTooltip("%s", s); }
// progress_bar draws a determinate bar of the given pixel width; overlay text ("")
// falls back to ImGui's percentage.
void obk_ig_progress_bar(float fraction, float w, const char* overlay) {
    const char* ov = (overlay && overlay[0]) ? overlay : NULL;
    ImGui::ProgressBar(fraction, ImVec2(w, 0.0f), ov);
}
// main_viewport_size reports the main viewport's size (toast/modal placement).
void obk_ig_main_viewport_size(float* w, float* h) {
    ImVec2 sz = ImGui::GetMainViewport()->Size;
    *w = sz.x; *h = sz.y;
}
// main_viewport_center is the anchor obk_ig_center_next_window pivots to; exposed so the
// centering math is regression-testable (Oblikovati#1474).
void obk_ig_main_viewport_center(float* x, float* y) {
    ImVec2 c = ImGui::GetMainViewport()->GetCenter();
    *x = c.x; *y = c.y;
}
// window_pos reads the current window's top-left (valid between Begin/End) — lets a test
// confirm a centred window landed where the pivot put it.
void obk_ig_window_pos(float* x, float* y) {
    ImVec2 p = ImGui::GetWindowPos();
    *x = p.x; *y = p.y;
}
// hover_seconds reports how long the last item has been hovered (the progressive
// tooltip's expand timer).
float obk_ig_hover_seconds(void) {
    return ImGui::IsItemHovered() ? ImGui::GetCurrentContext()->HoveredIdTimer : 0.0f;
}

// begin_popup_context_item opens (on right-click of the just-drawn item) and begins a
// context-menu popup keyed by id; returns 1 while the popup is open so the caller fills
// it with menu_items, then calls end_popup. set_scroll_here_y scrolls the current window
// so the last item is centered — used to reveal the browser node synced to the selection.
int  obk_ig_begin_popup_context_item(const char* id) { return ImGui::BeginPopupContextItem(id) ? 1 : 0; }
void obk_ig_open_popup(const char* id)       { ImGui::OpenPopup(id); }
int  obk_ig_begin_popup(const char* id)      { return ImGui::BeginPopup(id) ? 1 : 0; }
void obk_ig_close_current_popup(void)               { ImGui::CloseCurrentPopup(); }
void obk_ig_end_popup(void)                  { ImGui::EndPopup(); }
void obk_ig_set_scroll_here_y(void)          { ImGui::SetScrollHereY(0.5f); }
void obk_ig_scroll_to_bottom(void)           { ImGui::SetScrollHereY(1.0f); }

// Preference widgets: a float/int field and a checkbox. Each returns 1 when the value
// changed this frame and writes the new value back through the pointer.
int  obk_ig_input_float(const char* label, float* v) { return ImGui::InputFloat(label, v) ? 1 : 0; }
int  obk_ig_input_double(const char* label, double* v) { return ImGui::InputDouble(label, v) ? 1 : 0; }
int  obk_ig_input_int(const char* label, int* v)     { return ImGui::InputInt(label, v) ? 1 : 0; }
int  obk_ig_input_text(const char* label, char* buf, int buf_size) { return ImGui::InputText(label, buf, (size_t)buf_size) ? 1 : 0; }
int  obk_ig_input_text_submit(const char* label, char* buf, int buf_size) { return ImGui::InputText(label, buf, (size_t)buf_size, ImGuiInputTextFlags_EnterReturnsTrue) ? 1 : 0; }

// ObkCmdNav backs obk_ig_input_text_command: the command-recall history (with a caller-owned
// cursor, n meaning "the empty current line") plus the live autocomplete candidates (with a
// caller-owned selected index). It is stack-local per call. When candidates are present Up/Down
// move the selection and Tab completes; with none, Up/Down do shell-style history recall (M26).
struct ObkCmdNav {
    const char** hist; int nHist; int* histCursor;
    const char** comps; int nComps; int* compSel;
};

// obk_ig_cmd_cb handles both InputText callbacks: CallbackHistory (Up/Down) and
// CallbackCompletion (Tab). With an active completion list, Up/Down move the highlighted
// candidate (buffer untouched) and Tab replaces the buffer with it; otherwise Up/Down walk
// the command history into the buffer.
static int obk_ig_cmd_cb(ImGuiInputTextCallbackData* data) {
    ObkCmdNav* n = (ObkCmdNav*)data->UserData;
    if (data->EventFlag == ImGuiInputTextFlags_CallbackHistory) {
        if (n->nComps > 0) { // move the completion selection only
            int c = *n->compSel + (data->EventKey == ImGuiKey_UpArrow ? -1 : +1);
            if (c < 0) c = 0;
            if (c > n->nComps - 1) c = n->nComps - 1;
            *n->compSel = c;
            return 0;
        }
        if (n->nHist <= 0) return 0; // history recall into the buffer
        int c = *n->histCursor + (data->EventKey == ImGuiKey_UpArrow ? -1 : +1);
        if (c < 0) c = 0;
        if (c > n->nHist) c = n->nHist;
        *n->histCursor = c;
        const char* repl = (c == n->nHist) ? "" : n->hist[c];
        data->DeleteChars(0, data->BufTextLen);
        data->InsertChars(0, repl);
        return 0;
    }
    if (data->EventFlag == ImGuiInputTextFlags_CallbackCompletion) {
        if (n->nComps > 0 && *n->compSel >= 0 && *n->compSel < n->nComps) {
            data->DeleteChars(0, data->BufTextLen);
            data->InsertChars(0, n->comps[*n->compSel]);
        }
        return 0;
    }
    return 0;
}

// obk_ig_input_text_command is InputTextSubmit with Tab autocompletion and Up/Down navigation
// (the completion list when one is shown, else command history). hist/comps are the recall and
// candidate lists; histCursor/compSel are caller-owned indices persisted across frames. Returns
// 1 on the frame Enter is pressed.
int  obk_ig_input_text_command(const char* label, char* buf, int buf_size,
                               const char** hist, int nHist, int* histCursor,
                               const char** comps, int nComps, int* compSel) {
    ObkCmdNav nav{hist, nHist, histCursor, comps, nComps, compSel};
    return ImGui::InputText(label, buf, (size_t)buf_size,
        ImGuiInputTextFlags_EnterReturnsTrue | ImGuiInputTextFlags_CallbackHistory |
            ImGuiInputTextFlags_CallbackCompletion,
        obk_ig_cmd_cb, &nav) ? 1 : 0;
}

void obk_ig_set_keyboard_focus_here(void) { ImGui::SetKeyboardFocusHere(); }
int  obk_ig_input_text_multiline(const char* label, char* buf, int buf_size, float w, float h) {
    return ImGui::InputTextMultiline(label, buf, (size_t)buf_size, ImVec2(w, h)) ? 1 : 0;
}
int  obk_ig_begin_child(const char* id, float w, float h, int border) {
    return ImGui::BeginChild(id, ImVec2(w, h), border ? ImGuiChildFlags_Borders : 0) ? 1 : 0;
}
void obk_ig_end_child(void) { ImGui::EndChild(); }
int  obk_ig_checkbox(const char* label, int* v) {
    bool b = (*v != 0);
    bool changed = ImGui::Checkbox(label, &b);
    *v = b ? 1 : 0;
    return changed ? 1 : 0;
}
int  obk_ig_slider_float(const char* label, float* v, float lo, float hi) {
    return ImGui::SliderFloat(label, v, lo, hi) ? 1 : 0;
}
int  obk_ig_slider_float_fmt(const char* label, float* v, float lo, float hi, const char* fmt) {
    return ImGui::SliderFloat(label, v, lo, hi, fmt) ? 1 : 0;
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
        {"InputTextCursor", ImGuiCol_InputTextCursor},
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

// Scoped color override: pushes onto ImGui's style stack until obk_ig_pop_style_color.
void obk_ig_push_style_color(const char* name, float r, float g, float b, float a) {
    int idx = obk_col_index(name);
    if (idx < 0) return;
    ImGui::PushStyleColor((ImGuiCol)idx, ImVec4(r, g, b, a));
}

void obk_ig_pop_style_color(int n) { ImGui::PopStyleColor(n); }

int  obk_ig_color_edit4(const char* label, float* rgba) {
    return ImGui::ColorEdit4(label, rgba) ? 1 : 0;
}

int  obk_ig_color_swatch3(const char* label, float* rgba) {
    rgba[3] = 1.0f;
    return ImGui::ColorEdit3(label, rgba, ImGuiColorEditFlags_NoInputs | ImGuiColorEditFlags_NoLabel) ? 1 : 0;
}

// Window-draw-list primitives: free-form 2D shapes painted over the current window's
// content (e.g. the viewport's axis-orientation gizmo). Coordinates are screen-space
// pixels; colors are 0..1 RGBA packed to IM_COL32. Drawing happens within the current
// window's clip rect, so callers must invoke these inside the window's Begin/End.
static ImU32 obk_col32(float r, float g, float b, float a) {
    return IM_COL32((int)(r * 255), (int)(g * 255), (int)(b * 255), (int)(a * 255));
}
void obk_ig_draw_line(float x1, float y1, float x2, float y2,
                      float r, float g, float b, float a, float thickness) {
    ImGui::GetWindowDrawList()->AddLine(ImVec2(x1, y1), ImVec2(x2, y2),
                                        obk_col32(r, g, b, a), thickness);
}
void obk_ig_draw_triangle_filled(float x1, float y1, float x2, float y2, float x3, float y3,
                                 float r, float g, float b, float a) {
    ImGui::GetWindowDrawList()->AddTriangleFilled(ImVec2(x1, y1), ImVec2(x2, y2),
                                                  ImVec2(x3, y3), obk_col32(r, g, b, a));
}
// AddQuadFilled fills the quad as one convex polygon (a single anti-aliased outline), so a
// semi-transparent fill has no internal diagonal seam the way two triangles would.
void obk_ig_draw_quad_filled(float x0, float y0, float x1, float y1, float x2, float y2,
                             float x3, float y3, float r, float g, float b, float a) {
    ImGui::GetWindowDrawList()->AddQuadFilled(ImVec2(x0, y0), ImVec2(x1, y1), ImVec2(x2, y2),
                                              ImVec2(x3, y3), obk_col32(r, g, b, a));
}
void obk_ig_draw_text(float x, float y, float r, float g, float b, float a, const char* s) {
    ImGui::GetWindowDrawList()->AddText(ImVec2(x, y), obk_col32(r, g, b, a), s);
}

// The Script Console code editor draws itself directly onto the window draw list (it does not
// use ImGui's InputText), so it needs filled rects (selection/current-line/gutter), a scoped
// clip rect (the scrolling text viewport), fixed-width text, and the typed-character queue.
// The mono ImFont* is owned by app.cpp (added to the atlas at startup).
extern "C" ImFont* obk_head_mono_font(void);

void obk_ig_draw_rect_filled(float x0, float y0, float x1, float y1,
                             float r, float g, float b, float a) {
    ImGui::GetWindowDrawList()->AddRectFilled(ImVec2(x0, y0), ImVec2(x1, y1), obk_col32(r, g, b, a));
}
void obk_ig_push_clip_rect(float x0, float y0, float x1, float y1) {
    ImGui::GetWindowDrawList()->PushClipRect(ImVec2(x0, y0), ImVec2(x1, y1), true);
}
void obk_ig_pop_clip_rect(void) { ImGui::GetWindowDrawList()->PopClipRect(); }

// obk_ig_mono_size is the mono face's baked size scaled by the global UI text scale
// (style.FontScaleMain). Threading the scale through every mono metric keeps the Script Console
// editor tracking the user's UI-scale preference, with glyphs and cell geometry in step.
static float obk_ig_mono_size(ImFont* f) { return f->LegacySize * ImGui::GetStyle().FontScaleMain; }

// obk_ig_draw_text_mono draws s in the fixed-width face (falling back to the default font when
// the mono face failed to load), so editor glyphs land on integer columns.
void obk_ig_draw_text_mono(float x, float y, float r, float g, float b, float a, const char* s) {
    ImDrawList* dl = ImGui::GetWindowDrawList();
    ImU32 col = obk_col32(r, g, b, a);
    ImFont* f = obk_head_mono_font();
    if (f) dl->AddText(f, obk_ig_mono_size(f), ImVec2(x, y), col, s);
    else   dl->AddText(ImVec2(x, y), col, s);
}
// obk_ig_mono_char_width / obk_ig_mono_line_height give the editor its cell size: a mono
// glyph's advance and the line height. They fall back to the UI font metrics without a mono face.
// (ImGui's font API is baked-per-size here: the advance comes from the baked face.)
float obk_ig_mono_char_width(void) {
    ImFont* f = obk_head_mono_font();
    if (!f) return ImGui::CalcTextSize("M").x;
    ImFontBaked* baked = f->GetFontBaked(obk_ig_mono_size(f));
    return baked ? baked->GetCharAdvance((ImWchar)'M') : ImGui::CalcTextSize("M").x;
}
float obk_ig_mono_line_height(void) {
    ImFont* f = obk_head_mono_font();
    return f ? obk_ig_mono_size(f) : ImGui::GetTextLineHeight();
}

// obk_ig_utf8_encode appends the UTF-8 of code point c to buf at *off (bounded by buf_size),
// returning the bytes written (0 when it would overflow). Encoding here keeps the wrap free of
// imgui_internal.h's ImTextCharToUtf8.
static int obk_ig_utf8_encode(char* buf, int off, int buf_size, unsigned int c) {
    if (c < 0x80) {
        if (off + 1 >= buf_size) return 0;
        buf[off] = (char)c; return 1;
    }
    if (c < 0x800) {
        if (off + 2 >= buf_size) return 0;
        buf[off]   = (char)(0xC0 | (c >> 6));
        buf[off+1] = (char)(0x80 | (c & 0x3F));
        return 2;
    }
    if (c < 0x10000) {
        if (off + 3 >= buf_size) return 0;
        buf[off]   = (char)(0xE0 | (c >> 12));
        buf[off+1] = (char)(0x80 | ((c >> 6) & 0x3F));
        buf[off+2] = (char)(0x80 | (c & 0x3F));
        return 3;
    }
    if (off + 4 >= buf_size) return 0;
    buf[off]   = (char)(0xF0 | (c >> 18));
    buf[off+1] = (char)(0x80 | ((c >> 12) & 0x3F));
    buf[off+2] = (char)(0x80 | ((c >> 6) & 0x3F));
    buf[off+3] = (char)(0x80 | (c & 0x3F));
    return 4;
}
// obk_ig_input_chars writes the UTF-8 of every character typed this frame (ImGui's input
// queue) into buf and returns the byte count, NUL-terminating. The editor consumes this for
// text entry since it owns its own widget rather than using InputText.
int obk_ig_input_chars(char* buf, int buf_size) {
    if (buf_size <= 0) return 0;
    ImGuiIO& io = ImGui::GetIO();
    int off = 0;
    for (int i = 0; i < io.InputQueueCharacters.Size; i++) {
        int n = obk_ig_utf8_encode(buf, off, buf_size, (unsigned int)io.InputQueueCharacters[i]);
        if (n == 0) break;
        off += n;
    }
    buf[off] = 0;
    return off;
}
// Clipboard passthrough for the editor's cut/copy/paste. GetClipboardText returns ImGui's
// internal buffer (valid until the next clipboard call), so the Go side copies it immediately.
const char* obk_ig_get_clipboard(void) { return ImGui::GetClipboardText(); }
void obk_ig_set_clipboard(const char* s) { ImGui::SetClipboardText(s); }

// obk_ig_editor_keys returns a bitmask of the editor's navigation and shortcut keys pressed
// this frame. Navigation/edit keys (bits 0..9) use auto-repeat so a held arrow or Backspace
// repeats; the shortcut letters (bits 10..15) do not repeat — the Go side gates them on Ctrl.
// One call avoids a per-key cgo crossing each frame. Bit layout mirrors native.EditorKeys.
int obk_ig_editor_keys(void) {
    int m = 0;
    if (ImGui::IsKeyPressed(ImGuiKey_LeftArrow,  true)) m |= (1 << 0);
    if (ImGui::IsKeyPressed(ImGuiKey_RightArrow, true)) m |= (1 << 1);
    if (ImGui::IsKeyPressed(ImGuiKey_UpArrow,    true)) m |= (1 << 2);
    if (ImGui::IsKeyPressed(ImGuiKey_DownArrow,  true)) m |= (1 << 3);
    if (ImGui::IsKeyPressed(ImGuiKey_Backspace,  true)) m |= (1 << 4);
    if (ImGui::IsKeyPressed(ImGuiKey_Delete,     true)) m |= (1 << 5);
    if (ImGui::IsKeyPressed(ImGuiKey_Enter,      true) ||
        ImGui::IsKeyPressed(ImGuiKey_KeypadEnter, true)) m |= (1 << 6);
    if (ImGui::IsKeyPressed(ImGuiKey_Tab,        true)) m |= (1 << 7);
    if (ImGui::IsKeyPressed(ImGuiKey_Home,       true)) m |= (1 << 8);
    if (ImGui::IsKeyPressed(ImGuiKey_End,        true)) m |= (1 << 9);
    if (ImGui::IsKeyPressed(ImGuiKey_C, false)) m |= (1 << 10);
    if (ImGui::IsKeyPressed(ImGuiKey_V, false)) m |= (1 << 11);
    if (ImGui::IsKeyPressed(ImGuiKey_X, false)) m |= (1 << 12);
    if (ImGui::IsKeyPressed(ImGuiKey_A, false)) m |= (1 << 13);
    if (ImGui::IsKeyPressed(ImGuiKey_Z, false)) m |= (1 << 14);
    if (ImGui::IsKeyPressed(ImGuiKey_Y, false)) m |= (1 << 15);
    if (ImGui::IsKeyPressed(ImGuiKey_Slash, false)) m |= (1 << 16);
    if (ImGui::IsKeyPressed(ImGuiKey_F, false)) m |= (1 << 17);
    if (ImGui::IsKeyPressed(ImGuiKey_Space, false)) m |= (1 << 18);
    if (ImGui::IsKeyPressed(ImGuiKey_Escape, false)) m |= (1 << 19);
    return m;
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
void obk_ig_push_id_str(const char* id)                { ImGui::PushID(id); }
int  obk_ig_is_item_toggled_open(void)                 { return ImGui::IsItemToggledOpen() ? 1 : 0; }
void obk_ig_pop_id(void)                               { ImGui::PopID(); }
int  obk_ig_is_item_deactivated_after_edit(void)       { return ImGui::IsItemDeactivatedAfterEdit() ? 1 : 0; }

// any_mouse_down feeds the render-on-demand loop's "still animating" test (#1493): a held
// mouse button means a drag is in progress, so the loop keeps drawing instead of sleeping.
// (IsAnyItemActive is intentionally NOT exposed: the viewport's input-capturing
// InvisibleButton reads as active every frame, which would defeat the idle block.)
int  obk_ig_any_mouse_down(void)                       { return ImGui::IsAnyMouseDown() ? 1 : 0; }

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
// begin_overlay_window opens a borderless, transparent, auto-sized floating window for on-canvas
// controls (the Navigation Bar). Being its OWN window — not items inside the viewport window — its
// buttons receive clicks while the viewport's full-region InvisibleButton keeps the drag-to-orbit
// everywhere else, because ImGui routes the cursor to the topmost window under it (#1468). Pair with
// obk_ig_end regardless of the return value.
int obk_ig_begin_overlay_window(const char* name) {
    ImGuiWindowFlags f = ImGuiWindowFlags_NoDecoration | ImGuiWindowFlags_NoMove |
        ImGuiWindowFlags_NoSavedSettings | ImGuiWindowFlags_NoBackground |
        ImGuiWindowFlags_AlwaysAutoResize | ImGuiWindowFlags_NoDocking;
    return ImGui::Begin(name, nullptr, f) ? 1 : 0;
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
// obk_ig_key_ctrl reports the platform "command" modifier: Control on Windows/Linux, and
// Command (Super) on macOS — so a shortcut bound to Ctrl+S fires on Cmd+S on a Mac. The Windows
// key is NOT treated as Control (the Super→Ctrl mapping is gated on macOS behaviours).
int  obk_ig_key_ctrl(void) {
    ImGuiIO& io = ImGui::GetIO();
    bool cmdAsCtrl = io.ConfigMacOSXBehaviors && io.KeySuper;
    return (io.KeyCtrl || cmdAsCtrl) ? 1 : 0;
}
// escape_pressed fires once on the frame Esc is pressed (cancel the active tool).
int  obk_ig_escape_pressed(void)             { return ImGui::IsKeyPressed(ImGuiKey_Escape) ? 1 : 0; }
int  obk_ig_f1_pressed(void)                 { return ImGui::IsKeyPressed(ImGuiKey_F1) ? 1 : 0; }
// obk_ig_fkey_down reports whether F(n) is held (n in 2..4) — the hold-to-navigate keys
// F2 pan / F3 zoom / F4 orbit (#911).
int  obk_ig_fkey_down(int n) {
    ImGuiKey key;
    switch (n) {
        case 2: key = ImGuiKey_F2; break;
        case 3: key = ImGuiKey_F3; break;
        case 4: key = ImGuiKey_F4; break;
        default: return 0;
    }
    return ImGui::IsKeyDown(key) ? 1 : 0;
}
// undo/redo_pressed fire once on the frame Z / Y is pressed; the Go side gates them on
// Ctrl (and not WantTextInput) to form the global Ctrl+Z / Ctrl+Y shortcuts.
int  obk_ig_undo_pressed(void)               { return ImGui::IsKeyPressed(ImGuiKey_Z) ? 1 : 0; }
int  obk_ig_redo_pressed(void)               { return ImGui::IsKeyPressed(ImGuiKey_Y) ? 1 : 0; }
int  obk_ig_key_alt(void)                    { return ImGui::GetIO().KeyAlt ? 1 : 0; }
// pressed_keys writes the newline-separated names of every NAMED key pressed this frame (no
// key repeat, modifier keys excluded) into buf, so the Go side can form a chord per key and
// let the binding engine resolve rebindable shortcuts (M05-F17). Returns the count written.
int  obk_ig_pressed_keys(char* buf, int buf_size) {
    if (buf_size <= 0) return 0;
    buf[0] = 0;
    int count = 0, off = 0;
    for (ImGuiKey key = ImGuiKey_NamedKey_BEGIN; key < ImGuiKey_NamedKey_END; key = (ImGuiKey)(key + 1)) {
        if (!ImGui::IsKeyPressed(key, false)) continue; // false: ignore auto-repeat
        switch (key) { // modifier keys travel as the chord's modifier flags, not as keys
            case ImGuiKey_LeftCtrl:  case ImGuiKey_RightCtrl:
            case ImGuiKey_LeftShift: case ImGuiKey_RightShift:
            case ImGuiKey_LeftAlt:   case ImGuiKey_RightAlt:
            case ImGuiKey_LeftSuper: case ImGuiKey_RightSuper: continue;
            default: break;
        }
        const char* name = ImGui::GetKeyName(key);
        if (name == NULL || name[0] == 0) continue;
        const char* sep = (count > 0) ? "\n" : "";
        // snprintf is bounds-safe: it never writes past buf_size and always NUL-terminates.
        int n = snprintf(buf + off, (size_t)(buf_size - off), "%s%s", sep, name);
        if (n < 0 || n >= buf_size - off) { buf[off] = 0; break; } // would truncate: keep complete entries
        off += n;
        count++;
    }
    return count;
}
// want_text_input is true while a text/number field is being edited, so global shortcuts
// stand down and let the field keep the keystroke (incl. its own Ctrl+Z).
int  obk_ig_want_text_input(void)            { return ImGui::GetIO().WantTextInput ? 1 : 0; }
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
void obk_ig_dummy(float w, float h)           { ImGui::Dummy(ImVec2(w, h)); }

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
