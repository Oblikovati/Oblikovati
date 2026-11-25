// SPDX-License-Identifier: GPL-2.0-only
use imgui::{Condition, ImColor32, ImString, SelectableFlags, Ui};
#[path = "widget_text_projection.rs"]
mod text_projection;
pub(crate) use text_projection::projected_text;
#[cfg(test)]
#[path = "widget_text_line_advance_tests.rs"]
mod text_line_advance_tests;
unsafe extern "C" {
    fn oblikovati_gui_push_text_line_advance(factor: f32) -> f32;
    fn oblikovati_gui_pop_text_line_advance(original: f32);
    fn oblikovati_gui_shade_linked_field(
        label: *const std::ffi::c_char,
        line: *const u8,
        prefix: usize,
        end: usize,
        row: usize,
    );
    fn oblikovati_gui_window_is_moving() -> bool;
    fn oblikovati_gui_workspace_dockspace(width: f32, height: f32) -> u32;
    fn oblikovati_gui_outer_dockspace(width: f32, height: f32, seed_asset_dock: bool) -> u32;
    fn oblikovati_gui_ensure_tabbed_window(title: *const u8, group: *const u8);
    fn oblikovati_gui_draw_dock_tab_actions() -> u32;
    fn oblikovati_gui_draw_workspace_panel_menu() -> u32;
    fn oblikovati_gui_prepare_dock_hit_test(x: f32, y: f32);
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
    fn oblikovati_gui_fit_dialog(
        title: *const u8,
        x: f32,
        y: f32,
        width: f32,
        height: f32,
        reset_size_on_reopen: bool,
    );
    fn oblikovati_gui_begin_scroll_region(identity: *const u8, width: f32, height: f32) -> bool;
    fn oblikovati_gui_dummy(width: f32, height: f32);
    fn oblikovati_gui_window_is_docked() -> bool;
    fn oblikovati_gui_fit_viewport_properties(
        title: *const u8,
        left: f32,
        top: f32,
        right: f32,
        bottom: f32,
        width: f32,
        height: f32,
    );
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
/// Tint display spans in a native multiline widget just drawn in this frame.
/// For example, shade a projected numeric field in the Format Text Content input.
pub(crate) fn shade_linked_text(
    _: &Ui,
    label: &std::ffi::CStr,
    projection: &oblikovati_model::sketch::SketchTextFieldProjection,
) {
    let (mut scanned, mut line_start, mut row) = (0, 0, 0);
    for field in &projection.fields {
        for newline in projection.content[scanned..field.display.start].match_indices('\n') {
            line_start = scanned + newline.0 + 1;
            row += 1;
        }
        scanned = field.display.end;
        let line = &projection.content[line_start..scanned];
        // The UI frame and the string slices outlive the synchronous draw-list update.
        unsafe {
            oblikovati_gui_shade_linked_field(
                label.as_ptr(),
                line.as_ptr(),
                field.display.start - line_start,
                line.len(),
                row,
            );
        }
    }
}
/// Set the row height for one native multiline input, for example `text_line_advance(ui, 2.)`.
pub(crate) fn text_line_advance(ui: &Ui, factor: f32) -> Option<TextLineAdvance<'_>> {
    if factor == 1. || !(factor * ui.current_font_size()).is_finite() || factor <= 0. {
        return None;
    }
    // The UI frame owns the current font stack. The guard restores the original size
    // before any sibling widget reads it, including when the caller unwinds.
    let original = unsafe { oblikovati_gui_push_text_line_advance(factor) };
    Some(TextLineAdvance { original, _ui: ui })
}

pub(crate) struct TextLineAdvance<'ui> {
    original: f32,
    _ui: &'ui Ui,
}

impl Drop for TextLineAdvance<'_> {
    fn drop(&mut self) {
        // A borrowed Ui keeps the same frame and font stack live until this call returns.
        unsafe { oblikovati_gui_pop_text_line_advance(self.original) };
    }
}
/// Let workspace targets receive pointer input while an ImGui window is dragged.
/// Example: `moving_window(ui)` when drawing a fixed document-tab strip.
pub(crate) fn moving_window(_: &Ui) -> bool {
    unsafe { oblikovati_gui_window_is_moving() }
}
/// Route the next drag frame to the outer target when the pointer is over it.
/// Example: `prepare_dock_hit_test(position)` before starting the next ImGui frame.
pub(crate) fn prepare_dock_hit_test(position: [f32; 2]) {
    unsafe { oblikovati_gui_prepare_dock_hit_test(position[0], position[1]) };
}
/// Draw a dockspace in the current host and return its central bounds when available.
/// Example: `workspace_dockspace(ui, [800., 600.])` in a transparent host window.
pub(crate) fn workspace_dockspace(_: &Ui, size: [f32; 2]) -> Option<[f32; 4]> {
    let id = unsafe { oblikovati_gui_workspace_dockspace(size[0], size[1]) };
    dockspace_center(id)
}
/// Draw the outer workspace, including the default Model dock.
/// Example: `outer_dockspace(ui, [800., 600.], false)` in its transparent host.
pub(crate) fn outer_dockspace(
    _: &Ui,
    size: [f32; 2],
    seed_asset_dock: bool,
) -> Option<([f32; 4], Option<[f32; 2]>)> {
    let id = unsafe { oblikovati_gui_outer_dockspace(size[0], size[1], seed_asset_dock) };
    let mut span = [0.; 2];
    let [left, right] = &mut span;
    let has_bottom = unsafe { oblikovati_gui_outer_bottom_span(id, left, right) };
    dockspace_center(id).map(|center| (center, has_bottom.then_some(span)))
}
/// Dock a newly-created tab beside its group window; existing placement remains user-controlled.
pub(crate) fn ensure_tabbed_window(_: &Ui, title: &str, group: &str) {
    let title = ImString::new(title);
    let group = ImString::new(group);
    // SAFETY: the UI frame owns the current ImGui context; C++ copies these strings immediately.
    unsafe {
        oblikovati_gui_ensure_tabbed_window(title.as_ptr().cast(), group.as_ptr().cast());
    }
}
/// Read the action selected from native ImGui dock-tab buttons after all windows are submitted.
pub(crate) fn dock_tab_action(_: &Ui) -> (Option<DockTabAction>, bool) {
    const DRAW_AGAIN: u32 = 1 << 31;
    let result = unsafe { oblikovati_gui_draw_dock_tab_actions() };
    (
        decode_dock_action(result & !DRAW_AGAIN),
        result & DRAW_AGAIN != 0,
    )
}
/// Draw the same panel menu when the workspace has no browser tabs.
/// Example: `workspace_panel_menu(ui)` inside the panel launcher overlay.
pub(crate) fn workspace_panel_menu(_: &Ui) -> Option<DockTabAction> {
    decode_dock_action(unsafe { oblikovati_gui_draw_workspace_panel_menu() })
}
fn decode_dock_action(code: u32) -> Option<DockTabAction> {
    match code {
        1 => Some(DockTabAction::ModelBrowser),
        2 => Some(DockTabAction::LogPanel),
        4 => Some(DockTabAction::Favorites),
        8 => Some(DockTabAction::MaterialBrowser),
        16 => Some(DockTabAction::AppearanceBrowser),
        32 => Some(DockTabAction::Parameters),
        64 => Some(DockTabAction::SelectionFilter),
        128 => Some(DockTabAction::HistoryBrowser),
        256 => Some(DockTabAction::Search),
        _ => None,
    }
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum DockTabAction {
    ModelBrowser,
    LogPanel,
    Favorites,
    MaterialBrowser,
    AppearanceBrowser,
    Parameters,
    SelectionFilter,
    HistoryBrowser,
    Search,
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
/// Check a named popup at the current menu level, such as `popup_is_open(ui, c"File")`.
pub(crate) fn popup_is_open(_: &Ui, label: &std::ffi::CStr) -> bool {
    // The Ui owns the context and the label is already NUL-terminated.
    unsafe { imgui::sys::igIsPopupOpen_Str(label.as_ptr(), 0) }
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
/// Check pointer hover on a disabled item after keyboard navigation; for example, a command tooltip.
pub(crate) fn disabled_item_mouse_hovered(_: &Ui) -> bool {
    let flags = imgui::sys::ImGuiHoveredFlags_NoNavOverride
        | imgui::sys::ImGuiHoveredFlags_AllowWhenDisabled;
    // The live Ui owns the context; imgui-rs omits the pointer-only flag.
    unsafe { imgui::sys::igIsItemHovered(flags as i32) }
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
pub(crate) fn window_is_docked(_: &Ui) -> bool {
    // The Ui borrow keeps the active Dear ImGui window available.
    unsafe { oblikovati_gui_window_is_docked() }
}

/// Keep floating properties inside the live viewport. Docked properties follow their dock.
/// Example: `fit_viewport_properties(ui, "Extrude", area, [280., 340.])`.
pub(crate) fn fit_viewport_properties(
    _: &Ui,
    title: &str,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    size: [f32; 2],
) {
    let horizontal = ((area.right - area.left - 1.) * 0.5).clamp(0., 14.);
    let vertical = ((area.bottom - area.top - 1.) * 0.5).clamp(0., 14.);
    let title = ImString::new(title);
    // SAFETY: ImGui consumes the scalar geometry for its next window in this frame.
    unsafe {
        oblikovati_gui_fit_viewport_properties(
            title.as_ptr().cast(),
            area.left + horizontal,
            area.top + vertical,
            area.right - horizontal,
            area.bottom - vertical,
            size[0],
            size[1],
        )
    };
}

/// Example: `fit_dialog(ui, "Parameters", [320.,8.], [540.,360.])` before opening that window.
pub(crate) fn fit_dialog(_: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
    fit_floating_dialog(title, origin, size, false);
}

/// Fit a floating dialog that remembers placement and restores its default size on reopen.
pub(crate) fn fit_dialog_with_default_size(_: &Ui, title: &str, origin: [f32; 2], size: [f32; 2]) {
    fit_floating_dialog(title, origin, size, true);
}

fn fit_floating_dialog(title: &str, origin: [f32; 2], size: [f32; 2], reset_size_on_reopen: bool) {
    let title = ImString::new(title);
    // SAFETY: the frame owns the current context; the live ID and scalar values are copied by C++ (#3803, #3850).
    unsafe {
        oblikovati_gui_fit_dialog(
            title.as_ptr().cast(),
            origin[0],
            origin[1],
            size[0],
            size[1],
            reset_size_on_reopen,
        )
    };
}

/// Over a panel even while a popup just opened by the same click blocks the panel's hover state.
pub(crate) fn over_any_panel() -> imgui::WindowHoveredFlags {
    imgui::WindowHoveredFlags::ANY_WINDOW | imgui::WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_POPUP
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

/// Compute the exact ImGui window identity, including `###` semantics (#4073).
/// Example: `dock_window_identity(c"Find###tool")` identifies the stable suffix.
pub(crate) fn dock_window_identity(title: &std::ffi::CStr) -> u32 {
    unsafe { imgui::sys::igImHashStr(title.as_ptr(), 0, 0) }
}

/// Notify ImGui of a validated programmatic dock tab closure (#4073).
/// Example: `close_dock_tab(ui, c"Tool")` after the owner closes.
pub(crate) fn close_dock_tab(_ui: &Ui, title: &std::ffi::CStr) {
    unsafe { imgui::sys::igSetTabItemClosed(title.as_ptr()) };
}

/// Find the active draw-list clip bottom; example: align a label within the window clip.
pub(crate) fn window_draw_clip_bottom(_ui: &Ui) -> f32 {
    let mut maximum = imgui::sys::ImVec2 { x: 0., y: 0. };
    // The current window and its draw list remain live for the enclosing ImGui frame.
    unsafe {
        imgui::sys::ImDrawList_GetClipRectMax(&mut maximum, imgui::sys::igGetWindowDrawList())
    };
    maximum.y
}
