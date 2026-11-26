// SPDX-License-Identifier: GPL-2.0-only
//! Sketch ribbon commands keep their existing tools across appearance layouts.
use super::*;

use crate::GuiSketchConstraint::{
    Coincident, Collinear, Concentric, Equal, Fix, Horizontal, Parallel, Perpendicular, Tangent,
    Vertical,
};
pub(super) const CONSTRAINT_CONTROLS: [(crate::GuiSketchConstraint, [f32; 2]); 10] = [
    (Horizontal, [0.0, 0.0]),
    (Vertical, [0.0, 22.0]),
    (Coincident, [0.0, 44.0]),
    (Fix, [116.0, 44.0]),
    (Parallel, [232.0, 0.0]),
    (Perpendicular, [232.0, 22.0]),
    (Collinear, [232.0, 44.0]),
    (Equal, [348.0, 0.0]),
    (Concentric, [348.0, 22.0]),
    (Tangent, [348.0, 44.0]),
];
pub(super) fn sketch_constraint_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    if icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
        appearance_sketch::constrain(ui, origin, state, icons, actions);
        return;
    }
    for (choice, position) in CONSTRAINT_CONTROLS {
        icons.cursor(ui, origin, position);
        icons.command_button(
            ui,
            choice.label(),
            GuiShellAction::ConstrainSketch(choice),
            state,
            [108., 20.],
            actions,
        );
    }
    icons.cursor(ui, origin, [116., 0.]);
    icons.command_button(
        ui,
        "Dimension",
        GuiShellAction::Dimension(crate::GuiDimensionAction::Start),
        state,
        [108., 42.],
        actions,
    );
}
pub(super) fn sketch_create_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    if icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
        appearance_sketch::create(ui, origin, state, icons, actions);
        return;
    }
    crate::widget_abi::cursor_position(ui, origin);
    let rectangle = crate::GuiSketchTool::Rectangle;
    icons.command_button(
        ui,
        rectangle.label(),
        GuiShellAction::PlaceSketchTool(rectangle),
        state,
        icons.appearance.single_cell(),
        actions,
    );
    let secondary = crate::sketch_tools::SECONDARY_SKETCH_TOOLS;
    for (index, tool) in secondary.into_iter().enumerate() {
        icons.cursor(
            ui,
            origin,
            [124. + (index % 2) as f32 * 116., (index / 2) as f32 * 22.],
        );
        crate::sketch_flyouts::sketch_tool_flyout_button(ui, state, actions, tool, icons);
    }
    icons.cursor(ui, origin, [356., 0.]);
    icons.command_button(
        ui,
        "Fillet",
        GuiShellAction::PlaceSketchTool(crate::GuiSketchTool::Fillet),
        state,
        [108., 20.],
        actions,
    );
    icons.cursor(ui, origin, [356., 22.]);
    icons.command_button(
        ui,
        "Text",
        GuiShellAction::PlaceSketchTool(crate::GuiSketchTool::Text),
        state,
        [108., 20.],
        actions,
    );
    icons.cursor(ui, origin, [356., 44.]);
    crate::sketch_flyouts::sketch_tool_flyout_button(
        ui,
        state,
        actions,
        crate::GuiSketchTool::Line,
        icons,
    );
    icons.cursor(ui, origin, [472., 0.]);
    let disabled = ui.begin_disabled(true);
    icons.button(
        ui,
        "Project Geometry",
        crate::icons::IconGlyph::ProjectGeometry,
        [150., icons.appearance.command_height()],
    );
    drop(disabled);
}

/// The View checklist and the existing command to show or hide every sketch.
pub(super) fn visibility_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    owners: &mut crate::ribbon_panels::RibbonPanelOwners<'_>,
    actions: &mut Vec<GuiShellAction>,
) {
    let icons = owners.icons;
    let (_, mut size) = icons.appearance.cell(0);
    if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
        size[0] = 140.;
    }
    let disabled = ui.begin_disabled(state.document.is_none());
    owners.workspace.pending_layout |= owners.workspace.object_visibility.ribbon(ui, icons, size);
    drop(disabled);
    let (position, size) = icons.appearance.cell(1);
    icons.cursor(ui, origin, position);
    sketch_visibility(ui, state, icons, size, actions);
}

fn sketch_visibility(
    ui: &Ui,
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    size: [f32; 2],
    actions: &mut Vec<GuiShellAction>,
) {
    let sketches = state
        .browser
        .as_ref()
        .map_or(&[][..], |browser| &browser.sketches);
    let hide = sketches.iter().any(|row| row.visible);
    let disabled = ui.begin_disabled(sketches.is_empty() || state.transaction_open);
    let label = if hide {
        "Hide Sketches"
    } else {
        "Show Sketches"
    };
    let pressed = icons.button(ui, label, crate::icons::IconGlyph::SketchVisibility, size);
    drop(disabled);
    let Some(document) = state.document.filter(|_| pressed) else {
        return;
    };
    actions.extend(
        sketches
            .iter()
            .map(|row| GuiShellAction::SetSketchVisibility {
                document,
                sketch: row.id,
                visible: !hide,
            }),
    );
}
