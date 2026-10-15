// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Ui};

#[derive(Clone, Copy, Default)]
pub(crate) enum RibbonTab {
    #[default]
    Part,
    Sketch,
    Model,
    View,
}

#[allow(clippy::too_many_arguments)]
pub(crate) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    status: Option<&str>,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    active_tab: &mut RibbonTab,
    preferences_open: &mut bool,
    selection_open: &mut bool,
) {
    ui.dockspace_over_main_viewport();
    draw_menu_bar(
        ui,
        state,
        selected,
        parameters,
        preferences_open,
        selection_open,
    );
    draw_ribbon(
        ui,
        state,
        selected,
        parameters,
        active_tab,
        preferences_open,
        selection_open,
    );
    draw_model_browser(ui, state.model_tree.as_ref());
    draw_command_window(ui, status);
    draw_preferences(ui, preferences_open);
    draw_selection_filter(ui, selection_open);
}

fn draw_menu_bar(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    preferences_open: &mut bool,
    selection_open: &mut bool,
) {
    ui.main_menu_bar(|| {
        ui.menu("File", || {
            menu_action(ui, "New Part", GuiShellAction::NewPart, state, selected);
            menu_action(ui, "Save", GuiShellAction::Save, state, selected);
        });
        ui.menu("Edit", || {
            menu_action(ui, "Undo", GuiShellAction::Undo, state, selected);
            menu_action(ui, "Redo", GuiShellAction::Redo, state, selected);
        });
        ui.menu("Sketch", || {
            menu_action(
                ui,
                "Start Sketch on XY",
                GuiShellAction::SketchXY,
                state,
                selected,
            );
            menu_action(ui, "Rectangle", GuiShellAction::Rectangle, state, selected);
        });
        ui.menu("3D Model", || {
            menu_action(ui, "Extrude", GuiShellAction::Extrude, state, selected);
        });
        ui.menu("View", || {
            menu_action(ui, "Home", GuiShellAction::Home, state, selected);
            if ui.menu_item("Parameters") {
                parameters.set_open(true);
            }
            if ui.menu_item("Preferences") {
                *preferences_open = true;
            }
            if ui.menu_item("Selection Filter") {
                *selection_open = true;
            }
        });
    });
}

fn draw_ribbon(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    active_tab: &mut RibbonTab,
    preferences_open: &mut bool,
    selection_open: &mut bool,
) {
    ui.window("Ribbon")
        .position([0., 20.], Condition::Always)
        .size([640., 92.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            draw_ribbon_tabs(ui, active_tab);
            match active_tab {
                RibbonTab::Part => draw_part_tab(ui, state, selected),
                RibbonTab::Sketch => draw_sketch_tab(ui, state, selected),
                RibbonTab::Model => draw_model_tab(ui, state, selected),
                RibbonTab::View => draw_view_tab(
                    ui,
                    state,
                    selected,
                    parameters,
                    preferences_open,
                    selection_open,
                ),
            }
        });
}

fn draw_ribbon_tabs(ui: &Ui, active_tab: &mut RibbonTab) {
    for (label, tab) in [
        ("Part##ribbon-tab", RibbonTab::Part),
        ("Sketch##ribbon-tab", RibbonTab::Sketch),
        ("3D Model##ribbon-tab", RibbonTab::Model),
        ("View##ribbon-tab", RibbonTab::View),
    ] {
        if ui.button(label) {
            *active_tab = tab;
        }
        ui.same_line();
    }
    ui.new_line();
}

fn draw_part_tab(ui: &Ui, state: &GuiCommandState, selected: &mut Vec<GuiShellAction>) {
    ribbon_action(ui, "New Part", GuiShellAction::NewPart, state, selected);
    ribbon_action(ui, "Save", GuiShellAction::Save, state, selected);
    ribbon_action(ui, "Undo", GuiShellAction::Undo, state, selected);
    ribbon_action(ui, "Redo", GuiShellAction::Redo, state, selected);
}

fn draw_sketch_tab(ui: &Ui, state: &GuiCommandState, selected: &mut Vec<GuiShellAction>) {
    ribbon_action(ui, "Sketch XY", GuiShellAction::SketchXY, state, selected);
    ribbon_action(ui, "Rectangle", GuiShellAction::Rectangle, state, selected);
}

fn draw_model_tab(ui: &Ui, state: &GuiCommandState, selected: &mut Vec<GuiShellAction>) {
    ribbon_action(ui, "Extrude", GuiShellAction::Extrude, state, selected);
}

fn draw_view_tab(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    preferences_open: &mut bool,
    selection_open: &mut bool,
) {
    ribbon_action(ui, "Home", GuiShellAction::Home, state, selected);
    if ui.button("Parameters") {
        parameters.set_open(true);
    }
    if ui.button("Preferences") {
        *preferences_open = true;
    }
    if ui.button("Selection Filter") {
        *selection_open = true;
    }
}

fn draw_preferences(ui: &Ui, open: &mut bool) {
    ui.window("Preferences")
        .opened(open)
        .size([360., 240.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Appearance");
            ui.separator();
            ui.text(
                "Theme and display settings will use the document-independent desktop preferences.",
            );
        });
}
fn draw_selection_filter(ui: &Ui, open: &mut bool) {
    ui.window("Selection Filter")
        .opened(open)
        .size([260., 220.], Condition::FirstUseEver)
        .build(|| {
            let mut bodies = true;
            let mut faces = true;
            let mut edges = true;
            let mut vertices = true;
            ui.checkbox("Bodies", &mut bodies);
            ui.checkbox("Faces", &mut faces);
            ui.checkbox("Edges", &mut edges);
            ui.checkbox("Vertices", &mut vertices);
        });
}

fn draw_model_browser(ui: &Ui, tree: Option<&crate::GuiModelTree>) {
    ui.window("Model")
        .size([280., 420.], Condition::FirstUseEver)
        .build(|| {
            let Some(tree) = tree else {
                ui.text("No active part");
                return;
            };
            ui.text(&tree.document);
            ui.bullet_text("Origin");
            ui.bullet_text(format!("Parameters ({})", tree.parameters.len()));
            ui.bullet_text(format!("Sketches ({})", tree.sketches));
            ui.bullet_text(format!("Features ({})", tree.features.len()));
            for feature in &tree.features {
                ui.text(format!("  {} {}", feature.name, feature.health));
            }
            ui.bullet_text(format!("Bodies ({})", tree.bodies));
        });
}

fn draw_command_window(ui: &Ui, status: Option<&str>) {
    ui.window("Command")
        .size([360., 130.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Command");
            ui.separator();
            ui.text_wrapped(status.unwrap_or("Ready."));
        });
}

fn menu_action(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
) {
    let item = ui
        .menu_item_config(label)
        .enabled(action.dispatchable(state));
    if item.build() {
        selected.push(action);
    }
}

fn ribbon_action(
    ui: &Ui,
    label: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = ui.button(label);
    drop(disabled);
    if ui.is_item_hovered()
        && let Some(next) = action.next_label(state)
    {
        ui.tooltip_text(format!("{label} {next}"));
    }
    if clicked && action.dispatchable(state) {
        selected.push(action);
    }
    ui.same_line();
}
