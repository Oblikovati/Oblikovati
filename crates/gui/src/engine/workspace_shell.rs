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

pub(crate) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    status: Option<&str>,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    active_tab: &mut RibbonTab,
) {
    draw_menu_bar(ui, state, selected, parameters);
    draw_ribbon(ui, state, selected, parameters, active_tab);
    draw_command_panel(ui, status);
}

fn draw_menu_bar(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
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
        });
    });
}

fn draw_ribbon(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    active_tab: &mut RibbonTab,
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
                RibbonTab::View => draw_view_tab(ui, state, selected, parameters),
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
) {
    ribbon_action(ui, "Home", GuiShellAction::Home, state, selected);
    if ui.button("Parameters") {
        parameters.set_open(true);
    }
}

fn draw_command_panel(ui: &Ui, status: Option<&str>) {
    ui.window("Command Panel")
        .position([8., 120.], Condition::Always)
        .size([280., 130.], Condition::Always)
        .build(|| {
            ui.text("Cube workflow");
            ui.separator();
            ui.text_wrapped("New Part, Sketch XY, Rectangle, then Extrude.");
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
