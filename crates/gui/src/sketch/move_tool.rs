// SPDX-License-Identifier: GPL-2.0-only
//! The Move tool's current selection and canvas-pick stage.
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;

/// One stage of a canvas move.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchMoveStage {
    Geometry,
    Base,
    Target,
}
/// Current move stage and selection settings.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiSketchMoveStatus {
    pub stage: GuiSketchMoveStage,
    pub selection_count: usize,
    pub optimize_single: bool,
}
/// Control the current Move command.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchMoveAction {
    Start,
    Done,
    SetOptimizeSingle(bool),
    ChooseBase,
}

pub(crate) fn panel(
    ui: &Ui,
    state: &GuiCommandState,
    precise: &mut crate::sketch::precise_input::PreciseInputBar,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(status) = state.sketch_move else {
        return;
    };
    crate::widget_abi::fit_dialog(ui, "Move", [1100., 250.], [330., 180.]);
    ui.window("Move").collapsible(false).build(|| {
        ui.text(match status.stage {
            GuiSketchMoveStage::Geometry => "Select geometry",
            GuiSketchMoveStage::Base => "Select base point",
            GuiSketchMoveStage::Target => "Select end point",
        });
        let mut optimize = status.optimize_single;
        let disabled = ui.begin_disabled(status.stage != GuiSketchMoveStage::Geometry);
        if ui.checkbox("Optimize for Single Selection", &mut optimize) {
            actions.push(GuiShellAction::SketchMove(
                GuiSketchMoveAction::SetOptimizeSingle(optimize),
            ));
        }
        drop(disabled);
        ui.checkbox("Precise Input", &mut precise.open);
        let base = GuiShellAction::SketchMove(GuiSketchMoveAction::ChooseBase);
        let disabled = ui.begin_disabled(!base.enabled(state));
        if ui.button("Base Point") {
            actions.push(base);
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Done") {
            actions.push(GuiShellAction::SketchMove(GuiSketchMoveAction::Done));
        }
    });
}
