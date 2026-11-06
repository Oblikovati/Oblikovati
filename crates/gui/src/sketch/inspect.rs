// SPDX-License-Identifier: GPL-2.0-only
//! Measure, its panel, and the sketch canvas menu that opens the inspection commands.
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;

/// What the owner has measured so far, one line per row; example: `Selection 1: Curve`.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiMeasurePanel {
    pub rows: Vec<String>,
}
/// A choice in the sketch's inspection commands.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiInspectAction {
    StartMeasure,
    FinishMeasure,
    OpenRegionProperties,
    CalculateRegions,
    FinishRegionProperties,
}
const CANVAS_MENU: &str = "##sketch-canvas-menu";
const MEASURE_SIZE: [f32; 2] = [340., 300.];

/// Draw the open measurement; example: rows of a selected curve, with Done below them.
pub(crate) fn measure_panel(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(panel) = &state.measure else {
        return;
    };
    let [width, _] = ui.io().display_size;
    crate::widget_abi::fit_dialog(
        ui,
        "Measure",
        [width - MEASURE_SIZE[0] - 24., 220.],
        MEASURE_SIZE,
    );
    ui.window("Measure").collapsible(false).build(|| {
        if panel.rows.is_empty() {
            ui.text("Select geometry to measure");
        }
        for row in &panel.rows {
            ui.text(row);
        }
        ui.separator();
        if ui.button("Done") {
            actions.push(GuiShellAction::Inspect(GuiInspectAction::FinishMeasure));
        }
    });
}
/// The empty canvas's menu while a sketch waits at Ready; a running command has its own.
pub(crate) fn canvas_menu(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    if state.authoring != GuiAuthoringContext::Sketch || !state.sketch_at_ready {
        return;
    }
    let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
    if !over_panel && ui.is_mouse_released(imgui::MouseButton::Right) {
        ui.open_popup(CANVAS_MENU);
    }
    let Some(_popup) = ui.begin_popup(CANVAS_MENU) else {
        return;
    };
    if ui.menu_item("Measure") {
        actions.push(GuiShellAction::Inspect(GuiInspectAction::StartMeasure));
    }
    if ui.menu_item("Region Properties") {
        actions.push(GuiShellAction::Inspect(
            GuiInspectAction::OpenRegionProperties,
        ));
    }
}
