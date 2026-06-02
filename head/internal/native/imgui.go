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
int  obk_ig_begin(const char* name);
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
int  obk_ig_selectable(const char* label, int selected);
void obk_ig_end_tab_bar(void);
int  obk_ig_begin_tab_item(const char* label);
void obk_ig_end_tab_item(void);
int  obk_ig_collapsing_header(const char* l);
int  obk_ig_tree_node(const char* label);
void obk_ig_tree_pop(void);
void obk_ig_bullet_text(const char* s);
void obk_ig_set_item_tooltip(const char* s);
int  obk_ig_input_float(const char* label, float* v);
int  obk_ig_input_int(const char* label, int* v);
int  obk_ig_input_text(const char* label, char* buf, int buf_size);
int  obk_ig_checkbox(const char* label, int* v);
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
int  obk_ig_escape_pressed(void);
float obk_ig_mouse_wheel(void);
float obk_ig_delta_time(void);
void obk_ig_mouse_delta(float* dx, float* dy);
void obk_ig_get_cursor_pos(float* x, float* y);
void obk_ig_set_cursor_pos(float x, float y);
void obk_ig_set_next_window_pos(float x, float y);
void obk_ig_set_next_window_size(float w, float h);

unsigned int obk_ig_dockspace_over_main(void);
void obk_ig_dock_default_layout(unsigned int dockId, const char* ribbon, const char* model, const char* viewport, const char* status);

// Synthetic-input injection for in-window UI tests (defined in app.cpp).
void obk_inject_mouse_pos(float x, float y);
void obk_inject_mouse_button(int b, int down);
void obk_inject_mouse_wheel(float w);
void obk_inject_key_shift(int down);
*/
import "C"

import "unsafe"

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

// Begin / End bracket a window; Begin reports whether its content is visible.
func Begin(name string) bool {
	c, free := cstr(name)
	defer free()
	return C.obk_ig_begin(c) != 0
}
func End() { C.obk_ig_end() }

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

// Checkbox draws a checkbox; returns true (and writes *v) when toggled.
func Checkbox(label string, v *bool) bool {
	c, free := cstr(label)
	defer free()
	cv := cBool(*v)
	changed := C.obk_ig_checkbox(c, &cv) != 0
	*v = cv != 0
	return changed
}

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
func TreeNode(label string) bool {
	c, free := cstr(label)
	defer free()
	return C.obk_ig_tree_node(c) != 0
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
func SetItemTooltip(s string) {
	if s == "" {
		return
	}
	c, free := cstr(s)
	defer free()
	C.obk_ig_set_item_tooltip(c)
}

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

// EscapePressed reports whether Esc was pressed this frame (cancel the active tool).
func EscapePressed() bool { return C.obk_ig_escape_pressed() != 0 }

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

// DockSpaceOverMain hosts a full-window dockspace each frame and returns its stable id.
// Call it once per frame before the panels; the panels then dock into it.
func DockSpaceOverMain() uint32 { return uint32(C.obk_ig_dockspace_over_main()) }

// DockDefaultLayout arranges the named panels once: ribbon top, model left, status
// bottom, viewport filling the center. Call it a single time after the first
// DockSpaceOverMain (the names must match the panels' Begin() titles).
func DockDefaultLayout(dockID uint32, ribbon, model, viewport, status string) {
	cr, fr := cstr(ribbon)
	defer fr()
	cm, fm := cstr(model)
	defer fm()
	cv, fv := cstr(viewport)
	defer fv()
	cs, fs := cstr(status)
	defer fs()
	C.obk_ig_dock_default_layout(C.uint(dockID), cr, cm, cv, cs)
}

// Inject* feed synthetic pointer/keyboard input into the next frame's ImGui IO, winning
// over the real cursor. They exist so an in-window test can drive the actual chrome
// (InvisibleButton → ApplyNavigation → camera) through the live frame loop. Production
// code never calls them.
func InjectMousePos(x, y float32) { C.obk_inject_mouse_pos(C.float(x), C.float(y)) }

func InjectMouseButton(button int, down bool) { C.obk_inject_mouse_button(C.int(button), cBool(down)) }
func InjectMouseWheel(w float32)              { C.obk_inject_mouse_wheel(C.float(w)) }
func InjectKeyShift(down bool)                { C.obk_inject_key_shift(cBool(down)) }

// cBool converts a Go bool to the 0/1 C.int the wrappers expect.
func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
