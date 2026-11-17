// SPDX-License-Identifier: GPL-2.0-only
use imgui::{Condition, ImColor32, ImString, SelectableFlags, Ui};
unsafe extern "C" {
    fn oblikovati_gui_window_is_moving() -> bool;
    fn oblikovati_gui_drag_current_window() -> bool;
    fn oblikovati_gui_workspace_dockspace(width: f32, height: f32) -> u32;
    fn oblikovati_gui_outer_dockspace(width: f32, height: f32) -> u32;
    fn oblikovati_gui_outer_bottom_span(id: u32, left: *mut f32, right: *mut f32) -> bool;
    fn oblikovati_gui_dockspace_center(
        id: u32,
        left: *mut f32,
        top: *mut f32,
        right: *mut f32,
        bottom: *mut f32,
    ) -> bool;
    fn oblikovati_gui_icon_button(
        label: *const u8,
        texture: u32,
        width: f32,
        height: f32,
        icon_width: f32,
        icon_height: f32,
        large: bool,
        show_caption: bool,
    ) -> bool;
    fn oblikovati_gui_fit_dialog(title: *const u8, x: f32, y: f32, width: f32, height: f32);
    fn oblikovati_gui_begin_scroll_region(identity: *const u8, width: f32, height: f32) -> bool;
    fn oblikovati_gui_dummy(width: f32, height: f32);
    fn oblikovati_gui_window_position(x: f32, y: f32, condition: i32);
    fn oblikovati_gui_window_size(width: f32, height: f32, condition: i32);
    fn oblikovati_gui_cursor_position(x: f32, y: f32);
    fn oblikovati_gui_button(label: *const u8, width: f32, height: f32, selected: bool) -> bool;
    fn oblikovati_gui_selectable(label: *const u8, selected: bool, flags: i32) -> bool;
    fn oblikovati_gui_literal_text(
        text: *const u8,
        length: usize,
        x: f32,
        y: f32,
        color: u32,
        background: bool,
    );
}

/// Transfer an active tab drag to its owning window.
/// Example: call `drag_current_window(ui)` while the Model tab is held.
pub(crate) fn drag_current_window(_: &Ui) -> bool {
    unsafe { oblikovati_gui_drag_current_window() }
}
/// Let workspace targets receive pointer input while an ImGui window is dragged.
/// Example: `moving_window(ui)` when drawing a fixed document-tab strip.
pub(crate) fn moving_window(_: &Ui) -> bool {
    unsafe { oblikovati_gui_window_is_moving() }
}
/// Draw a dockspace in the current host and return its central bounds when available.
/// Example: `workspace_dockspace(ui, [800., 600.])` in a transparent host window.
pub(crate) fn workspace_dockspace(_: &Ui, size: [f32; 2]) -> Option<[f32; 4]> {
    let id = unsafe { oblikovati_gui_workspace_dockspace(size[0], size[1]) };
    dockspace_center(id)
}
/// Draw the outer workspace, including the default Model dock.
/// Example: `outer_dockspace(ui, [800., 600.])` in its transparent host.
pub(crate) fn outer_dockspace(_: &Ui, size: [f32; 2]) -> Option<([f32; 4], Option<[f32; 2]>)> {
    let id = unsafe { oblikovati_gui_outer_dockspace(size[0], size[1]) };
    let mut span = [0.; 2];
    let [left, right] = &mut span;
    let has_bottom = unsafe { oblikovati_gui_outer_bottom_span(id, left, right) };
    dockspace_center(id).map(|center| (center, has_bottom.then_some(span)))
}
fn dockspace_center(id: u32) -> Option<[f32; 4]> {
    let mut bounds = [0.; 4];
    let [left, top, right, bottom] = &mut bounds;
    let found = unsafe { oblikovati_gui_dockspace_center(id, left, top, right, bottom) };
    found.then_some(bounds)
}
/// Prepare a window through scalar ABI calls; example: `window_geometry(ui, [0.,0.], [640.,480.], Condition::Always)`.
pub(crate) fn window_geometry(_: &Ui, position: [f32; 2], size: [f32; 2], condition: Condition) {
    if condition == Condition::Never {
        return;
    }
    // The live Ui owns this thread's context; only scalar arguments cross into C++ (#3803).
    unsafe {
        oblikovati_gui_window_position(position[0], position[1], condition as i32);
        oblikovati_gui_window_size(size[0], size[1], condition as i32);
    }
}
/// Place the next window where it first appears; example: a list opened beside its menu row.
pub(crate) fn next_window_position(_: &Ui, position: [f32; 2]) {
    // The live Ui owns this thread's context; only scalar arguments cross into C++ (#3803).
    unsafe {
        oblikovati_gui_window_position(position[0], position[1], Condition::Appearing as i32)
    };
}
/// Move the current window cursor; example: `cursor_position(ui, [16.,59.])` before a ribbon control.
pub(crate) fn cursor_position(_: &Ui, position: [f32; 2]) {
    // The Ui borrow keeps the current context and window alive for this scalar call.
    unsafe { oblikovati_gui_cursor_position(position[0], position[1]) };
}
/// Draw a sized command control; example: `button(ui, "Save", [0.;2])` uses natural sizing.
pub(crate) fn button(ui: &Ui, label: &str, size: [f32; 2]) -> bool {
    toggle_button(ui, label, size, false)
}
/// Draw an owned mode's pressed state through scalars; example: `toggle_button(ui, "Construction", [108.,18.], true)`.
pub(crate) fn toggle_button(_: &Ui, label: &str, size: [f32; 2], selected: bool) -> bool {
    let label = ImString::new(label);
    // ImString owns terminated UTF-8 bytes through the call; C++ retains no label pointer.
    unsafe { oblikovati_gui_button(label.as_ptr().cast(), size[0], size[1], selected) }
}
/// Draw a current-window row; example: `selectable(ui, "##row", true, true)` permits double-click activation.
pub(crate) fn selectable(_: &Ui, label: &str, selected: bool, double_click: bool) -> bool {
    let flags = if double_click {
        SelectableFlags::ALLOW_DOUBLE_CLICK.bits()
    } else {
        0
    };
    let label = ImString::new(label);
    // The label owner and Ui remain live; no C++ vector class crosses the ABI.
    unsafe { oblikovati_gui_selectable(label.as_ptr().cast(), selected, flags as i32) }
}
/// A row inside a dialog: a plain selectable would close the dialog that contains it.
/// Example: `dialog_selectable(ui, "Metric", true)` keeps the dialog open when pressed.
pub(crate) fn dialog_selectable(_: &Ui, label: &str, selected: bool) -> bool {
    let flags = SelectableFlags::DONT_CLOSE_POPUPS.bits();
    let label = ImString::new(label);
    // The label owner and Ui remain live; no C++ vector class crosses the ABI.
    unsafe { oblikovati_gui_selectable(label.as_ptr().cast(), selected, flags as i32) }
}
/// A button naming its choice, and the list it opens; returns the row chosen this frame.
/// Example: `popup_choice(ui, "Design View: Primary", "##view", &rows)`.
pub(crate) fn popup_choice<Row: AsRef<str>>(
    ui: &Ui,
    label: &str,
    popup: &str,
    rows: &[Row],
) -> Option<usize> {
    if ui.button(label) {
        ui.open_popup(popup);
    }
    let _list = ui.begin_popup(popup)?;
    rows.iter().position(|row| ui.menu_item(row.as_ref()))
}
/// Leave no window focused; example: after the panel that held the keyboard has closed.
pub(crate) fn release_keyboard_focus(_: &Ui) {
    // A null name is the documented form for removing focus; no string crosses the ABI.
    unsafe { imgui::sys::igSetWindowFocus_Str(std::ptr::null()) };
}
/// Whether a list or dialog opened from inside the current popup is still open.
pub(crate) fn nested_popup_open(_: &Ui) -> bool {
    let flags = imgui::sys::ImGuiPopupFlags_AnyPopupId as i32;
    // A null identity is the documented form with AnyPopupId; no string crosses the ABI.
    unsafe { imgui::sys::igIsPopupOpen_Str(std::ptr::null(), flags) }
}
/// Whether any popup at any level is open; application keys wait for it.
pub(crate) fn any_popup_open(_: &Ui) -> bool {
    let flags =
        (imgui::sys::ImGuiPopupFlags_AnyPopupId | imgui::sys::ImGuiPopupFlags_AnyPopupLevel) as i32;
    // A null identity is the documented form with AnyPopupId; no string crosses the ABI.
    unsafe { imgui::sys::igIsPopupOpen_Str(std::ptr::null(), flags) }
}
/// Check pointer hover even after keyboard navigation; example: `item_mouse_hovered(ui)` after a row.
pub(crate) fn item_mouse_hovered(_: &Ui) -> bool {
    // The live Ui owns the context; imgui-rs omits this scalar flag.
    unsafe { imgui::sys::igIsItemHovered(imgui::sys::ImGuiHoveredFlags_NoNavOverride as i32) }
}
/// Draw literal UTF-8 in the current window; example: `literal_text(ui, point, color, "Part##1")` preserves hashes.
pub(crate) fn literal_text(ui: &Ui, position: [f32; 2], color: [f32; 4], text: &str) {
    draw_literal_text(ui, position, color, text, false);
}
/// Draw below windows without a vector ABI; example: `background_literal_text(ui, point, color, "d0 = 10 mm")`.
pub(crate) fn background_literal_text(ui: &Ui, position: [f32; 2], color: [f32; 4], text: &str) {
    draw_literal_text(ui, position, color, text, true);
}
fn draw_literal_text(_: &Ui, position: [f32; 2], color: [f32; 4], text: &str, background: bool) {
    let color: ImColor32 = color.into();
    // The byte slice outlives AddText, which copies its glyph vertices into the current draw list.
    unsafe {
        oblikovati_gui_literal_text(
            text.as_ptr(),
            text.len(),
            position[0],
            position[1],
            color.to_bits(),
            background,
        )
    };
}

#[cfg(test)]
#[path = "../tests/widget_abi/native.rs"]
mod tests;

/// A live child window is always ended, including clipped frames and unwinding.
pub(crate) struct ScrollingRegion<'ui> {
    pub visible: bool,
    _ui: &'ui Ui,
}
impl Drop for ScrollingRegion<'_> {
    fn drop(&mut self) {
        // SAFETY: this token retains the UI that opened exactly one child, including a false BeginChild result.
        unsafe { imgui::sys::igEndChild() };
    }
}
/// Open scalar-sized scrolling content; example: `let region = scroll_region(ui, "panels", [640.,80.]);`.
pub(crate) fn scroll_region<'ui>(
    ui: &'ui Ui,
    identity: &str,
    size: [f32; 2],
) -> ScrollingRegion<'ui> {
    let identity = ImString::new(identity);
    // SAFETY: the UI owns the current context; C++ copies the live ID and constructs its vector (#3803, #3849).
    let visible =
        unsafe { oblikovati_gui_begin_scroll_region(identity.as_ptr().cast(), size[0], size[1]) };
    ScrollingRegion { visible, _ui: ui }
}
/// Reserve layout space through scalar ABI; example: `dummy(ui, [364.,0.])` keeps the Create grid's width.
pub(crate) fn dummy(_: &Ui, size: [f32; 2]) {
    // SAFETY: the owning frame retains the context; only C++ constructs the aggregate argument (#3803).
    unsafe { oblikovati_gui_dummy(size[0], size[1]) };
}

/// Keep a floating dialog inside the current content viewport while preserving valid user placement.
/// Example: `fit_dialog(ui, "Parameters", [320.,8.], [540.,360.])` before opening that window.
pub(crate) fn fit_dialog(_: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
    let title = ImString::new(title);
    // SAFETY: the frame owns the current context; the live ID and scalar values are copied by C++ (#3803, #3850).
    unsafe {
        oblikovati_gui_fit_dialog(
            title.as_ptr().cast(),
            origin[0],
            origin[1],
            size[0],
            size[1],
        )
    };
}

/// Over a panel even while a popup just opened by the same click blocks the panel's hover state.
pub(crate) fn over_any_panel() -> imgui::WindowHoveredFlags {
    imgui::WindowHoveredFlags::ANY_WINDOW | imgui::WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP
}
/// Draw text centred on a point and turned by an angle in radians, in the current window.
/// Dear ImGui has no turned text, so the glyph quads just drawn are turned where they lie.
pub(crate) fn rotated_text(ui: &Ui, center: [f32; 2], color: [f32; 4], text: &str, angle: f32) {
    let [width, height] = ui.calc_text_size(text);
    let list = ui.get_window_draw_list();
    // SAFETY: the current window's list is live for this frame, and these reads and writes stay
    // inside the vertices this call appended; nothing else draws between the two reads.
    unsafe {
        let raw = imgui::sys::igGetWindowDrawList();
        let first = (*raw).VtxBuffer.Size;
        list.add_text(
            [center[0] - width / 2., center[1] - height / 2.],
            color,
            text,
        );
        let last = (*raw).VtxBuffer.Size;
        let (sin, cos) = angle.sin_cos();
        for index in first..last {
            let vertex = &mut *(*raw).VtxBuffer.Data.add(index as usize);
            let (x, y) = (vertex.pos.x - center[0], vertex.pos.y - center[1]);
            vertex.pos.x = center[0] + x * cos - y * sin;
            vertex.pos.y = center[1] + x * sin + y * cos;
        }
    }
}

/// Scalar payload for an owned ribbon image; example: a 40px glyph above its caption.
pub(crate) struct IconButtonPresentation {
    pub texture: crate::GuiTextureKey,
    pub size: [f32; 2],
    pub icon: [f32; 2],
    pub large: bool,
    pub show_caption: bool,
}
/// Keep normal button identity and last-item state; example: `icon_button(ui, "Rectangle", presentation)`.
pub(crate) fn icon_button(_: &Ui, label: &str, presentation: IconButtonPresentation) -> bool {
    let label = ImString::new(label);
    // #3803: C++ owns all ImVec2/ImRect construction; only validated resource IDs cross ABI.
    // SAFETY: the NUL-terminated label stays live for this synchronous call; resource keys are scalars.
    unsafe {
        oblikovati_gui_icon_button(
            label.as_ptr().cast(),
            presentation.texture.raw() as u32,
            presentation.size[0],
            presentation.size[1],
            presentation.icon[0],
            presentation.icon[1],
            presentation.large,
            presentation.show_caption,
        )
    }
}
