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
/// Start or finish the current Move command.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchMoveAction {
    Start,
    Done,
}

pub(crate) fn panel(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(stage) = state.sketch_move else {
        return;
    };
    crate::widget_abi::fit_dialog(ui, "Move", [1100., 250.], [260., 110.]);
    ui.window("Move").collapsible(false).build(|| {
        ui.text(match stage {
            GuiSketchMoveStage::Geometry => "Select geometry",
            GuiSketchMoveStage::Base => "Select base point",
            GuiSketchMoveStage::Target => "Select end point",
        });
        if ui.button("Done") {
            actions.push(GuiShellAction::SketchMove(GuiSketchMoveAction::Done));
        }
    });
}
