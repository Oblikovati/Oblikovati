// SPDX-License-Identifier: GPL-2.0-only
use super::workspace_panels;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, Image, TextureId, Ui};

#[derive(Clone, Copy, Default)]
pub(crate) enum RibbonTab {
    #[default]
    Part,
    Sketch,
    Model,
    View,
}

#[derive(Default)]
pub(crate) struct WorkspaceShellState {
    pub(crate) active_tab: RibbonTab,
    pub(crate) preferences_open: bool,
    pub(crate) selection_open: bool,
    selection_filter: SelectionFilterState,
    icons: crate::icon_assets::GuiIconCache,
}

impl WorkspaceShellState {
    pub(crate) fn textures(&self) -> &[crate::GuiTexture] {
        self.icons.textures()
    }
}

struct SelectionFilterState {
    entries: Vec<SelectionFilterEntry>,
}

struct SelectionFilterEntry {
    label: &'static str,
    enabled: bool,
}

impl Default for SelectionFilterState {
    fn default() -> Self {
        let labels = [
            "Work points",
            "Work axes",
            "Point cloud points",
            "Vertices",
            "Edges",
            "Occurrences",
            "Bodies",
            "Faces",
            "Mesh faces",
            "Work planes",
            "Profiles",
            "Sketch entities",
        ];
        Self {
            entries: labels
                .into_iter()
                .map(|label| SelectionFilterEntry {
                    label,
                    enabled: true,
                })
                .collect(),
        }
    }
}

pub(crate) fn draw(
    ui: &Ui,
    state: &GuiCommandState,
    status: Option<&str>,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut WorkspaceShellState,
) {
    ui.dockspace_over_main_viewport();
    draw_menu_bar(
        ui,
        state,
        selected,
        parameters,
        &mut workspace.preferences_open,
        &mut workspace.selection_open,
    );
    draw_ribbon(ui, state, selected, parameters, workspace);
    workspace_panels::draw_document_tabs(ui, &state.documents, selected);
    workspace_panels::draw_model_browser(ui, state.model_tree.as_ref());
    workspace_panels::draw_command_window(ui, status);
    workspace_panels::draw_status_bar(ui, status);
    draw_preferences(ui, &mut workspace.preferences_open);
    draw_selection_filter(
        ui,
        &mut workspace.selection_open,
        &mut workspace.selection_filter,
    );
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
    workspace: &mut WorkspaceShellState,
) {
    let display = ui.io().display_size;
    ui.window("Ribbon")
        .position([0., 20.], Condition::Always)
        .size([display[0], 92.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            draw_ribbon_tabs(
                ui,
                &mut workspace.active_tab,
                state,
                selected,
                &mut workspace.icons,
            );
            if let Some(tab) = state
                .ribbon_tabs
                .iter()
                .find(|tab| tab.name == ribbon_tab_name(workspace.active_tab))
            {
                draw_host_controls(ui, &tab.controls, selected, &mut workspace.icons);
                return;
            }
            match workspace.active_tab {
                RibbonTab::Part => draw_part_tab(ui, state, selected, &mut workspace.icons),
                RibbonTab::Sketch => draw_sketch_tab(ui, state, selected, &mut workspace.icons),
                RibbonTab::Model => draw_model_tab(ui, state, selected, &mut workspace.icons),
                RibbonTab::View => draw_view_tab(
                    ui,
                    state,
                    selected,
                    parameters,
                    &mut workspace.preferences_open,
                    &mut workspace.selection_open,
                    &mut workspace.icons,
                ),
            }
        });
}

fn draw_ribbon_tabs(
    ui: &Ui,
    active_tab: &mut RibbonTab,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
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
    draw_quick_access(ui, state, selected, icons);
    ui.new_line();
}

fn draw_quick_access(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    for (label, icon, action) in [
        ("New Part", "new-part", GuiShellAction::NewPart),
        ("Save", "save", GuiShellAction::Save),
        ("Undo", "undo", GuiShellAction::Undo),
        ("Redo", "redo", GuiShellAction::Redo),
    ] {
        quick_access_action(ui, label, icon, action, state, selected, icons);
    }
}

fn quick_access_action(
    ui: &Ui,
    label: &str,
    icon: &str,
    action: GuiShellAction,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = icons.texture_id(icon).map_or_else(
        || ui.button(label),
        |id| {
            ui.image_button(
                format!("{label}##quick-access"),
                TextureId::new(id),
                [18., 18.],
            )
        },
    );
    drop(disabled);
    if ui.is_item_hovered() {
        ui.tooltip_text(label);
    }
    if clicked && action.dispatchable(state) {
        selected.push(action);
    }
    ui.same_line();
}

fn draw_part_tab(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    ribbon_action(
        ui,
        "New Part",
        GuiShellAction::NewPart,
        state,
        selected,
        icons,
    );
    ribbon_action(ui, "Save", GuiShellAction::Save, state, selected, icons);
    ribbon_action(ui, "Undo", GuiShellAction::Undo, state, selected, icons);
    ribbon_action(ui, "Redo", GuiShellAction::Redo, state, selected, icons);
}

fn draw_sketch_tab(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    ribbon_action(
        ui,
        "Sketch XY",
        GuiShellAction::SketchXY,
        state,
        selected,
        icons,
    );
    ribbon_action(
        ui,
        "Rectangle",
        GuiShellAction::Rectangle,
        state,
        selected,
        icons,
    );
}

fn draw_model_tab(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    ribbon_action(
        ui,
        "Extrude",
        GuiShellAction::Extrude,
        state,
        selected,
        icons,
    );
}

fn draw_view_tab(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    preferences_open: &mut bool,
    selection_open: &mut bool,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    ribbon_action(ui, "Home", GuiShellAction::Home, state, selected, icons);
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
fn draw_selection_filter(ui: &Ui, open: &mut bool, filter: &mut SelectionFilterState) {
    ui.window("Selection Filter")
        .opened(open)
        .size([300., 360.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Tick to allow picking. Earlier rows have higher priority.");
            ui.separator();
            for entry in &mut filter.entries {
                ui.checkbox(entry.label, &mut entry.enabled);
            }
            ui.separator();
            if ui.button("Select All") {
                filter
                    .entries
                    .iter_mut()
                    .for_each(|entry| entry.enabled = true);
            }
            ui.same_line();
            if ui.button("Deselect All") {
                filter
                    .entries
                    .iter_mut()
                    .for_each(|entry| entry.enabled = false);
            }
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
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let disabled = ui.begin_disabled(!action.enabled(state));
    let clicked = ui.button(label);
    if let Some(id) = icons.texture_id(icon_for_action(action)) {
        ui.same_line();
        Image::new(TextureId::new(id), [18., 18.]).build(ui);
    }
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

fn icon_for_action(action: GuiShellAction) -> &'static str {
    match action {
        GuiShellAction::NewPart => "new-part",
        GuiShellAction::Save => "save",
        GuiShellAction::Undo => "undo",
        GuiShellAction::Redo => "redo",
        GuiShellAction::Home => "home",
        GuiShellAction::SketchXY => "create-sketch",
        GuiShellAction::Rectangle => "rectangle",
        GuiShellAction::Extrude => "extrude",
        GuiShellAction::ActivateDocument(_) => "view-new",
    }
}

fn ribbon_tab_name(tab: RibbonTab) -> &'static str {
    match tab {
        RibbonTab::Part => "Part",
        RibbonTab::Sketch => "Sketch",
        RibbonTab::Model => "3D Model",
        RibbonTab::View => "View",
    }
}
fn draw_host_controls(
    ui: &Ui,
    controls: &[crate::GuiRibbonControl],
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    for control in controls {
        if let Some(action) = host_action(&control.command_id) {
            let disabled = ui.begin_disabled(!control.enabled);
            let clicked = ui.button(&control.display_name);
            if let Some(id) = icons.texture_id(&control.icon) {
                ui.same_line();
                Image::new(TextureId::new(id), [18., 18.]).build(ui);
            }
            drop(disabled);
            if clicked && control.enabled {
                selected.push(action);
            }
            ui.same_line();
        }
    }
}
fn host_action(id: &str) -> Option<GuiShellAction> {
    match id {
        "GetStarted.NewPart" => Some(GuiShellAction::NewPart),
        "File.Save" => Some(GuiShellAction::Save),
        "Sketch.Create" => Some(GuiShellAction::SketchXY),
        "Sketch.Rectangle" => Some(GuiShellAction::Rectangle),
        "Feature.Extrude" => Some(GuiShellAction::Extrude),
        "View.Home" => Some(GuiShellAction::Home),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::SelectionFilterState;

    #[test]
    fn selection_filter_defaults_to_the_host_pick_order() {
        let filter = SelectionFilterState::default();
        assert_eq!(filter.entries.len(), 12);
        assert!(filter.entries.iter().all(|entry| entry.enabled));
        assert_eq!(filter.entries[0].label, "Work points");
        assert_eq!(filter.entries[11].label, "Sketch entities");
    }
}
