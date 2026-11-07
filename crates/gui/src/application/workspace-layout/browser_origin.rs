// SPDX-License-Identifier: GPL-2.0-only
//! The Origin branch's rows: their menus, what a chosen row shows in the view, and clearing them.
use super::browser_view::{BrowserBranch, BrowserView, muted_menu_item};
use crate::browser::ORIGIN_ROWS;
use crate::{GuiCommandState, GuiFrameRequest, GuiModelBrowser, GuiShellAction};
use imgui::{MouseButton, Ui};

const ORIGIN_MENU: &str = "##origin-menu";
const ROW_MENU: &str = "##origin-row-menu";
const ACCENT: [f32; 4] = [0.25, 0.6, 1., 1.];
const PLANE_FILL: [f32; 4] = [0.25, 0.6, 1., 0.3];
const AXIS_THICKNESS: f32 = 2.;

/// One origin axis in pixels, with the row that names it.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiOriginAxis {
    pub label: &'static str,
    pub ends: [[i32; 2]; 2],
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
    view: &mut BrowserView,
    row: usize,
    label: &str,
    actions: &mut Vec<GuiShellAction>,
) {
    let _row = ui.push_id(label);
    ui.bullet();
    ui.same_line();
    let selected = view.selected_origin_row == Some(row);
    if ui.selectable_config(label).selected(selected).build() {
        view.selected_origin_row = Some(row);
    }
    if ui.is_item_hovered() && ui.is_mouse_clicked(MouseButton::Right) {
        view.selected_origin_row = Some(row);
        ui.open_popup(ROW_MENU);
    }
    let Some(_menu) = ui.begin_popup(ROW_MENU) else {
        return;
    };
    origin_row_commands(ui, label, actions);
}
fn origin_row_commands(ui: &Ui, label: &str, actions: &mut Vec<GuiShellAction>) {
    use crate::GuiOriginPlaneKind::{Xy, Xz, Yz};
    let plane = [Xy, Xz, Yz].into_iter().find(|kind| kind.label() == label);
    if let Some(plane) = plane {
        if ui.menu_item("New Sketch") {
            actions.push(GuiShellAction::StartSketchOn(plane));
        }
        for row in ["Measure", "Visibility"] {
            muted_menu_item(ui, row);
        }
        auto_resize_row(ui);
        return;
    }
    if label.ends_with("Axis") {
        for row in ["Copy", "Measure", "Visibility"] {
            muted_menu_item(ui, row);
        }
        auto_resize_row(ui);
        return;
    }
    for row in [
        "Copy",
        "Measure",
        "Create Note",
        "Visibility",
        "Relationships",
    ] {
        muted_menu_item(ui, row);
    }
}
/// Datums keep to the size of what they surround in this build, so the row only says so.
fn auto_resize_row(ui: &Ui) {
    ui.menu_item_config("Auto-Resize")
        .selected(true)
        .enabled(false)
        .build();
}
/// A click on nothing in the part's view, once no menu is left to close, chooses no row.
pub(crate) fn clear_on_empty_click(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &GuiModelBrowser,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let on_nothing = ui.is_mouse_clicked(MouseButton::Left)
        && !ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        && !crate::widget_abi::any_popup_open(ui);
    if !on_nothing || state.authoring != crate::GuiAuthoringContext::Part {
        return;
    }
    view.selected_origin_row = None;
    if browser.selected_feature.is_some() || !browser.selected_sketches.is_empty() {
        actions.push(GuiShellAction::ClearBrowserSelection {
            document: state.document.unwrap_or(0),
        });
    }
}
/// Show the chosen origin row in the view: a plane as a clear surface, an axis as a line.
pub(crate) fn draw_selected_origin(ui: &Ui, request: &GuiFrameRequest, selected: Option<usize>) {
    let Some(label) = selected.and_then(|row| ORIGIN_ROWS.get(row)) else {
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
        let [a, b, c, d] = plane.corners.map(logical);
        list.add_triangle(a, b, c, PLANE_FILL).filled(true).build();
        list.add_triangle(a, c, d, PLANE_FILL).filled(true).build();
    }
    if let Some(axis) = commands
        .origin_axes
        .iter()
        .find(|axis| axis.label == *label)
    {
        list.add_line(logical(axis.ends[0]), logical(axis.ends[1]), ACCENT)
            .thickness(AXIS_THICKNESS)
            .build();
    }
}
