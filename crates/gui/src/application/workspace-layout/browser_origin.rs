// SPDX-License-Identifier: GPL-2.0-only
//! The Origin branch's rows: their menus, what a chosen row shows in the view, and clearing them.
use super::browser_view::{BrowserBranch, BrowserView, muted_menu_item};
use crate::application::model_browser::relationships::RelatedRow;
use crate::browser::ORIGIN_ROWS;
use crate::{GuiCommandState, GuiFrameRequest, GuiShellAction};
use imgui::{MouseButton, Ui};

const ORIGIN_MENU: &str = "##origin-menu";
const ROW_MENU: &str = "##origin-row-menu";
const ACCENT: [f32; 4] = [0.25, 0.6, 1., 1.];
const PLANE_FILL: [f32; 4] = [0.25, 0.6, 1., 0.3];
/// A row made visible and not chosen shows as a quiet part of the view.
const MUTED_FILL: Color = [0.45, 0.62, 0.8, 0.35];
const MUTED_LINE: Color = [0.75, 0.6, 0.3, 0.9];
type Color = [f32; 4];
const AXIS_THICKNESS: f32 = 2.;

/// One origin axis in pixels, with the row that names it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiOriginAxis {
    pub label: &'static str,
    pub ends: [[i32; 2]; 2],
    /// Scene-sized endpoints for an automatically resized browser datum.
    pub automatic_ends: [[i32; 2]; 2],
}
/// A right-click on the Origin row; its direct rows stay when its children collapse.
pub(crate) fn origin_branch_menu(ui: &Ui, view: &mut BrowserView) {
    if ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right) {
        ui.open_popup(ORIGIN_MENU);
    }
    let Some(_menu) = ui.begin_popup(ORIGIN_MENU) else {
        return;
    };
    if ui.menu_item("Expand All Children") {
        view.open_branch(BrowserBranch::Origin);
    }
    // Origin's rows have nothing below them, so there is nothing for this to fold yet.
    ui.menu_item("Collapse All Children");
}
/// One origin row; a press on it, with either button, chooses it.
pub(crate) fn origin_row(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    row: usize,
    label: &str,
    actions: &mut Vec<GuiShellAction>,
) {
    let _row = ui.push_id(label);
    use crate::icons::IconGlyph;
    let glyph = match row {
        0..=2 => IconGlyph::WorkPlane,
        3..=5 => IconGlyph::WorkAxis,
        _ => IconGlyph::WorkPoint,
    };
    crate::browser::browser_art::prefix(ui, view, glyph, ui.cursor_screen_pos());
    ui.dummy([16., ui.text_line_height()]);
    ui.same_line_with_spacing(0., 4.);
    let selected = view.selected_origin_row == Some(row);
    if ui.selectable_config(label).selected(selected).build() {
        choose_origin_row(state, view, row, actions);
        view.picked_plane_reference =
            origin_plane_at_row(row).map(|plane| plane.work_reference().to_owned());
        view.picked_axis_reference = origin_axis_at_row(row);
    }
    if ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right) {
        choose_origin_row(state, view, row, actions);
        ui.open_popup(ROW_MENU);
    }
    if !selected && view.focused_origin_row == Some(row) {
        ui.get_window_draw_list()
            .add_rect(
                ui.item_rect_min(),
                ui.item_rect_max(),
                ui.style_color(imgui::StyleColor::NavHighlight),
            )
            .build();
    }
    let Some(_menu) = ui.begin_popup(ROW_MENU) else {
        return;
    };
    origin_row_commands(ui, state, view, row, label, actions);
}
pub(super) fn origin_plane_at_row(row: usize) -> Option<crate::GuiOriginPlaneKind> {
    use crate::GuiOriginPlaneKind::{Xy, Xz, Yz};
    let label = *crate::browser::ORIGIN_ROWS.get(row)?;
    [Xy, Xz, Yz]
        .into_iter()
        .find(|plane| plane.label() == label)
}
pub(super) fn origin_axis_at_row(
    row: usize,
) -> Option<oblikovati_model::feature::work_axis::WorkAxisReference> {
    use oblikovati_model::feature::work_axis::WorkAxisReference::{OriginX, OriginY, OriginZ};
    match *crate::browser::ORIGIN_ROWS.get(row)? {
        "X Axis" => Some(OriginX),
        "Y Axis" => Some(OriginY),
        "Z Axis" => Some(OriginZ),
        _ => None,
    }
}
/// Only one kind of row reads as chosen at a time, so whatever else was chosen lets go.
fn choose_origin_row(
    state: &GuiCommandState,
    view: &mut BrowserView,
    row: usize,
    actions: &mut Vec<GuiShellAction>,
) {
    view.selected_origin_row = Some(row);
    view.focused_origin_row = Some(row);
    view.origin_folder_chosen = false;
    actions.push(GuiShellAction::BrowserChoice {
        document: state.document.unwrap_or(0),
        choice: crate::GuiBrowserChoice::Nothing,
    });
}
fn origin_row_commands(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    row: usize,
    label: &str,
    actions: &mut Vec<GuiShellAction>,
) {
    let plane = origin_plane_at_row(row);
    if let Some(plane) = plane {
        if ui.menu_item("New Sketch") {
            actions.push(GuiShellAction::StartSketchOn(plane));
        }
        muted_menu_item(ui, "Measure");
        visibility_row(ui, state, view, row);
        auto_resize_row(ui, state, view, row);
    } else if label.ends_with("Axis") {
        for listed in ["Copy", "Measure"] {
            muted_menu_item(ui, listed);
        }
        visibility_row(ui, state, view, row);
        auto_resize_row(ui, state, view, row);
    } else {
        for listed in ["Copy", "Measure", "Create Note"] {
            muted_menu_item(ui, listed);
        }
        visibility_row(ui, state, view, row);
    }
    if ui.menu_item("Relationships") {
        view.relationships.request(RelatedRow::Origin(row));
    }
}
/// An origin row's visibility is view state of the open document, as a body's is.
fn visibility_row(ui: &Ui, state: &GuiCommandState, view: &mut BrowserView, row: usize) {
    let key = (state.document.unwrap_or(0), row);
    let visible = view.visible_origin_rows.contains(&key);
    if ui.menu_item_config("Visibility").selected(visible).build() {
        if visible {
            view.visible_origin_rows.remove(&key);
        } else {
            view.visible_origin_rows.insert(key);
        }
    }
}
/// The arrow keys, while Origin or a row under it is chosen.
pub(crate) fn follow_keys(
    ui: &Ui,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
    state: &GuiCommandState,
) {
    use crate::application::model_browser::browser_keys::{OriginCursor, moved, pressed_arrow};
    if view.selected_origin_row.is_none() && !view.origin_folder_chosen {
        return;
    }
    let Some(key) = pressed_arrow(ui) else {
        return;
    };
    view.arrow_consumed = true;
    let cursor = OriginCursor {
        row: view.selected_origin_row,
        folder: view.origin_folder_chosen,
        open: view.origin_open,
    };
    let step = moved(cursor, key);
    view.selected_origin_row = step.cursor.row;
    view.focused_origin_row = step.cursor.row;
    view.origin_folder_chosen = step.cursor.folder;
    view.set_origin_open(step.set_open);
    if step.cursor.row != cursor.row {
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: crate::GuiBrowserChoice::Nothing,
        });
    }
}
/// Auto-sized datums follow the visible scene; fixed datums keep the standard display extent.
fn auto_resize_row(ui: &Ui, state: &GuiCommandState, view: &mut BrowserView, row: usize) {
    let Some(document) = state.document else {
        return;
    };
    let selected = view.origin_auto_resize(document, row);
    if ui
        .menu_item_config("Auto-Resize")
        .selected(selected)
        .build()
    {
        view.toggle_origin_auto_resize(document, row);
    }
}
/// A click on the part's view, once no menu is left to close: no origin row stays chosen, and
/// the owner looks for a face under the pointer.
pub(crate) fn follow_view_click(
    ui: &Ui,
    state: &GuiCommandState,
    view: &mut BrowserView,
    part_context_open: bool,
    selector_context_gesture: bool,
    actions: &mut Vec<GuiShellAction>,
) {
    let left_click = ui.is_mouse_clicked(MouseButton::Left)
        && (part_context_open || !crate::widget_abi::any_popup_open(ui));
    let context_click = ui.is_mouse_clicked(MouseButton::Right)
        && (selector_context_gesture || !crate::widget_abi::any_popup_open(ui));
    // An active text field can commit when clicked away. The panel underneath the click
    // still owns it even when ImGui would otherwise report its window as blocked.
    let panel_hover = crate::widget_abi::over_any_panel()
        | imgui::WindowHoveredFlags::ALLOW_WHEN_BLOCKED_BY_ACTIVE_ITEM;
    let on_view = (left_click || context_click) && !ui.is_window_hovered_with_flags(panel_hover);
    let [x, y] = ui.io().mouse_pos;
    // A tab may change the document in this same frame; the click was never on its model.
    let on_document = view.document_area.is_some_and(|area| {
        (area.left..area.right).contains(&x) && (area.top..area.bottom).contains(&y)
    });
    let assembly = state.authoring == crate::GuiAuthoringContext::OtherDocument
        && state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.kind == crate::GuiBrowserKind::Assembly);
    if !on_view
        || !on_document
        || view.view_clicks_taken
        || (assembly && context_click && !left_click)
        || (state.authoring != crate::GuiAuthoringContext::Part && !assembly)
    {
        return;
    }
    view.selected_origin_row = None;
    view.origin_folder_chosen = false;
    let scale = view.layout_scale;
    let document = state.document.unwrap_or(0);
    let pixel = [(x * scale[0]) as i32, (y * scale[1]) as i32];
    if context_click && !assembly {
        actions.push(GuiShellAction::ContextClickModel { document, pixel });
    } else if left_click {
        actions.push(GuiShellAction::ClickModel {
            document,
            pixel,
            toggle: ui.io().key_ctrl || ui.io().key_shift,
            extrude_profile: None,
        });
    }
}
/// Show the chosen origin row in the view, in the accent, and those made visible, muted.
pub(crate) fn draw_selected_origin(
    ui: &Ui,
    request: &GuiFrameRequest,
    selected: Option<usize>,
    visible: impl Iterator<Item = usize>,
    view: &BrowserView,
) {
    for row in visible {
        draw_origin_row(ui, request, row, MUTED_FILL, MUTED_LINE, view);
    }
    if let Some(row) = selected {
        draw_origin_row(ui, request, row, PLANE_FILL, ACCENT, view);
    }
}
fn draw_origin_row(
    ui: &Ui,
    request: &GuiFrameRequest,
    row: usize,
    fill: Color,
    line: Color,
    view: &BrowserView,
) {
    let Some(label) = ORIGIN_ROWS.get(row) else {
        return;
    };
    let scale = request.layout_scale();
    let logical = |pixel: [i32; 2]| [pixel[0] as f32 / scale[0], pixel[1] as f32 / scale[1]];
    let list = ui.get_background_draw_list();
    let commands = &request.commands;
    if let Some(plane) = commands
        .origin_planes
        .iter()
        .find(|plane| plane.kind.label() == *label)
    {
        let automatic = view.origin_auto_resize(commands.document.unwrap_or(0), row);
        let corners = if automatic {
            plane.automatic_corners
        } else {
            plane.corners
        };
        let [a, b, c, d] = corners.map(logical);
        list.add_triangle(a, b, c, fill).filled(true).build();
        list.add_triangle(a, c, d, fill).filled(true).build();
    }
    if let Some(axis) = commands
        .origin_axes
        .iter()
        .find(|axis| axis.label == *label)
    {
        let automatic = view.origin_auto_resize(commands.document.unwrap_or(0), row);
        let ends = if automatic {
            axis.automatic_ends
        } else {
            axis.ends
        };
        list.add_line(logical(ends[0]), logical(ends[1]), line)
            .thickness(AXIS_THICKNESS)
            .build();
    }
}

#[cfg(test)]
#[path = "../../../tests/application/workspace-layout/browser_origin_focus.rs"]
mod focus_tests;
