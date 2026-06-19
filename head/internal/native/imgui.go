//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

/*
#include <stdlib.h>

int  obk_ig_begin_main_menu_bar(void);
void obk_ig_end_main_menu_bar(void);
int  obk_ig_begin_menu(const char* label);
void obk_ig_end_menu(void);
int  obk_ig_menu_item(const char* label);
int  obk_ig_menu_item_ex(const char* label, const char* shortcut, int enabled);
int  obk_ig_begin(const char* name);
int  obk_ig_begin_closable(const char* name, int* open);
void obk_ig_end(void);
void obk_ig_text(const char* s);
int  obk_ig_button(const char* label);
void obk_ig_same_line(void);
void obk_ig_separator(void);
void obk_ig_separator_text(const char* s);
void obk_ig_begin_group(void);
void obk_ig_end_group(void);
void obk_ig_separator_vertical(void);
int  obk_ig_mouse_double_clicked(int button);
int  obk_ig_begin_tab_bar(const char* id);
int  obk_ig_begin_tab_item_ex(const char* label, int setSelected);
int  obk_ig_begin_tab_item_closable(const char* label, int setSelected, int* open);
int  obk_ig_selectable(const char* label, int selected);
void obk_ig_end_tab_bar(void);
int  obk_ig_begin_tab_item(const char* label);
void obk_ig_end_tab_item(void);
int  obk_ig_collapsing_header(const char* l);
void obk_ig_set_next_item_open(int open, int first_use);
int  obk_ig_tree_node(const char* label);
int  obk_ig_tree_node_selectable(const char* label, int selected);
void obk_ig_tree_pop(void);
void obk_ig_bullet_text(const char* s);
void obk_ig_set_item_tooltip(const char* s);
void obk_ig_progress_bar(float fraction, float w, const char* overlay);
void obk_ig_main_viewport_size(float* w, float* h);
float obk_ig_hover_seconds(void);
int  obk_ig_begin_popup_context_item(const char* id);
void obk_ig_open_popup(const char* id);
int  obk_ig_begin_popup(const char* id);
void obk_ig_close_current_popup(void);
void obk_ig_end_popup(void);
void obk_ig_set_scroll_here_y(void);
void obk_ig_scroll_to_bottom(void);
int  obk_ig_input_float(const char* label, float* v);
int  obk_ig_input_double(const char* label, double* v);
int  obk_ig_input_int(const char* label, int* v);
int  obk_ig_input_text(const char* label, char* buf, int buf_size);
int  obk_ig_input_text_submit(const char* label, char* buf, int buf_size);
int  obk_ig_input_text_command(const char* label, char* buf, int buf_size, const char** hist, int nHist, int* histCursor, const char** comps, int nComps, int* compSel);
void obk_ig_set_keyboard_focus_here(void);
int  obk_ig_input_text_multiline(const char* label, char* buf, int buf_size, float w, float h);
int  obk_ig_begin_child(const char* id, float w, float h, int border);
void obk_ig_end_child(void);
int  obk_ig_checkbox(const char* label, int* v);
int  obk_ig_slider_float(const char* label, float* v, float lo, float hi);
void obk_ig_begin_disabled(int disabled);
void obk_ig_end_disabled(void);
int  obk_ig_want_capture_mouse(void);
int  obk_ig_invisible_button(const char* id, float w, float h);
int  obk_ig_is_item_active(void);
int  obk_ig_is_item_hovered(void);
int  obk_ig_mouse_down(int button);
int  obk_ig_is_item_clicked(int button);
void obk_ig_item_rect_min(float* x, float* y);
void obk_ig_mouse_pos(float* x, float* y);
int  obk_ig_key_shift(void);
int  obk_ig_key_ctrl(void);
int  obk_ig_key_alt(void);
int  obk_ig_escape_pressed(void);
int  obk_ig_f1_pressed(void);
int  obk_ig_fkey_down(int n);
int  obk_ig_undo_pressed(void);
int  obk_ig_redo_pressed(void);
int  obk_ig_pressed_keys(char* buf, int buf_size);
int  obk_ig_want_text_input(void);
float obk_ig_mouse_wheel(void);
float obk_ig_delta_time(void);
void obk_ig_mouse_delta(float* dx, float* dy);
void obk_ig_get_cursor_pos(float* x, float* y);
void obk_ig_set_cursor_pos(float x, float y);
int  obk_ig_begin_ribbon_band(const char* name, float height);
void obk_ig_get_cursor_screen_pos(float* x, float* y);
void obk_ig_set_cursor_screen_pos(float x, float y);
void obk_ig_item_rect_max(float* x, float* y);
float obk_ig_calc_text_width(const char* s);
float obk_ig_frame_height(void);
float obk_ig_text_line_height(void);
void obk_ig_style_metrics(float* fpx, float* fpy, float* isx, float* isy, float* wpx, float* wpy);
void obk_ig_set_next_window_pos(float x, float y);
void obk_ig_set_next_window_size(float w, float h);
void obk_ig_set_next_window_size_first_use(float w, float h);

// Table verbs (the Parameters dialog grid). begin_table opens a bordered, row-striped,
// vertically scrolling table of `columns` columns; pair with end_table only when it
// returned non-zero. setup_column/setup_scroll_freeze/headers_row define the header;
// next_row/next_column advance the cursor (next_column returns visibility). push_id_int/
// pop_id scope per-row widget ids so identical cell labels don't collide.
int  obk_ig_begin_table(const char* id, int columns, float outer_w, float outer_h);
void obk_ig_table_setup_column(const char* label);
void obk_ig_table_setup_scroll_freeze(int cols, int rows);
void obk_ig_table_headers_row(void);
void obk_ig_table_next_row(void);
int  obk_ig_table_next_column(void);
void obk_ig_end_table(void);
void obk_ig_set_next_item_width(float w);
void obk_ig_push_id_int(int id);
void obk_ig_push_id_str(const char* id);
int  obk_ig_is_item_toggled_open(void);
void obk_ig_pop_id(void);
int  obk_ig_is_item_deactivated_after_edit(void);

// Theming verbs (ADR-0021). set_style_color overwrites one persistent ImGuiStyle color,
// addressed by its ImGui name (e.g. "WindowBg", "Button") so Go never hardcodes the
// enum index (which shifts between ImGui versions); an unknown name is ignored.
// color_edit4 is the swatch+picker editing a 4-float RGBA in place (1 on change).
// begin_combo/end_combo bracket the theme selector dropdown.
void obk_ig_set_style_color(const char* name, float r, float g, float b, float a);
void obk_ig_push_style_color(const char* name, float r, float g, float b, float a);
void obk_ig_pop_style_color(int n);
int  obk_ig_color_edit4(const char* label, float* rgba);
void obk_ig_draw_line(float x1, float y1, float x2, float y2, float r, float g, float b, float a, float thickness);
void obk_ig_draw_triangle_filled(float x1, float y1, float x2, float y2, float x3, float y3, float r, float g, float b, float a);
void obk_ig_draw_quad_filled(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3, float r, float g, float b, float a);
void obk_ig_draw_text(float x, float y, float r, float g, float b, float a, const char* s);
void obk_ig_draw_rect_filled(float x0, float y0, float x1, float y1, float r, float g, float b, float a);
void obk_ig_push_clip_rect(float x0, float y0, float x1, float y1);
void obk_ig_pop_clip_rect(void);
void obk_ig_draw_text_mono(float x, float y, float r, float g, float b, float a, const char* s);
float obk_ig_mono_char_width(void);
float obk_ig_mono_line_height(void);
int  obk_ig_input_chars(char* buf, int buf_size);
const char* obk_ig_get_clipboard(void);
void obk_ig_set_clipboard(const char* s);
int  obk_ig_begin_combo(const char* label, const char* preview);
void obk_ig_end_combo(void);

unsigned int obk_ig_dockspace_over_main(void);
void obk_ig_dock_default_layout(unsigned int dockId, const char* model, const char* viewport, const char* status, unsigned int* outLeft, unsigned int* outBottom, unsigned int* outCenter);
unsigned int obk_ig_dock_split(unsigned int* nodeId, int dir, float ratio);
void obk_ig_set_next_window_dock(unsigned int nodeId);

// Synthetic-input injection for in-window UI tests (defined in app.cpp).
void obk_inject_mouse_pos(float x, float y);
void obk_inject_mouse_button(int b, int down);
void obk_inject_mouse_wheel(float w);
void obk_inject_key_shift(int down);
void obk_inject_fkey(int n, int down);
*/
import "C"

import (
	"strings"
	"unsafe"
)

// ImGui-widget wrappers in idiomatic Go. These are the verbs the chrome (head/ui)
// composes each frame; the LAYOUT logic stays in Go. cstr converts once per call —
// fine at UI scale, and keeps the call sites clean.

func cstr(s string) (*C.char, func()) {
	c := C.CString(s)
	return c, func() { C.free(unsafe.Pointer(c)) }
}

// BeginMainMenuBar / EndMainMenuBar bracket the top menu bar. BeginMainMenuBar
// reports whether the bar is visible (guard the menu calls with it).
func BeginMainMenuBar() bool { return C.obk_ig_begin_main_menu_bar() != 0 }
func EndMainMenuBar()        { C.obk_ig_end_main_menu_bar() }

// BeginMenu / EndMenu bracket a drop-down; BeginMenu reports whether it is open.
func BeginMenu(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_begin_menu(c) != 0
}
func EndMenu() { C.obk_ig_end_menu() }

// MenuItem renders a clickable item and reports whether it was activated this frame.
func MenuItem(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_menu_item(c) != 0
}

// MenuItemEx renders a menu item with a right-aligned shortcut hint and an enabled
// state — a disabled item greys out and cannot be clicked (Inventor's Edit ▸ Undo when
// there is nothing to undo). Reports whether it was activated this frame. shortcut "" or
// enabled false are both honored.
func MenuItemEx(label, shortcut string, enabled bool) bool {
	cl, freeL := cstr(label)
	defer freeL()
	cs, freeS := cstr(shortcut)
	defer freeS()
	return C.obk_ig_menu_item_ex(cl, cs, cBool(enabled)) != 0
}

// Begin / End bracket a window; Begin reports whether its content is visible.
func Begin(name string) bool {
	c, free := cstr(name)
	defer free()
	return C.obk_ig_begin(c) != 0
}
func End() { C.obk_ig_end() }

// BeginClosable is Begin with ImGui's title-bar close button: it reports whether
// the content is visible and whether the window is still open (false the frame the
// user clicks the X) — used by add-in dockable windows so closing reaches the
// owning add-in as a visibility event (M05-F03). Pair with End like Begin.
func BeginClosable(name string) (visible, open bool) {
	c, free := cstr(name)
	defer free()
	cOpen := C.int(1)
	v := C.obk_ig_begin_closable(c, &cOpen)
	return v != 0, cOpen != 0
}

// Text draws a line of unformatted text.
func Text(s string) {
	c, free := cstr(s)
	defer free()
	C.obk_ig_text(c)
}

// Button draws a button and reports whether it was clicked this frame.
func Button(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_button(c) != 0
}

func SameLine()  { C.obk_ig_same_line() }
func Separator() { C.obk_ig_separator() }

// BeginGroup / EndGroup bracket a layout group: everything between them is treated as
// one item for SameLine, so a ribbon panel (button row + title) can sit beside the next.
func BeginGroup() { C.obk_ig_begin_group() }
func EndGroup()   { C.obk_ig_end_group() }

// SeparatorVertical draws a vertical divider line, for separating horizontally-laid
// ribbon panels.
func SeparatorVertical() { C.obk_ig_separator_vertical() }

// InputFloat draws a float field; returns true (and writes *v) when edited.
func InputFloat(label string, v *float32) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_float(c, (*C.float)(v)) != 0
}

// InputDouble draws a float64 field; returns true (and writes *v) when edited. Used by the
// material editors, where values (e.g. resistivity ~1e15) exceed float32 precision.
func InputDouble(label string, v *float64) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_double(c, (*C.double)(v)) != 0
}

// InputInt draws an integer field; returns true (and writes *v) when edited.
func InputInt(label string, v *int32) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_int(c, (*C.int)(v)) != 0
}

// InputText draws a single-line text field bound to buf, a fixed byte buffer the
// caller owns across frames; ImGui edits it in place (NUL-terminated, up to
// len(buf)-1 bytes) and returns true on the frame the text changed. The head's
// file-path modal is the only string input on the chrome today.
func InputText(label string, buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_text(c, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf))) != 0
}

// InputTextSubmit is InputText that returns true only on the frame the user presses Enter
// (ImGuiInputTextFlags_EnterReturnsTrue) — the commit signal for the command-alias box.
func InputTextSubmit(label string, buf []byte) bool {
	if len(buf) == 0 {
		return false
	}
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_text_submit(c, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf))) != 0
}

// InputTextCommand is InputTextSubmit with Tab autocompletion and Up/Down navigation, driven
// by Dear ImGui's CallbackCompletion + CallbackHistory. history is the recall list and comps
// the live autocomplete candidates (oldest/first respectively); histCursor and compSel are
// caller-owned indices persisted across frames. When comps is non-empty Up/Down move compSel
// (the buffer is untouched) and Tab replaces the buffer with comps[compSel]; otherwise Up/Down
// recall history (histCursor==len(history) ⇒ the empty line). Returns true on Enter.
func InputTextCommand(label string, buf []byte, history []string, histCursor *int32, comps []string, compSel *int32) bool {
	if len(buf) == 0 {
		return false
	}
	c, free := cstr(label)
	defer free()
	hist, freeHist := cStringArray(history)
	defer freeHist()
	cmps, freeComps := cStringArray(comps)
	defer freeComps()
	hc, sc := C.int(*histCursor), C.int(*compSel)
	r := C.obk_ig_input_text_command(c, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)),
		hist, C.int(len(history)), &hc, cmps, C.int(len(comps)), &sc)
	*histCursor, *compSel = int32(hc), int32(sc)
	return r != 0
}

// cStringArray allocates a C array of C strings from ss and returns it with a free func
// that releases every string and the array. Returns (nil, no-op) for an empty slice.
func cStringArray(ss []string) (**C.char, func()) {
	if len(ss) == 0 {
		return nil, func() { /* no cleanup needed */ }
	}
	arr := C.malloc(C.size_t(len(ss)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	slice := unsafe.Slice((**C.char)(arr), len(ss))
	for i, s := range ss {
		slice[i] = C.CString(s)
	}
	return (**C.char)(arr), func() {
		for _, p := range slice {
			C.free(unsafe.Pointer(p))
		}
		C.free(arr)
	}
}

// SetKeyboardFocusHere focuses the next widget on the coming frame — used to put the caret
// in the command-alias box the moment it opens.
func SetKeyboardFocusHere() { C.obk_ig_set_keyboard_focus_here() }

// InputTextMultiline draws a multi-line text editor over buf (NUL-terminated, edited in
// place up to len(buf)-1 bytes) sized w×h logical pixels (0 ⇒ fill the available content
// region). Returns true on the frame buf changed. The Script Console source pane uses it.
func InputTextMultiline(label string, buf []byte, w, h float32) bool {
	if len(buf) == 0 {
		return false
	}
	c, free := cstr(label)
	defer free()
	return C.obk_ig_input_text_multiline(c, (*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)), C.float(w), C.float(h)) != 0
}

// BeginChild opens a scrollable child region id sized w×h (0 ⇒ fill the remaining space);
// border draws a frame. Always pair with EndChild even when it returns false — ImGui
// requires the matching call regardless (used for the console's output pane).
func BeginChild(id string, w, h float32, border bool) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_child(c, C.float(w), C.float(h), cBool(border)) != 0
}

// EndChild closes the region opened by BeginChild.
func EndChild() { C.obk_ig_end_child() }

// Checkbox draws a checkbox; returns true (and writes *v) when toggled.
func Checkbox(label string, v *bool) bool {
	c, free := cstr(label)
	defer free()
	cv := cBool(*v)
	changed := C.obk_ig_checkbox(c, &cv) != 0
	*v = cv != 0
	return changed
}

// SliderFloat draws a horizontal slider over [lo,hi], editing *v in place and returning true
// on the frame the value changed.
func SliderFloat(label string, v *float32, lo, hi float32) bool {
	c, free := cstr(label)
	defer free()
	cv := C.float(*v)
	changed := C.obk_ig_slider_float(c, &cv, C.float(lo), C.float(hi)) != 0
	*v = float32(cv)
	return changed
}

// SetStyleColor overwrites one Dear ImGui style color, addressed by its ImGui name
// (e.g. "WindowBg", "Button", "TabSelected"). ImGui's style is persistent global state,
// so the theme apply layer calls this once per token when the theme changes — not every
// frame. An unrecognized name is a no-op. See head/ui theme apply (ADR-0021).
func SetStyleColor(name string, c [4]float32) {
	cn, free := cstr(name)
	defer free()
	C.obk_ig_set_style_color(cn, C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]))
}

// PushStyleColor overrides one Dear ImGui style color (by ImGui name, e.g. "Button") for
// the widgets drawn until the matching PopStyleColor — so a single control can render in,
// say, the accent color without disturbing the global theme. Pair every push with a pop.
func PushStyleColor(name string, c [4]float32) {
	cn, free := cstr(name)
	defer free()
	C.obk_ig_push_style_color(cn, C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]))
}

// PopStyleColor undoes the last n PushStyleColor calls.
func PopStyleColor(n int) { C.obk_ig_pop_style_color(C.int(n)) }

// ColorEdit4 draws a color swatch that opens a picker, editing the RGBA in place and
// returning true on the frame the color changed — the per-token control in the
// Appearance editor.
func ColorEdit4(label string, c *[4]float32) bool {
	cl, free := cstr(label)
	defer free()
	return C.obk_ig_color_edit4(cl, (*C.float)(unsafe.Pointer(&c[0]))) != 0
}

// DrawLine draws a screen-space line in the current window's draw list (color is 0..1
// RGBA). Used for free-form overlays like the viewport's axis-orientation gizmo; call it
// inside the owning window's Begin/End so it clips to that window.
func DrawLine(x1, y1, x2, y2 float32, c [4]float32, thickness float32) {
	C.obk_ig_draw_line(C.float(x1), C.float(y1), C.float(x2), C.float(y2),
		C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]), C.float(thickness))
}

// DrawTriangleFilled fills a screen-space triangle in the current window's draw list
// (color is 0..1 RGBA) — e.g. the axis gizmo's arrowheads.
func DrawTriangleFilled(x1, y1, x2, y2, x3, y3 float32, c [4]float32) {
	C.obk_ig_draw_triangle_filled(C.float(x1), C.float(y1), C.float(x2), C.float(y2),
		C.float(x3), C.float(y3), C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]))
}

// DrawQuadFilled fills a screen-space quad (corners in order) as a single convex polygon,
// so a translucent fill shows no internal diagonal seam (unlike two triangles).
func DrawQuadFilled(x0, y0, x1, y1, x2, y2, x3, y3 float32, c [4]float32) {
	C.obk_ig_draw_quad_filled(C.float(x0), C.float(y0), C.float(x1), C.float(y1),
		C.float(x2), C.float(y2), C.float(x3), C.float(y3),
		C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]))
}

// DrawText draws a screen-space text label at (x,y) in the current window's draw list
// (color is 0..1 RGBA), using the default font — e.g. the axis gizmo's X/Y/Z letters.
func DrawText(x, y float32, s string, c [4]float32) {
	cs, free := cstr(s)
	defer free()
	C.obk_ig_draw_text(C.float(x), C.float(y), C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]), cs)
}

// DrawRectFilled fills a screen-space rectangle [x0,y0]-[x1,y1] in the current window's draw
// list (color is 0..1 RGBA) — the Script Console editor's selection, current-line and gutter
// backgrounds. Call inside the owning window's Begin/End so it clips to that window.
func DrawRectFilled(x0, y0, x1, y1 float32, c [4]float32) {
	C.obk_ig_draw_rect_filled(C.float(x0), C.float(y0), C.float(x1), C.float(y1),
		C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]))
}

// PushClipRect clips subsequent draw-list output to [x0,y0]-[x1,y1] (intersecting the current
// clip), bounding the editor's scrolling text region; pair every call with PopClipRect.
func PushClipRect(x0, y0, x1, y1 float32) {
	C.obk_ig_push_clip_rect(C.float(x0), C.float(y0), C.float(x1), C.float(y1))
}

// PopClipRect undoes the last PushClipRect.
func PopClipRect() { C.obk_ig_pop_clip_rect() }

// DrawTextMono draws text in the fixed-width editor face at (x,y) (color 0..1 RGBA); it falls
// back to the default font if the mono face is unavailable.
func DrawTextMono(x, y float32, s string, c [4]float32) {
	cs, free := cstr(s)
	defer free()
	C.obk_ig_draw_text_mono(C.float(x), C.float(y), C.float(c[0]), C.float(c[1]), C.float(c[2]), C.float(c[3]), cs)
}

// MonoCharWidth and MonoLineHeight are the editor's cell metrics: the advance of one
// fixed-width glyph and the line height, both in logical pixels.
func MonoCharWidth() float32  { return float32(C.obk_ig_mono_char_width()) }
func MonoLineHeight() float32 { return float32(C.obk_ig_mono_line_height()) }

// InputChars returns the characters typed this frame (ImGui's input queue) as a string. The
// code editor consumes these for text entry since it does not use ImGui's InputText widget.
func InputChars() string {
	var buf [256]byte
	n := C.obk_ig_input_chars((*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	if n == 0 {
		return ""
	}
	return string(buf[:n])
}

// ClipboardText returns the system clipboard's text; SetClipboardText writes it — the editor's
// cut/copy/paste payload.
func ClipboardText() string { return C.GoString(C.obk_ig_get_clipboard()) }
func SetClipboardText(s string) {
	cs, free := cstr(s)
	defer free()
	C.obk_ig_set_clipboard(cs)
}

// BeginCombo / EndCombo bracket a dropdown showing preview as the closed value; when
// BeginCombo returns true the popup is open, so draw the Selectable options and pair it
// with EndCombo. Used by the theme selector.
func BeginCombo(label, preview string) bool {
	cl, free := cstr(label)
	defer free()
	cp, free2 := cstr(preview)
	defer free2()
	return C.obk_ig_begin_combo(cl, cp) != 0
}

func EndCombo() { C.obk_ig_end_combo() }

// BeginTable opens a bordered, row-striped, scrolling table of columns columns sized to
// (w, h) (0 ⇒ fit content / fill avail). Draw its rows only when it returns true, and
// pair every true with EndTable. Define headers with TableSetupColumn + TableHeadersRow.
func BeginTable(id string, columns int, w, h float32) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_table(c, C.int(columns), C.float(w), C.float(h)) != 0
}
func EndTable() { C.obk_ig_end_table() }

// TableSetupColumn declares the next header column; TableSetupScrollFreeze keeps the
// first cols columns / rows rows pinned while scrolling; TableHeadersRow emits the
// header row from the declared columns.
func TableSetupColumn(label string) {
	c, free := cstr(label)
	defer free()
	C.obk_ig_table_setup_column(c)
}

func TableSetupScrollFreeze(cols, rows int) {
	C.obk_ig_table_setup_scroll_freeze(C.int(cols), C.int(rows))
}
func TableHeadersRow() { C.obk_ig_table_headers_row() }

// TableNextRow starts a new row; TableNextColumn advances to the next cell and reports
// whether it is visible (so a Go loop can skip clipped cells).
func TableNextRow()         { C.obk_ig_table_next_row() }
func TableNextColumn() bool { return C.obk_ig_table_next_column() != 0 }

// SetNextItemWidth sets the width of the next widget (-1 ⇒ fill the cell/column).
func SetNextItemWidth(w float32) { C.obk_ig_set_next_item_width(C.float(w)) }

// PushIDInt / PopID scope an integer id (a parameter's id) so identical cell-widget
// labels across rows do not collide. Pair every PushIDInt with a PopID.
func PushIDInt(id int) { C.obk_ig_push_id_int(C.int(id)) }

// PushID pushes a string id onto ImGui's id stack — for rows whose ids come from
// declared data (add-in pane nodes) rather than a frame-local counter.
func PushID(id string) {
	c, free := cstr(id)
	defer free()
	C.obk_ig_push_id_str(c)
}

// IsItemToggledOpen reports whether the last tree node was opened or closed by this
// frame's interaction — how the browser detects an expand/collapse gesture to report
// to the owning add-in (M05-F03).
func IsItemToggledOpen() bool { return C.obk_ig_is_item_toggled_open() != 0 }
func PopID()                  { C.obk_ig_pop_id() }

// IsItemDeactivatedAfterEdit reports whether the most recent item was edited and then
// committed this frame (Enter or focus loss) — so a text cell commits once the user is
// done, not on every keystroke.
func IsItemDeactivatedAfterEdit() bool { return C.obk_ig_is_item_deactivated_after_edit() != 0 }

// SeparatorText draws a labeled horizontal separator (used as panel titles).
func SeparatorText(s string) {
	c, free := cstr(s)
	defer free()
	C.obk_ig_separator_text(c)
}

// BeginTabBar / EndTabBar bracket a tab bar; BeginTabBar reports whether it is visible
// (only call EndTabBar when it returned true). The id also labels the bar internally.
func BeginTabBar(id string) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_tab_bar(c) != 0
}
func EndTabBar() { C.obk_ig_end_tab_bar() }

// BeginTabItem / EndTabItem bracket one tab; BeginTabItem reports whether that tab is
// selected this frame (only draw its contents, and call EndTabItem, when true).
func BeginTabItem(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_begin_tab_item(c) != 0
}
func EndTabItem() { C.obk_ig_end_tab_item() }

// BeginTabItemSelected is BeginTabItem that optionally forces this tab selected this
// frame (for the contextual Sketch tab auto-switch). Pass setSelected only on the
// transition frame so the user can still switch tabs by hand afterwards.
func BeginTabItemSelected(label string, setSelected bool) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_begin_tab_item_ex(c, cBool(setSelected)) != 0
}

// BeginTabItemClosable is BeginTabItemSelected with ImGui's close button. It reports
// whether the tab content is visible and whether the tab should remain open.
func BeginTabItemClosable(label string, setSelected bool) (bool, bool) {
	c, free := cstr(label)
	defer free()
	open := C.int(1)
	visible := C.obk_ig_begin_tab_item_closable(c, cBool(setSelected), &open) != 0
	return visible, open != 0
}

// Selectable draws a clickable, highlightable row (browser entries); returns true when
// clicked this frame.
func Selectable(label string, selected bool) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_selectable(c, cBool(selected)) != 0
}

// CollapsingHeader draws a collapsible section header; reports whether it is open.
func CollapsingHeader(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_collapsing_header(c) != 0
}

// TreeNode / TreePop bracket a tree node; TreeNode reports whether it is expanded
// (only call TreePop when it returned true).
func SetNextItemOpen(open, firstUse bool) { C.obk_ig_set_next_item_open(cBool(open), cBool(firstUse)) }

func TreeNode(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_tree_node(c) != 0
}

// TreeNodeSelectable draws a tree node that is BOTH expandable and selectable (a browser
// feature row that nests its consumed sketch): the disclosure arrow toggles it open, while
// a click on the label selects rather than expands. selected draws it highlighted. Returns
// whether it is expanded (call TreePop only when true). Pair with IsItemClicked for the
// click and IsItemHovered/IsMouseDoubleClicked for edit-on-double-click.
func TreeNodeSelectable(label string, selected bool) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_tree_node_selectable(c, cBool(selected)) != 0
}
func TreePop() { C.obk_ig_tree_pop() }

// BulletText draws a leaf row with a bullet (browser leaves).
func BulletText(s string) {
	c, free := cstr(s)
	defer free()
	C.obk_ig_bullet_text(c)
}

// SetItemTooltip shows a hover tooltip for the most recently drawn item (the ribbon
// button), matching Inventor's on-hover command tooltips. No-op if the text is empty.
// ProgressBar draws a determinate progress bar of the given pixel width with an
// optional overlay text (M05-F09 status-bar progress).
func ProgressBar(fraction, width float32, overlay string) {
	c, free := cstr(overlay)
	defer free()
	C.obk_ig_progress_bar(C.float(fraction), C.float(width), c)
}

// MainViewportSize reports the main viewport's pixel size — used to anchor the
// balloon toasts and the prompt modal (M05-F09).
func MainViewportSize() (w, h float32) {
	var cw, ch C.float
	C.obk_ig_main_viewport_size(&cw, &ch)
	return float32(cw), float32(ch)
}

// HoverSeconds reports how long the last item has been hovered — drives the
// progressive tooltip's expanded text (M05-F09).
func HoverSeconds() float32 { return float32(C.obk_ig_hover_seconds()) }

func SetItemTooltip(s string) {
	if s == "" {
		return
	}
	c, free := cstr(s)
	defer free()
	C.obk_ig_set_item_tooltip(c)
}

// BeginPopupContextItem opens (on a right-click of the item just drawn) and begins a
// context-menu popup identified by id, returning true while it is open — fill it with
// MenuItem rows, then call EndPopup only when this returned true.
func BeginPopupContextItem(id string) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_popup_context_item(c) != 0
}

// OpenPopup marks the popup with id to open on this frame (pair with BeginPopup). Use it to
// open a popup from arbitrary logic (e.g. a right-click on a custom-drawn region) rather
// than tying it to the last widget like BeginPopupContextItem.
func OpenPopup(id string) {
	c, free := cstr(id)
	defer free()
	C.obk_ig_open_popup(c)
}

// BeginPopup returns true while the popup with id (opened via OpenPopup) is showing; fill
// it with MenuItem rows and call EndPopup only when this returned true.
func BeginPopup(id string) bool {
	c, free := cstr(id)
	defer free()
	return C.obk_ig_begin_popup(c) != 0
}

// EndPopup closes a popup begun by BeginPopupContextItem/BeginPopup (call only when it
// returned true).
func EndPopup() { C.obk_ig_end_popup() }

// CloseCurrentPopup closes the popup being drawn (a marking-menu pick).
func CloseCurrentPopup() { C.obk_ig_close_current_popup() }

// SetScrollHereY scrolls the current window to center the most recently drawn item — used
// to reveal the browser node that just synced to the active selection.
func SetScrollHereY() { C.obk_ig_set_scroll_here_y() }

// ScrollToBottom scrolls the current window so the most recently drawn item sits at the
// bottom edge — the shell behaviour where new lines appear at the bottom and roll older
// text up (the Command Window's scrollback).
func ScrollToBottom() { C.obk_ig_scroll_to_bottom() }

// BeginDisabled / EndDisabled gray out and disable the widgets between them.
func BeginDisabled(disabled bool) {
	v := 0
	if disabled {
		v = 1
	}
	C.obk_ig_begin_disabled(C.int(v))
}
func EndDisabled() { C.obk_ig_end_disabled() }

// WantCaptureMouse reports whether ImGui consumed the pointer this frame, so the loop
// knows when a click belongs to the 3D viewport instead of the chrome.
func WantCaptureMouse() bool { return C.obk_ig_want_capture_mouse() != 0 }

// MouseButton indexes the pointer buttons for MouseDown (ImGui's convention).
const (
	MouseLeft   = 0
	MouseRight  = 1
	MouseMiddle = 2
)

// InvisibleButton reserves a w×h input-capturing region (the viewport drag surface) at
// the cursor; it accepts any mouse button so a middle-/right-drag keeps the item active.
func InvisibleButton(id string, w, h float32) {
	c, free := cstr(id)
	defer free()
	C.obk_ig_invisible_button(c, C.float(w), C.float(h))
}

// IsItemActive / IsItemHovered report the state of the most recent item (the viewport
// drag surface): Active = held/being dragged (even if the cursor left it), Hovered = the
// pointer is over it this frame.
func IsItemActive() bool  { return C.obk_ig_is_item_active() != 0 }
func IsItemHovered() bool { return C.obk_ig_is_item_hovered() != 0 }

// MouseDown reports whether the given mouse button is currently pressed.
func MouseDown(button int) bool { return C.obk_ig_mouse_down(C.int(button)) != 0 }

// IsItemClicked reports whether the last item was clicked with the given button this
// frame (used to place sketch geometry by clicking in the viewport).
func IsItemClicked(button int) bool { return C.obk_ig_is_item_clicked(C.int(button)) != 0 }

// IsMouseDoubleClicked reports a double-click of the given button this frame (used to
// re-open a dimension for editing).
func IsMouseDoubleClicked(button int) bool { return C.obk_ig_mouse_double_clicked(C.int(button)) != 0 }

// ItemRectMin returns the screen-space top-left of the last item (the viewport image),
// so a mouse position can be converted to a pixel local to the viewport.
func ItemRectMin() (float32, float32) {
	var x, y C.float
	C.obk_ig_item_rect_min(&x, &y)
	return float32(x), float32(y)
}

// MousePos returns the pointer position in screen space.
func MousePos() (float32, float32) {
	var x, y C.float
	C.obk_ig_mouse_pos(&x, &y)
	return float32(x), float32(y)
}

// KeyShift reports whether a Shift key is held (orbit modifier).
func KeyShift() bool { return C.obk_ig_key_shift() != 0 }

// KeyCtrl reports whether a Ctrl key is held (multi-select modifier).
func KeyCtrl() bool { return C.obk_ig_key_ctrl() != 0 }

// KeyAlt reports whether an Alt key is held (a shortcut-chord modifier).
func KeyAlt() bool { return C.obk_ig_key_alt() != 0 }

// PressedKeys returns the names of the named keys pressed this frame (no auto-repeat,
// modifier keys excluded) — e.g. ["E"] or ["F1"]. The names match the canonical KeyChord
// key tokens, so the head forwards each as a chord to the binding engine (M05-F17).
func PressedKeys() []string {
	var buf [256]byte
	if C.obk_ig_pressed_keys((*C.char)(unsafe.Pointer(&buf[0])), C.int(len(buf))) == 0 {
		return nil
	}
	return strings.Split(C.GoString((*C.char)(unsafe.Pointer(&buf[0]))), "\n")
}

// EscapePressed reports whether Esc was pressed this frame (cancel the active tool).
// F1Pressed fires once on the frame F1 is pressed — the host help shortcut (M05-F14).
func F1Pressed() bool { return C.obk_ig_f1_pressed() != 0 }

// FKeyDown reports whether F(n) is currently held (n in 2..4) — the hold-to-navigate keys
// F2 pan / F3 zoom / F4 orbit (#911).
func FKeyDown(n int) bool { return C.obk_ig_fkey_down(C.int(n)) != 0 }

func EscapePressed() bool { return C.obk_ig_escape_pressed() != 0 }

// UndoPressed / RedoPressed report whether the Z / Y key was pressed this frame (the
// caller pairs them with KeyCtrl and WantTextInput to form the Ctrl+Z / Ctrl+Y / Ctrl+
// Shift+Z bindings only when no text field is capturing input).
func UndoPressed() bool { return C.obk_ig_undo_pressed() != 0 }
func RedoPressed() bool { return C.obk_ig_redo_pressed() != 0 }

// WantTextInput reports whether an ImGui widget currently has keyboard text focus (a
// text/number field being edited). Global shortcuts must be suppressed when it is true so
// the field's own editing — including its built-in Ctrl+Z — keeps the keystroke.
func WantTextInput() bool { return C.obk_ig_want_text_input() != 0 }

// MouseWheel returns this frame's vertical scroll amount (zoom).
func MouseWheel() float32 { return float32(C.obk_ig_mouse_wheel()) }

// DeltaTime returns the seconds elapsed since the previous frame (for animations).
func DeltaTime() float32 { return float32(C.obk_ig_delta_time()) }

// MouseDelta returns this frame's pointer movement in pixels (x right, y down).
func MouseDelta() (float32, float32) {
	var dx, dy C.float
	C.obk_ig_mouse_delta(&dx, &dy)
	return float32(dx), float32(dy)
}

// GetCursorPos / SetCursorPos read and restore the window-local layout cursor, so the
// viewport image can be drawn back over the invisible drag button.
func GetCursorPos() (float32, float32) {
	var x, y C.float
	C.obk_ig_get_cursor_pos(&x, &y)
	return float32(x), float32(y)
}
func SetCursorPos(x, y float32) { C.obk_ig_set_cursor_pos(C.float(x), C.float(y)) }

// SetNextWindowPos / SetNextWindowSize force the next Begin's window geometry — used by
// in-window tests to put the viewport panel at a known rect so injected input lands on it.
func SetNextWindowPos(x, y float32)  { C.obk_ig_set_next_window_pos(C.float(x), C.float(y)) }
func SetNextWindowSize(w, h float32) { C.obk_ig_set_next_window_size(C.float(w), C.float(h)) }

// SetNextWindowSizeOnce sets the next window's size only the first time it is shown
// (ImGuiCond_FirstUseEver), so it gives a sensible default the user can still resize.
func SetNextWindowSizeOnce(w, h float32) {
	C.obk_ig_set_next_window_size_first_use(C.float(w), C.float(h))
}

// DockSpaceOverMain hosts a dockspace over the main viewport's work area each frame and
// returns its stable id. The work area excludes the menu bar and the ribbon band (both
// claim their slice via the viewport side-bar mechanism), so the docked panels lay out
// below the fixed chrome. Call it once per frame before the panels.
func DockSpaceOverMain() uint32 { return uint32(C.obk_ig_dockspace_over_main()) }

// DockSideNodes are the dock-node ids of the default arrangement, so late-created
// windows (add-in dockable windows, M05-F03) can be docked beside the built-ins.
type DockSideNodes struct{ Left, Bottom, Center uint32 }

// DockDefaultLayout arranges the named panels once: model left, status bottom, viewport
// filling the center, returning the side node ids. Call it a single time after the
// first DockSpaceOverMain (the names must match the panels' Begin() titles). The
// ribbon is not docked — it is a fixed band drawn with BeginRibbonBand.
func DockDefaultLayout(dockID uint32, model, viewport, status string) DockSideNodes {
	cm, fm := cstr(model)
	defer fm()
	cv, fv := cstr(viewport)
	defer fv()
	cs, fs := cstr(status)
	defer fs()
	var left, bottom, center C.uint
	C.obk_ig_dock_default_layout(C.uint(dockID), cm, cv, cs, &left, &bottom, &center)
	return DockSideNodes{Left: uint32(left), Bottom: uint32(bottom), Center: uint32(center)}
}

// DockSplit carves a new node off *node (ImGuiDir: 0=left, 1=right, 2=up, 3=down)
// at the given ratio, returning the new node's id; *node becomes the remainder.
// Used to create a band on demand for an add-in window.
func DockSplit(node *uint32, dir int, ratio float32) uint32 {
	cNode := C.uint(*node)
	fresh := C.obk_ig_dock_split(&cNode, C.int(dir), C.float(ratio))
	*node = uint32(cNode)
	return uint32(fresh)
}

// SetNextWindowDock docks the next Begin'd window into the given node on its first
// appearance; the user's later re-docking wins.
func SetNextWindowDock(nodeID uint32) { C.obk_ig_set_next_window_dock(C.uint(nodeID)) }

// BeginRibbonBand pins a full-width band of the given height across the top of the main
// viewport, under the menu bar. The band is fixed chrome: not movable, resizable, or
// dockable, and the dockspace lays out beneath it. Reports whether the content is
// visible; pair with End regardless.
func BeginRibbonBand(name string, height float32) bool {
	c, free := cstr(name)
	defer free()
	return C.obk_ig_begin_ribbon_band(c, C.float(height)) != 0
}

// GetCursorScreenPos / SetCursorScreenPos read and write the layout cursor in screen
// space — for pinning the ribbon's panel-name strip at a fixed band Y.
func GetCursorScreenPos() (float32, float32) {
	var x, y C.float
	C.obk_ig_get_cursor_screen_pos(&x, &y)
	return float32(x), float32(y)
}
func SetCursorScreenPos(x, y float32) { C.obk_ig_set_cursor_screen_pos(C.float(x), C.float(y)) }

// ItemRectMax returns the screen-space bottom-right of the last item (pairs with
// ItemRectMin to measure a just-drawn group, e.g. a ribbon panel's button block).
func ItemRectMax() (float32, float32) {
	var x, y C.float
	C.obk_ig_item_rect_max(&x, &y)
	return float32(x), float32(y)
}

// CalcTextWidth returns the rendered width of s in the current font (for centering
// ribbon captions and panel names).
func CalcTextWidth(s string) float32 {
	c, free := cstr(s)
	defer free()
	return float32(C.obk_ig_calc_text_width(c))
}

// FrameHeight is the height of a framed widget row (font size + frame padding) — the
// tab-strip height the ribbon band budget builds on.
func FrameHeight() float32 { return float32(C.obk_ig_frame_height()) }

// TextLineHeight is the height of one line of text in the current font.
func TextLineHeight() float32 { return float32(C.obk_ig_text_line_height()) }

// StyleMetrics are the live ImGui style paddings/spacings, so Go layout math (the
// ribbon band height, caption centering) tracks the style instead of hardcoding pixels.
type StyleMetrics struct {
	FramePadX, FramePadY       float32
	ItemSpacingX, ItemSpacingY float32
	WindowPadX, WindowPadY     float32
}

// Metrics reads the current style's paddings and spacings.
func Metrics() StyleMetrics {
	var fpx, fpy, isx, isy, wpx, wpy C.float
	C.obk_ig_style_metrics(&fpx, &fpy, &isx, &isy, &wpx, &wpy)
	return StyleMetrics{
		FramePadX: float32(fpx), FramePadY: float32(fpy),
		ItemSpacingX: float32(isx), ItemSpacingY: float32(isy),
		WindowPadX: float32(wpx), WindowPadY: float32(wpy),
	}
}

// Inject* feed synthetic pointer/keyboard input into the next frame's ImGui IO, winning
// over the real cursor. They exist so an in-window test can drive the actual chrome
// (InvisibleButton → ApplyNavigation → camera) through the live frame loop. Production
// code never calls them.
func InjectMousePos(x, y float32) { C.obk_inject_mouse_pos(C.float(x), C.float(y)) }

func InjectMouseButton(button int, down bool) { C.obk_inject_mouse_button(C.int(button), cBool(down)) }
func InjectMouseWheel(w float32)              { C.obk_inject_mouse_wheel(C.float(w)) }
func InjectKeyShift(down bool)                { C.obk_inject_key_shift(cBool(down)) }

// InjectFKey holds (down) or releases the hold-to-navigate function key F(n), n in 2..4 (#911).
func InjectFKey(n int, down bool) { C.obk_inject_fkey(C.int(n), cBool(down)) }

// cBool converts a Go bool to the 0/1 C.int the wrappers expect.
func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
