// SPDX-License-Identifier: GPL-2.0-only
//! Measure, its panel, and the sketch canvas menu that opens the inspection commands.
use crate::dockable_window::DockableWindow;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;

/// What the owner has measured so far, one line per row; example: `Selection 1: Curve`.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiMeasurePanel {
    pub rows: Vec<String>,
}

/// The last sketch command that can be started again from its context menu.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchRepeat {
    Move,
    Placement(crate::GuiSketchTool),
    Constraint(crate::GuiSketchConstraint),
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
const MEASURE_SIZE: [f32; 2] = [300., 300.];

/// Draw the open measurement; example: rows of a selected curve, with Done below them.
pub(crate) fn measure_panel(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    dock: &crate::external_dock_window::ExternalDockWindow,
) {
    if !dock.is_open() {
        return;
    }
    let Some(panel) = &state.measure else {
        return;
    };
    let [width, _] = ui.io().display_size;
    crate::external_dock_window::ExternalDockWindow::fit_dockable(
        ui,
        "Measure",
        [width - MEASURE_SIZE[0] - 24., 220.],
        MEASURE_SIZE,
    );
    let minimum_width = measure_width(ui, panel);
    let mut open = true;
    crate::external_dock_window::ExternalDockWindow::dockable_window(ui, "Measure", &mut open)
        .size_constraints([minimum_width, 1.], [width.max(minimum_width), f32::MAX])
        .collapsible(false)
        .build(|| {
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
    if !open {
        actions.push(GuiShellAction::Inspect(GuiInspectAction::FinishMeasure));
    }
}
fn measure_width(ui: &Ui, panel: &GuiMeasurePanel) -> f32 {
    let padding = ui.clone_style().window_padding[0] * 2.;
    let content = panel
        .rows
        .iter()
        .map(|row| ui.calc_text_size(row)[0])
        .fold(0., f32::max);
    (content + padding)
        .max(MEASURE_SIZE[0])
        .min((ui.io().display_size[0] - 32.).max(1.))
}
/// The empty canvas's menu while a sketch waits at Ready; a running command has its own.
/// What was under the pointer when the canvas menu was asked for.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSketchContext {
    /// Unique owner-generated identity for this context-menu invocation.
    pub token: u64,
    /// The point the menu's Center Point row acts on; none on empty canvas.
    pub point: Option<u64>,
    /// The placed block under the context pointer, when one was hit.
    pub block: Option<oblikovati_model::sketch::SketchEntityId>,
    /// The geometric constraint whose displayed glyph was hit.
    pub glyph: Option<oblikovati_model::sketch::SketchConstraintId>,
    pub center_point: bool,
    /// Every entity within reach; two or more is what Select Other is for.
    pub candidates: Vec<GuiSketchCandidate>,
    /// Geometric relations owned by the curve under the context pointer.
    pub relations: Vec<GuiSketchRelation>,
}
/// One identifiable selectable under the sketch context pointer.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSketchCandidate {
    pub entity: oblikovati_model::sketch::SketchEntityId,
    pub label: String,
}
/// One geometric relation resolved from the picked curve.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSketchRelation {
    pub id: u64,
    pub label: String,
}
fn repeat_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(command) = state.repeat_command else {
        return;
    };
    let (label, action) = match command {
        GuiSketchRepeat::Move => (
            "Move",
            GuiShellAction::SketchMove(crate::GuiSketchMoveAction::Start),
        ),
        GuiSketchRepeat::Placement(tool) => (tool.label(), GuiShellAction::PlaceSketchTool(tool)),
        GuiSketchRepeat::Constraint(choice) => {
            (choice.label(), GuiShellAction::ConstrainSketch(choice))
        }
    };
    if ui
        .menu_item_config(format!("Repeat {label}"))
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}
/// On a point it changes that point; on empty canvas, what new points will be.
fn center_point_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let target = state
        .sketch_context
        .as_ref()
        .and_then(|context| context.point.map(|point| (point, context.center_point)));
    let checked = target.map_or(state.center_point_format, |(_, center)| center);
    if !ui
        .menu_item_config("Center Point")
        .selected(checked)
        .build()
    {
        return;
    }
    actions.push(match target {
        Some((point, center)) => GuiShellAction::SetCenterPoint {
            point,
            center: !center,
        },
        None => GuiShellAction::ToggleCenterPointFormat,
    });
}
fn delete_selection_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let action = GuiShellAction::DeleteSketchSelection;
    if ui
        .menu_item_config("Delete")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}
fn relation_rows(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(context) = &state.sketch_context else {
        return;
    };
    for relation in &context.relations {
        let action = GuiShellAction::DeleteSketchRelation(relation.id);
        if ui
            .menu_item_config(format!(
                "Delete {} Constraint##relation-{}",
                relation.label, relation.id
            ))
            .enabled(action.enabled(state))
            .build()
        {
            actions.push(action);
        }
    }
}
fn select_other_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(context) = state
        .sketch_context
        .as_ref()
        .filter(|context| context.candidates.len() > 1)
    else {
        return;
    };
    ui.menu("Select Other", || {
        for (index, candidate) in context.candidates.iter().enumerate() {
            if ui.menu_item(format!("{}##candidate-{index}", candidate.label)) {
                actions.push(GuiShellAction::SelectSketchContextCandidate {
                    token: context.token,
                    entity: candidate.entity,
                });
            }
        }
    });
}
pub(crate) fn canvas_menu(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    open_token: &mut Option<u64>,
    marking_enabled: bool,
) {
    if state.authoring != GuiAuthoringContext::Sketch || !state.sketch_at_ready {
        retire_closed_context(open_token, actions);
        return;
    }
    let over_panel = ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel());
    if !over_panel && ui.is_mouse_released(imgui::MouseButton::Right) {
        ui.open_popup(CANVAS_MENU);
    }
    let Some(_popup) = ui.begin_popup(CANVAS_MENU) else {
        retire_closed_context(open_token, actions);
        return;
    };
    *open_token = state.sketch_context.as_ref().map(|context| context.token);
    if constraint_glyph_rows(ui, state, actions) {
        return;
    }
    if let Some(menu) = &state.sketch_marking_menu {
        if crate::marking_menu::use_classic(state.sketch_classic_menu, marking_enabled) {
            super::marking_menu::draw_classic(ui, menu, state, actions);
        } else {
            super::marking_menu::draw(ui, menu, state, actions);
        }
    }
    repeat_row(ui, state, actions);
    center_point_row(ui, state, actions);
    explode_block_row(ui, state, actions);
    delete_selection_row(ui, state, actions);
    relation_rows(ui, state, actions);
    select_other_row(ui, state, actions);
    isolation_rows(ui, state, actions);
    if ui.menu_item("Measure") {
        actions.push(GuiShellAction::Inspect(GuiInspectAction::StartMeasure));
    }
    if ui.menu_item("Region Properties") {
        actions.push(GuiShellAction::Inspect(
            GuiInspectAction::OpenRegionProperties,
        ));
    }
}

fn constraint_glyph_rows(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let Some(context) = state
        .sketch_context
        .as_ref()
        .filter(|context| context.glyph.is_some())
    else {
        return false;
    };
    let Some(glyph) = context.glyph else {
        return false;
    };
    if ui.menu_item("Hide") {
        actions.push(GuiShellAction::HideContextConstraintGlyph {
            token: context.token,
            glyph,
        });
    }
    if ui.menu_item("Show All Constraints") {
        actions.push(GuiShellAction::SetContextConstraintGlyphs {
            token: context.token,
            shown: true,
        });
    }
    if ui.menu_item("Hide All Constraints") {
        actions.push(GuiShellAction::SetContextConstraintGlyphs {
            token: context.token,
            shown: false,
        });
    }
    true
}

fn explode_block_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(context) = &state.sketch_context else {
        return;
    };
    let Some(instance) = context.block else {
        return;
    };
    let action = GuiShellAction::ExplodeSketchContext {
        token: context.token,
        instance,
    };
    if ui
        .menu_item_config("Explode")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}

fn isolation_rows(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    for (label, isolated) in [("Hide Others", true), ("Show All", false)] {
        let action = GuiShellAction::SetSketchIsolation(isolated);
        if ui
            .menu_item_config(label)
            .enabled(action.enabled(state))
            .build()
        {
            actions.push(action);
        }
    }
}

fn retire_closed_context(open_token: &mut Option<u64>, actions: &mut Vec<GuiShellAction>) {
    if let Some(token) = open_token.take() {
        actions.push(GuiShellAction::DismissSketchContext { token });
    }
}
