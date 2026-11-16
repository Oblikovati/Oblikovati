// SPDX-License-Identifier: GPL-2.0-only
//! The Move tool's current selection and canvas-pick stage.
use crate::dockable_window::DockableWindow;
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
    pub copy: bool,
}
impl DockableWindow for GuiSketchMoveStatus {}
/// Control the current Move command.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchMoveAction {
    Start,
    Done,
    SetOptimizeSingle(bool),
    SetCopy(bool),
    ChooseBase,
}

pub(crate) fn panel(
    ui: &Ui,
    state: &GuiCommandState,
    precise: &mut crate::sketch::precise_input::PreciseInputBar,
    actions: &mut Vec<GuiShellAction>,
) {
    precise.sync_move_context(state.sketch_move.is_some());
    let Some(status) = state.sketch_move else {
        return;
    };
    GuiSketchMoveStatus::fit_dockable(ui, "Move", [1100., 250.], [330., 205.]);
    let mut open = true;
    GuiSketchMoveStatus::dockable_window(ui, "Move", &mut open).build(|| {
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
        let mut copy = status.copy;
        if ui.checkbox("Copy", &mut copy) {
            actions.push(GuiShellAction::SketchMove(GuiSketchMoveAction::SetCopy(
                copy,
            )));
        }
        drop(disabled);
        let mut enabled = precise.move_option();
        if ui.checkbox("Precise Input", &mut enabled) {
            precise.set_move_option(enabled);
        }
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
    if !open {
        actions.push(GuiShellAction::SketchMove(GuiSketchMoveAction::Done));
    }
}
