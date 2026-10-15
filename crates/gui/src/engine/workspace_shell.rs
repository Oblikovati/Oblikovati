// SPDX-License-Identifier: GPL-2.0-only
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
    bodies: bool,
    faces: bool,
    edges: bool,
    vertices: bool,
}

impl Default for SelectionFilterState {
    fn default() -> Self {
        Self {
            bodies: true,
            faces: true,
            edges: true,
            vertices: true,
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
    draw_document_tabs(ui, &state.documents, selected);
    draw_model_browser(ui, state.model_tree.as_ref());
    draw_command_window(ui, status);
    draw_status_bar(ui, status);
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

fn draw_status_bar(ui: &Ui, status: Option<&str>) {
    let display = ui.io().display_size;
    ui.window("Status")
        .position([0., display[1] - 28.], Condition::Always)
        .size([display[0], 28.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| ui.text(status.unwrap_or("Press F1 for more help")));
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
        .size([260., 220.], Condition::FirstUseEver)
        .build(|| {
            ui.checkbox("Bodies", &mut filter.bodies);
            ui.checkbox("Faces", &mut filter.faces);
            ui.checkbox("Edges", &mut filter.edges);
            ui.checkbox("Vertices", &mut filter.vertices);
        });
}

fn draw_document_tabs(
    ui: &Ui,
    documents: &[crate::GuiDocumentTab],
    selected: &mut Vec<GuiShellAction>,
) {
    if documents.is_empty() {
        return;
    }
    let display = ui.io().display_size;
    ui.window("Documents")
        .position([0., 112.], Condition::Always)
        .size([display[0], 30.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            for document in documents {
                let label = if document.dirty {
                    format!("{} *##document-{}", document.name, document.id)
                } else {
                    format!("{}##document-{}", document.name, document.id)
                };
                if ui
                    .selectable_config(&label)
                    .selected(document.active)
                    .build()
                    && !document.active
                {
                    selected.push(GuiShellAction::ActivateDocument(document.id));
                }
                ui.same_line();
            }
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
                ui.text(format!(
                    "  {} ({}) {}",
                    feature.name, feature.kind, feature.health
                ));
            }
            ui.bullet_text(format!("Bodies ({})", tree.bodies.len()));
            for body in &tree.bodies {
                let visibility = if body.visible { "visible" } else { "hidden" };
                ui.text(format!(
                    "  {} — {} faces, {}",
                    body.name, body.faces, visibility
                ));
            }
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
