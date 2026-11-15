// SPDX-License-Identifier: GPL-2.0-only
//! Sketch commands in the shared appearance grid keep their existing actions and flyouts.
use super::*;

pub(super) fn create(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    sketch_tool(
        ui,
        origin,
        state,
        icons,
        actions,
        0,
        crate::GuiSketchTool::Rectangle,
    );
    for (index, tool) in crate::sketch_tools::SECONDARY_SKETCH_TOOLS
        .into_iter()
        .enumerate()
    {
        let (position, _) = icons.appearance.flyout_cell(index + 1);
        icons.cursor(ui, origin, position);
        crate::sketch_flyouts::circular_tool_button(ui, state, actions, tool, icons);
    }
    sketch_tool(
        ui,
        origin,
        state,
        icons,
        actions,
        7,
        crate::GuiSketchTool::Line,
    );
}
fn sketch_tool(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
    index: usize,
    tool: crate::GuiSketchTool,
) {
    let (position, size) = icons.appearance.flyout_cell(index);
    icons.cursor(ui, origin, position);
    icons.command_button(
        ui,
        tool.label(),
        GuiShellAction::PlaceSketchTool(tool),
        state,
        size,
        actions,
    );
}
pub(super) fn constrain(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    for (index, (choice, _)) in CONSTRAINT_CONTROLS.into_iter().enumerate() {
        let (position, size) = icons.appearance.cell(index);
        icons.cursor(ui, origin, position);
        icons.command_button(
            ui,
            choice.label(),
            GuiShellAction::ConstrainSketch(choice),
            state,
            size,
            actions,
        );
    }
    let (position, size) = icons.appearance.cell(CONSTRAINT_CONTROLS.len());
    icons.cursor(ui, origin, position);
    icons.command_button(
        ui,
        "Dimension",
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
        state,
        size,
        actions,
    );
}
