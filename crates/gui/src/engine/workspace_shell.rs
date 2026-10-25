// SPDX-License-Identifier: GPL-2.0-only
use super::workspace_panels;
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Condition, TabItemFlags, TextureId, Ui};
use std::collections::BTreeMap;

// Go's live 16 px menu uses the default 3 px vertical frame padding, plus its border.
pub(super) const MENU_BAR_HEIGHT: f32 = 22.;
// Go computes this from its 16 px Liberation Sans font and stock Dear ImGui metrics:
// 2*8 window padding + 22 tab strip + 4 gap + 108 control grid + 4 gap + 16 caption.
const RIBBON_HEIGHT: f32 = 170.;
const RIBBON_PANEL_DIVIDER_HEIGHT: f32 = 108.;
pub(super) const RIBBON_BOTTOM: f32 = MENU_BAR_HEIGHT + RIBBON_HEIGHT;

pub(crate) struct WorkspaceShellState {
    pub(crate) active_tab: String,
    force_active_tab: bool,
    active_ribbon_key: String,
    contextual_sketch_tab_visible: bool,
    pub(crate) preferences_open: bool,
    pub(crate) selection_open: bool,
    pub(crate) history_open: bool,
    pub(crate) units_open: bool,
    pub(crate) model_browser_open: bool,
    pub(crate) command_panel_open: bool,
    pub(super) command_input: String,
    pub(super) command_focus_next: bool,
    history_documents: Vec<u64>,
    ribbon_panel_widths: BTreeMap<String, f32>,
    control_targets: Vec<crate::GuiControlTarget>,
    icons: crate::icon_assets::GuiIconCache,
}

impl Default for WorkspaceShellState {
    fn default() -> Self {
        Self {
            active_tab: String::new(),
            force_active_tab: false,
            active_ribbon_key: String::new(),
            contextual_sketch_tab_visible: false,
            preferences_open: false,
            selection_open: false,
            history_open: false,
            units_open: false,
            model_browser_open: true,
            command_panel_open: true,
            command_input: String::new(),
            command_focus_next: true,
            history_documents: Vec::new(),
            ribbon_panel_widths: BTreeMap::new(),
            control_targets: Vec::new(),
            icons: Default::default(),
        }
    }
}
impl WorkspaceShellState {
    pub(crate) fn textures(&self) -> &[crate::GuiTexture] {
        self.icons.textures()
    }
    pub(crate) fn control_targets(&self) -> Vec<crate::GuiControlTarget> {
        self.control_targets.clone()
    }
    fn begin_frame(&mut self) {
        self.control_targets.clear();
    }
    fn record_control(&mut self, control: &crate::GuiRibbonControl, min: [f32; 2], max: [f32; 2]) {
        self.record_target(&control.command_id, control.enabled, min, max);
    }
    pub(super) fn record_target(
        &mut self,
        command_id: &str,
        enabled: bool,
        min: [f32; 2],
        max: [f32; 2],
    ) {
        self.control_targets.push(crate::GuiControlTarget {
            command_id: command_id.into(),
            center: [(min[0] + max[0]) / 2., (min[1] + max[1]) / 2.],
            enabled,
        });
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
    workspace.begin_frame();
    ui.dockspace_over_main_viewport();
    draw_menu_bar(ui, state, selected, parameters, workspace);
    draw_ribbon(ui, state, selected, parameters, workspace);
    workspace_panels::draw_viewport_header(ui, !state.documents.is_empty());
    workspace_panels::draw_document_tabs(ui, &state.documents, selected, workspace);
    if workspace.model_browser_open {
        workspace_panels::draw_model_browser(
            ui,
            state.model_tree.as_ref(),
            selected,
            &mut workspace.model_browser_open,
        );
    }
    if workspace.command_panel_open {
        workspace_panels::draw_command_window(ui, workspace);
    }
    let mut icons = std::mem::take(&mut workspace.icons);
    workspace_panels::draw_viewport_overlays(
        ui,
        !state.documents.is_empty(),
        workspace.model_browser_open,
        selected,
        &mut icons,
        workspace,
    );
    workspace.icons = icons;
    workspace_panels::draw_status_bar(ui, status);
    draw_preferences(
        ui,
        &mut workspace.preferences_open,
        state.sketch_preferences.as_ref(),
        selected,
    );
    draw_selection_filter(
        ui,
        &mut workspace.selection_open,
        state.selection_filter.as_ref(),
        selected,
    );
    draw_history_browser(
        ui,
        &mut workspace.history_open,
        &state.histories,
        &mut workspace.history_documents,
        selected,
    );
    draw_units_settings(
        ui,
        &mut workspace.units_open,
        state.document_units.as_ref(),
        selected,
    );
}

fn draw_menu_bar(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut WorkspaceShellState,
) {
    ui.main_menu_bar(|| {
        ui.menu("File", || {
            menu_action(ui, "New Part", GuiShellAction::NewPart, state, selected);
            menu_action(
                ui,
                "New Assembly",
                GuiShellAction::NewAssembly,
                state,
                selected,
            );
            menu_action(
                ui,
                "New Drawing",
                GuiShellAction::NewDrawing,
                state,
                selected,
            );
            menu_action(ui, "Save", GuiShellAction::Save, state, selected);
        });
        ui.menu("Edit", || {
            menu_action(ui, "Undo", GuiShellAction::Undo, state, selected);
            menu_action(ui, "Redo", GuiShellAction::Redo, state, selected);
        });
        ui.menu("View", || {
            menu_action(ui, "Home", GuiShellAction::Home, state, selected);
            menu_action(
                ui,
                "Previous View",
                GuiShellAction::PreviousView,
                state,
                selected,
            );
            if ui.menu_item("Parameters") {
                parameters.set_open(true);
            }
            if ui
                .menu_item_config("Model Browser")
                .selected(workspace.model_browser_open)
                .build()
            {
                workspace.model_browser_open = !workspace.model_browser_open;
            }
            if ui
                .menu_item_config("Command Panel")
                .selected(workspace.command_panel_open)
                .build()
            {
                workspace.command_panel_open = !workspace.command_panel_open;
            }
            if ui.menu_item("History Browser") {
                workspace.history_open = true;
            }
            if ui.menu_item("Document Settings — Units") {
                workspace.units_open = true;
            }
        });
        ui.menu("Tools", || {
            if ui.menu_item("Preferences") {
                workspace.preferences_open = true;
            }
            if ui.menu_item("Selection Filter") {
                workspace.selection_open = true;
            }
        });
        ui.menu("Help", || {
            ui.menu_item_config("Documentation").enabled(false).build();
            ui.separator();
            ui.menu_item_config("Check for Updates")
                .enabled(false)
                .build();
            ui.menu_item_config("Report Bug").enabled(false).build();
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
    select_available_ribbon_tab(state, workspace);
    let display = ui.io().display_size;
    ui.window("Ribbon")
        .position([0., MENU_BAR_HEIGHT], Condition::Always)
        .size([display[0], RIBBON_HEIGHT], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            let mut icons = std::mem::take(&mut workspace.icons);
            draw_ribbon_content(ui, state, selected, parameters, workspace, &mut icons);
            workspace.icons = icons;
        });
}

fn select_available_ribbon_tab(state: &GuiCommandState, workspace: &mut WorkspaceShellState) {
    select_document_ribbon(state, workspace);
    select_contextual_ribbon_tab(state, workspace);
    if state
        .ribbon_tabs
        .iter()
        .any(|tab| tab.name == workspace.active_tab)
    {
        return;
    }
    workspace.active_tab = state
        .ribbon_tabs
        .first()
        .map(|tab| tab.name.clone())
        .unwrap_or_default();
    workspace.force_active_tab = true;
}

fn select_document_ribbon(state: &GuiCommandState, workspace: &mut WorkspaceShellState) {
    if workspace.active_ribbon_key == state.ribbon_key {
        return;
    }
    workspace.active_ribbon_key.clone_from(&state.ribbon_key);
    workspace.active_tab = state
        .ribbon_tabs
        .first()
        .map(|tab| tab.name.clone())
        .unwrap_or_default();
    workspace.force_active_tab = true;
}

fn select_contextual_ribbon_tab(state: &GuiCommandState, workspace: &mut WorkspaceShellState) {
    let sketch_visible = state.ribbon_tabs.iter().any(|tab| tab.name == "Sketch");
    if sketch_visible && !workspace.contextual_sketch_tab_visible {
        workspace.active_tab = "Sketch".into();
        workspace.force_active_tab = true;
    }
    if !sketch_visible && workspace.contextual_sketch_tab_visible {
        workspace.active_tab = "Create & Modify".into();
        workspace.force_active_tab = true;
    }
    workspace.contextual_sketch_tab_visible = sketch_visible;
}

fn draw_ribbon_content(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut WorkspaceShellState,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    draw_quick_access(ui, state, selected, icons, workspace);
    let Some(_tabs) = ui.tab_bar("##ribbon-tabs") else {
        return;
    };
    for tab in &state.ribbon_tabs {
        draw_ribbon_tab(ui, tab, selected, parameters, workspace, icons);
    }
}

fn draw_ribbon_tab(
    ui: &Ui,
    tab: &crate::GuiRibbonTab,
    selected: &mut Vec<GuiShellAction>,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut WorkspaceShellState,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let label = format!("{}##ribbon-tab", tab.name);
    let flags = if workspace.force_active_tab && workspace.active_tab == tab.name {
        TabItemFlags::SET_SELECTED
    } else {
        TabItemFlags::empty()
    };
    let Some(_item) = ui.tab_item_with_flags(label, None, flags) else {
        return;
    };
    if workspace.force_active_tab && workspace.active_tab != tab.name {
        return;
    }
    workspace.active_tab = tab.name.clone();
    workspace.force_active_tab = false;
    draw_host_panels(ui, &tab.name, &tab.panels, selected, icons, workspace);
    if tab.name == "View" {
        draw_view_window_controls(ui, parameters, workspace);
    }
}

fn draw_quick_access(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    draw_application_menu(ui, state, selected, icons);
    for button in quick_access_buttons() {
        if button.label == "Open" {
            draw_disabled_quick_access(ui, button.label, button.icon, icons);
            continue;
        }
        quick_access_action(ui, &button, state, selected, icons, workspace);
    }
}

struct QuickAccessButton {
    label: &'static str,
    icon: &'static str,
    action: GuiShellAction,
}

fn quick_access_buttons() -> [QuickAccessButton; 5] {
    [
        QuickAccessButton {
            label: "New Part",
            icon: "new-part",
            action: GuiShellAction::NewPart,
        },
        QuickAccessButton {
            label: "Open",
            icon: "open",
            action: GuiShellAction::Save,
        },
        QuickAccessButton {
            label: "Save",
            icon: "save",
            action: GuiShellAction::Save,
        },
        QuickAccessButton {
            label: "Undo",
            icon: "undo",
            action: GuiShellAction::Undo,
        },
        QuickAccessButton {
            label: "Redo",
            icon: "redo",
            action: GuiShellAction::Redo,
        },
    ]
}

fn draw_application_menu(
    ui: &Ui,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let clicked = icons.texture_id("file-menu").map_or_else(
        || ui.button("File##app-menu"),
        |id| ui.image_button("##app-menu", TextureId::new(id), [18., 18.]),
    );
    if clicked {
        ui.open_popup("##app-menu-popup");
    }
    if let Some(_popup) = ui.begin_popup("##app-menu-popup") {
        menu_action(ui, "New Part", GuiShellAction::NewPart, state, selected);
        menu_action(
            ui,
            "New Assembly",
            GuiShellAction::NewAssembly,
            state,
            selected,
        );
        menu_action(
            ui,
            "New Drawing",
            GuiShellAction::NewDrawing,
            state,
            selected,
        );
        ui.menu_item_config("Open").enabled(false).build();
        menu_action(ui, "Save", GuiShellAction::Save, state, selected);
    }
    ui.same_line();
}

fn draw_disabled_quick_access(
    ui: &Ui,
    label: &str,
    icon: &str,
    icons: &mut crate::icon_assets::GuiIconCache,
) {
    let disabled = ui.begin_disabled(true);
    icons.texture_id(icon).map_or_else(
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
    ui.same_line();
}

fn quick_access_action(
    ui: &Ui,
    button: &QuickAccessButton,
    state: &GuiCommandState,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    let disabled = ui.begin_disabled(!button.action.enabled(state));
    let clicked = icons.texture_id(button.icon).map_or_else(
        || ui.button(button.label),
        |id| {
            ui.image_button(
                format!("{}##quick-access", button.label),
                TextureId::new(id),
                [18., 18.],
            )
        },
    );
    drop(disabled);
    workspace.record_target(
        &format!("QuickAccess.{}", button.label),
        button.action.enabled(state),
        ui.item_rect_min(),
        ui.item_rect_max(),
    );
    if ui.is_item_hovered() {
        ui.tooltip_text(button.label);
    }
    if clicked && button.action.dispatchable(state) {
        selected.push(button.action.clone());
    }
    ui.same_line();
}

fn draw_view_window_controls(
    ui: &Ui,
    parameters: &mut crate::parameter_panel::ParameterPanel,
    workspace: &mut WorkspaceShellState,
) {
    if ui.button("Parameters") {
        parameters.set_open(true);
    }
    if ui.button("Preferences") {
        workspace.preferences_open = true;
    }
    if ui.button("Selection Filter") {
        workspace.selection_open = true;
    }
    if ui.button("History Browser") {
        workspace.history_open = true;
    }
    if ui.button("Document Units") {
        workspace.units_open = true;
    }
}

fn draw_preferences(
    ui: &Ui,
    open: &mut bool,
    preferences: Option<&crate::GuiSketchPreferences>,
    selected: &mut Vec<GuiShellAction>,
) {
    if !*open {
        return;
    }
    ui.window("Preferences")
        .opened(open)
        .size([640., 480.], Condition::FirstUseEver)
        .build(|| {
            if let Some(_tabs) = ui.tab_bar("##preferences-tabs") {
                preference_tab(ui, "General", "On startup", "New part");
                preference_tab(
                    ui,
                    "UI",
                    "Interface scale",
                    "Desktop UI settings use the application preference service.",
                );
                preference_tab(
                    ui,
                    "Privacy",
                    "Anonymous usage statistics",
                    "No personal data is collected.",
                );
                draw_sketch_grid_tab(ui, preferences, selected);
                preference_tab(
                    ui,
                    "Modeling",
                    "Chamfer corners",
                    "Modeling defaults apply to new features.",
                );
                preference_tab(
                    ui,
                    "Theme",
                    "Appearance",
                    "Theme selection uses the application preference service.",
                );
            }
        });
}

fn draw_sketch_grid_tab(
    ui: &Ui,
    preferences: Option<&crate::GuiSketchPreferences>,
    selected: &mut Vec<GuiShellAction>,
) {
    let Some(_tab) = ui.tab_item("Sketch Grid") else {
        return;
    };
    let Some(preferences) = preferences else {
        ui.text("Sketch preferences are unavailable.");
        return;
    };
    ui.text("Sketch grid");
    ui.separator();
    edit_grid_spacing(ui, preferences.grid_spacing_cm, selected);
    edit_grid_major_every(ui, preferences.grid_major_every, selected);
    preference_checkbox(
        ui,
        "Show grid in sketch",
        preferences.grid_visible,
        GuiShellAction::SetGridVisible,
        selected,
    );
    ui.separator();
    preference_checkbox(
        ui,
        "Heads-up dynamic input while sketching",
        preferences.heads_up_display,
        GuiShellAction::SetHeadsUpDisplay,
        selected,
    );
    preference_checkbox(
        ui,
        "Pointer input",
        preferences.pointer_input,
        GuiShellAction::SetPointerInput,
        selected,
    );
    preference_checkbox(
        ui,
        "Dimension input",
        preferences.dimension_input,
        GuiShellAction::SetDimensionInput,
        selected,
    );
    preference_checkbox(
        ui,
        "Snap to points",
        preferences.snap_to_points,
        GuiShellAction::SetSnapToPoints,
        selected,
    );
    preference_checkbox(
        ui,
        "Snap to grid",
        preferences.snap_to_grid,
        GuiShellAction::SetSnapToGrid,
        selected,
    );
    preference_checkbox(
        ui,
        "Relax mode",
        preferences.relax_mode,
        GuiShellAction::SetRelaxMode,
        selected,
    );
    preference_checkbox(
        ui,
        "Auto-project origin",
        preferences.auto_project_origin,
        GuiShellAction::SetAutoProjectOrigin,
        selected,
    );
    preference_checkbox(
        ui,
        "Suppress format overrides",
        preferences.suppress_format_overrides,
        GuiShellAction::SetSuppressFormatOverrides,
        selected,
    );
}

fn edit_grid_spacing(ui: &Ui, current: f64, selected: &mut Vec<GuiShellAction>) {
    let mut spacing = current as f32;
    if ui.input_float("Grid spacing (cm)", &mut spacing).build() {
        let spacing = f64::from(spacing);
        if spacing.is_finite() && spacing > 0. {
            selected.push(GuiShellAction::SetGridSpacingCm(spacing));
        }
    }
}

fn edit_grid_major_every(ui: &Ui, current: i64, selected: &mut Vec<GuiShellAction>) {
    let Ok(mut major) = i32::try_from(current) else {
        ui.text("Major line interval is outside the desktop input range.");
        return;
    };
    if ui.input_int("Major line every", &mut major).build() && major >= 1 {
        selected.push(GuiShellAction::SetGridMajorEvery(i64::from(major)));
    }
}

fn preference_checkbox(
    ui: &Ui,
    label: &str,
    value: bool,
    action: impl FnOnce(bool) -> GuiShellAction,
    selected: &mut Vec<GuiShellAction>,
) {
    let mut edited = value;
    if ui.checkbox(label, &mut edited) {
        selected.push(action(edited));
    }
}

fn preference_tab(ui: &Ui, name: &str, heading: &str, detail: &str) {
    let Some(_tab) = ui.tab_item(name) else {
        return;
    };
    ui.text(heading);
    ui.separator();
    ui.text_wrapped(detail);
}

fn draw_selection_filter(
    ui: &Ui,
    open: &mut bool,
    filter: Option<&crate::GuiSelectionFilter>,
    selected: &mut Vec<GuiShellAction>,
) {
    if !*open {
        return;
    }
    ui.window("Selection Filter")
        .opened(open)
        .size([300., 360.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Tick to allow picking. Earlier rows have higher priority.");
            ui.separator();
            let Some(filter) = filter else {
                ui.text("No active document");
                return;
            };
            for (index, entry) in filter.entries.iter().enumerate() {
                draw_selection_filter_entry(ui, entry, index, selected);
            }
            ui.separator();
            if ui.button("Select All") {
                for entry in &filter.entries {
                    selected.push(GuiShellAction::SetSelectionEnabled {
                        kind: entry.kind.clone(),
                        enabled: true,
                    });
                }
            }
            ui.same_line();
            if ui.button("Deselect All") {
                for entry in &filter.entries {
                    selected.push(GuiShellAction::SetSelectionEnabled {
                        kind: entry.kind.clone(),
                        enabled: false,
                    });
                }
            }
        });
}

fn draw_selection_filter_entry(
    ui: &Ui,
    entry: &crate::GuiSelectionFilterEntry,
    index: usize,
    selected: &mut Vec<GuiShellAction>,
) {
    let mut enabled = entry.enabled;
    if ui.checkbox(&entry.label, &mut enabled) {
        selected.push(GuiShellAction::SetSelectionEnabled {
            kind: entry.kind.clone(),
            enabled,
        });
    }
    ui.same_line();
    let move_up = ui.small_button(format!("↑##selection-up-{index}"));
    ui.same_line();
    let move_down = ui.small_button(format!("↓##selection-down-{index}"));
    if move_up {
        selected.push(GuiShellAction::MoveSelectionKind {
            from: index,
            to: index - 1,
        });
    }
    if move_down {
        selected.push(GuiShellAction::MoveSelectionKind {
            from: index,
            to: index + 1,
        });
    }
}

fn draw_history_browser(
    ui: &Ui,
    open: &mut bool,
    histories: &[crate::GuiDocumentHistory],
    selected_documents: &mut Vec<u64>,
    selected: &mut Vec<GuiShellAction>,
) {
    if !*open {
        return;
    }
    sync_history_documents(histories, selected_documents);
    ui.window("History Browser")
        .opened(open)
        .size([560., 480.], Condition::FirstUseEver)
        .build(|| draw_history_browser_body(ui, histories, selected_documents, selected));
}

fn draw_history_browser_body(
    ui: &Ui,
    histories: &[crate::GuiDocumentHistory],
    selected_documents: &mut Vec<u64>,
    selected: &mut Vec<GuiShellAction>,
) {
    draw_history_document_picker(ui, histories, selected_documents);
    ui.separator();
    draw_history_timelines(ui, histories, selected_documents, selected);
}

fn sync_history_documents(histories: &[crate::GuiDocumentHistory], selected: &mut Vec<u64>) {
    selected.retain(|id| histories.iter().any(|history| history.document == *id));
    if selected.is_empty()
        && let Some(history) = histories.first()
    {
        selected.push(history.document);
    }
}

fn draw_history_document_picker(
    ui: &Ui,
    histories: &[crate::GuiDocumentHistory],
    selected: &mut Vec<u64>,
) {
    for history in histories {
        let mut shown = selected.contains(&history.document);
        if ui.checkbox(
            format!("{}##history-{}", history.name, history.document),
            &mut shown,
        ) {
            set_history_document(selected, history.document, shown);
        }
    }
}

fn set_history_document(selected: &mut Vec<u64>, document: u64, shown: bool) {
    selected.retain(|id| *id != document);
    if shown {
        selected.push(document);
    }
}

fn draw_history_timelines(
    ui: &Ui,
    histories: &[crate::GuiDocumentHistory],
    selected_documents: &[u64],
    selected: &mut Vec<GuiShellAction>,
) {
    let mut shown = false;
    for history in histories
        .iter()
        .filter(|history| selected_documents.contains(&history.document))
    {
        shown = true;
        draw_history_timeline(ui, history, selected);
    }
    if !shown {
        ui.text("Select a document timeline.");
    }
}

fn draw_history_timeline(
    ui: &Ui,
    history: &crate::GuiDocumentHistory,
    selected: &mut Vec<GuiShellAction>,
) {
    ui.text(&history.name);
    draw_history_position(ui, history, "Open", 0, selected);
    for (index, entry) in history.entries.iter().enumerate() {
        let position = i64::try_from(index + 1).unwrap_or(i64::MAX);
        draw_history_position(ui, history, &history_entry_label(entry), position, selected);
    }
    ui.separator();
}

fn draw_history_position(
    ui: &Ui,
    history: &crate::GuiDocumentHistory,
    label: &str,
    position: i64,
    selected: &mut Vec<GuiShellAction>,
) {
    if history_row(
        ui,
        label,
        history.document,
        position,
        history.position == position,
    ) {
        selected.push(GuiShellAction::JumpHistoryTo {
            document: history.document,
            position,
        });
    }
}

fn history_row(ui: &Ui, label: &str, document: u64, position: i64, current: bool) -> bool {
    ui.selectable_config(format!("{label}##history-{document}-{position}"))
        .selected(current)
        .build()
}

fn history_entry_label(entry: &crate::GuiHistoryEntry) -> String {
    let saved = if entry.saved { "* " } else { "  " };
    format!("{saved}{}", entry.label)
}

fn draw_units_settings(
    ui: &Ui,
    open: &mut bool,
    units: Option<&crate::GuiDocumentUnits>,
    selected: &mut Vec<GuiShellAction>,
) {
    if !*open {
        return;
    }
    ui.window("Document Settings — Units")
        .opened(open)
        .size([340., 360.], Condition::FirstUseEver)
        .build(|| draw_units_settings_body(ui, units, selected));
}

fn draw_units_settings_body(
    ui: &Ui,
    units: Option<&crate::GuiDocumentUnits>,
    selected: &mut Vec<GuiShellAction>,
) {
    let Some(units) = units else {
        ui.text("Document units are unavailable.");
        return;
    };
    draw_unit_pickers(ui, units, selected);
    ui.separator();
    draw_unit_precision(ui, units, selected);
}

fn draw_unit_pickers(ui: &Ui, units: &crate::GuiDocumentUnits, selected: &mut Vec<GuiShellAction>) {
    unit_picker(
        ui,
        "Length",
        &units.length_unit,
        &["µm", "mm", "cm", "m", "in", "ft"],
        GuiShellAction::SetLengthUnit,
        selected,
    );
    unit_picker(
        ui,
        "Angle",
        &units.angle_unit,
        &["deg", "rad"],
        GuiShellAction::SetAngleUnit,
        selected,
    );
    unit_picker(
        ui,
        "Mass",
        &units.mass_unit,
        &["kg", "g", "lb"],
        GuiShellAction::SetMassUnit,
        selected,
    );
    unit_picker(
        ui,
        "Time",
        &units.time_unit,
        &["s", "ms", "min", "hr"],
        GuiShellAction::SetTimeUnit,
        selected,
    );
}

fn unit_picker(
    ui: &Ui,
    label: &str,
    current: &str,
    options: &[&str],
    action: impl Fn(String) -> GuiShellAction,
    selected: &mut Vec<GuiShellAction>,
) {
    let Some(_combo) = ui.begin_combo(label, current) else {
        return;
    };
    for option in options {
        if ui
            .selectable_config(option)
            .selected(*option == current)
            .build()
        {
            selected.push(action((*option).into()));
        }
    }
}

fn draw_unit_precision(
    ui: &Ui,
    units: &crate::GuiDocumentUnits,
    selected: &mut Vec<GuiShellAction>,
) {
    edit_precision(
        ui,
        "Length precision",
        units.length_precision,
        GuiShellAction::SetLengthPrecision,
        selected,
    );
    edit_precision(
        ui,
        "Angle precision",
        units.angle_precision,
        GuiShellAction::SetAnglePrecision,
        selected,
    );
    unit_picker(
        ui,
        "Length format",
        &units.length_format,
        &["decimal", "fractional", "architectural"],
        GuiShellAction::SetLengthFormat,
        selected,
    );
}

fn edit_precision(
    ui: &Ui,
    label: &str,
    current: i64,
    action: impl Fn(i64) -> GuiShellAction,
    selected: &mut Vec<GuiShellAction>,
) {
    let Ok(mut value) = i32::try_from(current) else {
        ui.text(format!("{label} is outside the desktop input range."));
        return;
    };
    if ui.input_int(label, &mut value).build() && value >= 0 {
        selected.push(action(i64::from(value)));
    }
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

fn draw_host_panels(
    ui: &Ui,
    tab_name: &str,
    panels: &[crate::GuiRibbonPanel],
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    let label_y = ui.cursor_screen_pos()[1] + RIBBON_PANEL_DIVIDER_HEIGHT + 4.;
    let collapsed_width = collapsed_panel_width(ui, panels);
    let widths = cached_panel_widths(tab_name, panels, workspace);
    let expanded = panels_that_fit(
        &widths,
        panel_divider_width(ui),
        collapsed_width,
        ui.content_region_avail()[0],
    );
    for (index, panel) in panels.iter().enumerate() {
        if index > 0 {
            draw_panel_divider(ui);
        }
        if index < expanded {
            let width = draw_host_panel(ui, panel, label_y, selected, icons, workspace);
            workspace
                .ribbon_panel_widths
                .insert(ribbon_panel_width_key(tab_name, &panel.name), width);
        } else {
            draw_collapsed_panel(ui, tab_name, panel, label_y, selected, icons, workspace);
        }
        if index + 1 != panels.len() {
            ui.same_line();
        }
    }
}

fn draw_panel_divider(ui: &Ui) {
    let (start, end) = panel_divider_line(ui.cursor_screen_pos());
    ui.get_window_draw_list()
        .add_line(start, end, [0.42, 0.44, 0.49, 0.64])
        .thickness(1.)
        .build();
    ui.set_cursor_screen_pos([end[0] + 7., start[1]]);
}

fn panel_divider_width(ui: &Ui) -> f32 {
    2. * ui.clone_style().item_spacing[0] + 1.
}

fn panel_divider_line(cursor: [f32; 2]) -> ([f32; 2], [f32; 2]) {
    let x = cursor[0] + 3.;
    ([x, cursor[1]], [x, cursor[1] + RIBBON_PANEL_DIVIDER_HEIGHT])
}

fn draw_host_panel(
    ui: &Ui,
    panel: &crate::GuiRibbonPanel,
    label_y: f32,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) -> f32 {
    ui.group(|| {
        ui.group(|| {
            draw_panel_columns(
                ui,
                &pack_panel_columns(&panel.controls),
                selected,
                icons,
                workspace,
            );
        });
        let content_minimum = ui.item_rect_min();
        let content_maximum = ui.item_rect_max();
        ui.set_cursor_screen_pos([content_minimum[0], label_y]);
        draw_panel_name(ui, &panel.name, content_minimum, content_maximum);
    });
    ui.item_rect_max()[0] - ui.item_rect_min()[0]
}

fn draw_collapsed_panel(
    ui: &Ui,
    tab_name: &str,
    panel: &crate::GuiRibbonPanel,
    label_y: f32,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    let width = collapsed_panel_width(ui, std::slice::from_ref(panel));
    let popup = format!("##ribbon-panel-{tab_name}-{}", panel.name);
    ui.group(|| {
        let minimum = ui.cursor_screen_pos();
        if ui.button_with_size(format!("⌄{popup}"), [width, RIBBON_PANEL_DIVIDER_HEIGHT]) {
            ui.open_popup(&popup);
        }
        if ui.is_item_hovered() {
            ui.tooltip_text(format!("{} — show commands", panel.name));
        }
        let maximum = ui.item_rect_max();
        workspace.record_target(
            &format!("RibbonPanel.{tab_name}.{}", panel.name),
            true,
            minimum,
            maximum,
        );
        ui.set_cursor_screen_pos([minimum[0], label_y]);
        draw_panel_name(ui, &panel.name, minimum, maximum);
    });
    if let Some(_popup) = ui.begin_popup(&popup) {
        draw_panel_columns(
            ui,
            &pack_panel_columns(&panel.controls),
            selected,
            icons,
            workspace,
        );
    }
}

fn cached_panel_widths(
    tab_name: &str,
    panels: &[crate::GuiRibbonPanel],
    workspace: &WorkspaceShellState,
) -> Vec<f32> {
    panels
        .iter()
        .map(|panel| {
            workspace
                .ribbon_panel_widths
                .get(&ribbon_panel_width_key(tab_name, &panel.name))
                .copied()
                .unwrap_or_default()
        })
        .collect()
}

fn ribbon_panel_width_key(tab_name: &str, panel_name: &str) -> String {
    format!("{tab_name}\u{0}{panel_name}")
}

fn collapsed_panel_width(ui: &Ui, panels: &[crate::GuiRibbonPanel]) -> f32 {
    panels
        .iter()
        .map(|panel| ui.calc_text_size(&panel.name)[0] + 2. * ui.clone_style().frame_padding[0])
        .fold(44., f32::max)
}

fn panels_that_fit(widths: &[f32], divider: f32, collapsed: f32, available: f32) -> usize {
    (0..=widths.len())
        .rev()
        .find(|expanded| panel_row_width(widths, *expanded, divider, collapsed) <= available)
        .unwrap_or_default()
}

fn panel_row_width(widths: &[f32], expanded: usize, divider: f32, collapsed: f32) -> f32 {
    let panel_width = widths
        .iter()
        .enumerate()
        .map(|(index, width)| if index < expanded { *width } else { collapsed })
        .sum::<f32>();
    panel_width + divider * widths.len().saturating_sub(1) as f32
}

fn draw_panel_name(ui: &Ui, name: &str, minimum: [f32; 2], maximum: [f32; 2]) {
    let width = ui.calc_text_size(name)[0];
    let x = ((minimum[0] + maximum[0] - width) / 2.).max(minimum[0]);
    let y = ui.cursor_screen_pos()[1];
    ui.set_cursor_screen_pos([x, y]);
    ui.text(name);
}

fn draw_panel_columns(
    ui: &Ui,
    columns: &[Vec<&crate::GuiRibbonControl>],
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    for (index, column) in columns.iter().enumerate() {
        ui.group(|| {
            for control in column {
                draw_host_control(ui, control, selected, icons, workspace);
            }
        });
        if index + 1 != columns.len() {
            ui.same_line();
        }
    }
}

fn pack_panel_columns(controls: &[crate::GuiRibbonControl]) -> Vec<Vec<&crate::GuiRibbonControl>> {
    let mut columns = Vec::new();
    let mut stack = Vec::new();
    for control in controls {
        if control.button_style == 2 {
            push_panel_stack(&mut columns, &mut stack);
            columns.push(vec![control]);
            continue;
        }
        stack.push(control);
        if stack.len() == 3 {
            push_panel_stack(&mut columns, &mut stack);
        }
    }
    push_panel_stack(&mut columns, &mut stack);
    columns
}

fn push_panel_stack<'a>(
    columns: &mut Vec<Vec<&'a crate::GuiRibbonControl>>,
    stack: &mut Vec<&'a crate::GuiRibbonControl>,
) {
    if stack.is_empty() {
        return;
    }
    columns.push(std::mem::take(stack));
}

fn draw_host_control(
    ui: &Ui,
    control: &crate::GuiRibbonControl,
    selected: &mut Vec<GuiShellAction>,
    icons: &mut crate::icon_assets::GuiIconCache,
    workspace: &mut WorkspaceShellState,
) {
    let action = host_action(&control.command_id);
    let disabled = ui.begin_disabled(!control.enabled);
    let label = format!("{}##{}", control.display_name, control.command_id);
    let (clicked, minimum, maximum) = if control.button_style == 2 {
        draw_large_ribbon_control(ui, control, &label, icons)
    } else {
        let clicked = draw_small_ribbon_control(ui, control, &label, icons);
        (clicked, ui.item_rect_min(), ui.item_rect_max())
    };
    drop(disabled);
    workspace.record_control(control, minimum, maximum);
    if ui.is_item_hovered() && !control.display_name.is_empty() {
        ui.tooltip_text(&control.display_name);
    }
    if clicked && control.enabled {
        dispatch_ribbon_control(&control.command_id, action, selected, workspace);
    }
}

fn dispatch_ribbon_control(
    command_id: &str,
    action: Option<GuiShellAction>,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut WorkspaceShellState,
) {
    if command_id == "GetStarted.Preferences" {
        workspace.preferences_open = true;
        return;
    }
    if let Some(action) = action {
        selected.push(action);
    }
}

fn draw_large_ribbon_control(
    ui: &Ui,
    control: &crate::GuiRibbonControl,
    label: &str,
    icons: &mut crate::icon_assets::GuiIconCache,
) -> (bool, [f32; 2], [f32; 2]) {
    let Some(icon) = icons.texture_id(&control.icon) else {
        let clicked = ui.button_with_size(label, [70., 52.]);
        return (clicked, ui.item_rect_min(), ui.item_rect_max());
    };
    let clicked = ui.image_button(label, TextureId::new(icon), [40., 40.]);
    let minimum = ui.item_rect_min();
    let maximum = ui.item_rect_max();
    draw_large_caption(ui, &control.display_name, minimum, maximum);
    (clicked, minimum, maximum)
}

fn draw_large_caption(ui: &Ui, caption: &str, minimum: [f32; 2], maximum: [f32; 2]) {
    for line in large_caption_lines(caption) {
        let width = ui.calc_text_size(line)[0];
        let x = ((minimum[0] + maximum[0] - width) / 2.).max(minimum[0]);
        ui.set_cursor_screen_pos([x, ui.cursor_screen_pos()[1]]);
        ui.text(line);
    }
}

fn large_caption_lines(caption: &str) -> Vec<&str> {
    let starts: Vec<_> = caption
        .char_indices()
        .filter_map(|(index, character)| {
            (index == 0 || caption[..index].ends_with(char::is_whitespace))
                .then_some((index, character))
        })
        .map(|(index, _)| index)
        .collect();
    if starts.len() < 2 {
        return vec![caption];
    }
    let split = starts[starts.len().div_ceil(2)];
    vec![caption[..split].trim(), caption[split..].trim()]
}

fn draw_small_ribbon_control(
    ui: &Ui,
    control: &crate::GuiRibbonControl,
    label: &str,
    icons: &mut crate::icon_assets::GuiIconCache,
) -> bool {
    let Some(icon) = icons.texture_id(&control.icon) else {
        return ui.button(label);
    };
    ui.group(|| {
        let clicked = ui.image_button(format!("{}##icon", label), TextureId::new(icon), [18., 18.]);
        ui.same_line();
        ui.text(&control.display_name);
        clicked
    })
}
fn host_action(id: &str) -> Option<GuiShellAction> {
    match id {
        "GetStarted.NewPart" => Some(GuiShellAction::NewPart),
        "GetStarted.NewAssembly" => Some(GuiShellAction::NewAssembly),
        "GetStarted.NewDrawing" => Some(GuiShellAction::NewDrawing),
        "File.Save" => Some(GuiShellAction::Save),
        "Sketch.Create" => Some(GuiShellAction::SketchXY),
        "Sketch.Rectangle" => Some(GuiShellAction::Rectangle),
        "Sketch.Finish" => Some(GuiShellAction::ExitSketch),
        "Feature.Extrude" => Some(GuiShellAction::Extrude),
        "View.Home" => Some(GuiShellAction::Home),
        "View.Previous" => Some(GuiShellAction::PreviousView),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::{
        WorkspaceShellState, dispatch_ribbon_control, history_entry_label, large_caption_lines,
        pack_panel_columns, panel_divider_line, panels_that_fit, select_available_ribbon_tab,
        select_contextual_ribbon_tab, set_history_document,
    };

    #[test]
    fn zero_document_preferences_opens_the_shared_preferences_panel() {
        let mut workspace = WorkspaceShellState::default();
        let mut selected = Vec::new();
        dispatch_ribbon_control(
            "GetStarted.Preferences",
            None,
            &mut selected,
            &mut workspace,
        );
        assert!(workspace.preferences_open);
        assert!(selected.is_empty());
    }

    #[test]
    fn command_window_starts_with_an_empty_focused_input_line() {
        let workspace = WorkspaceShellState::default();
        assert!(workspace.command_input.is_empty());
        assert!(workspace.command_focus_next);
    }
    use crate::{
        GuiCommandState, GuiFrameEngine, GuiFrameRequest, GuiHistoryEntry, GuiRibbonControl,
        GuiRibbonPanel, GuiRibbonTab,
    };

    #[test]
    fn history_picker_replaces_a_document_and_marks_saved_entries() {
        let mut selected = vec![7, 9];
        set_history_document(&mut selected, 7, false);
        set_history_document(&mut selected, 12, true);
        assert_eq!(selected, [9, 12]);
        assert_eq!(
            history_entry_label(&GuiHistoryEntry {
                label: "Extrude".into(),
                saved: true,
            }),
            "* Extrude"
        );
    }

    #[test]
    fn panel_columns_match_the_go_three_row_ribbon_flow() {
        let controls = [
            small("Move"),
            small("Copy"),
            small("Rotate"),
            large("Extrude"),
            small("Hole"),
        ];
        let names = pack_panel_columns(&controls)
            .into_iter()
            .map(|column| {
                column
                    .into_iter()
                    .map(|control| control.display_name.as_str())
                    .collect::<Vec<_>>()
            })
            .collect::<Vec<_>>();
        assert_eq!(
            names,
            vec![
                vec!["Move", "Copy", "Rotate"],
                vec!["Extrude"],
                vec!["Hole"]
            ]
        );
    }

    #[test]
    fn large_caption_wraps_at_the_middle_word_like_the_go_ribbon() {
        assert_eq!(large_caption_lines("New 2D Sketch"), ["New 2D", "Sketch"]);
        assert_eq!(large_caption_lines("Extrude"), ["Extrude"]);
    }

    #[test]
    fn panel_divider_uses_the_go_control_grid_height() {
        assert_eq!(panel_divider_line([100., 40.]), ([103., 40.], [103., 148.]));
    }

    #[test]
    fn narrow_ribbon_collapses_trailing_panels_before_overflowing() {
        assert_eq!(panels_that_fit(&[100., 90., 80.], 10., 40., 240.), 1);
        assert_eq!(panels_that_fit(&[0., 0., 0.], 10., 40., 100.), 3);
    }

    #[test]
    fn contextual_sketch_tab_selects_on_entry_and_model_tab_on_exit() {
        let mut workspace = WorkspaceShellState::default();
        let sketch = GuiCommandState {
            ribbon_tabs: vec![GuiRibbonTab {
                name: "Sketch".into(),
                ..Default::default()
            }],
            ..Default::default()
        };
        select_contextual_ribbon_tab(&sketch, &mut workspace);
        assert_eq!(workspace.active_tab, "Sketch");
        select_contextual_ribbon_tab(&GuiCommandState::default(), &mut workspace);
        assert_eq!(workspace.active_tab, "Create & Modify");
    }

    #[test]
    fn document_ribbon_change_selects_its_first_tab() {
        let mut workspace = WorkspaceShellState {
            active_tab: "View".into(),
            active_ribbon_key: "Part".into(),
            ..Default::default()
        };
        let assembly = GuiCommandState {
            ribbon_key: "Assembly".into(),
            ribbon_tabs: vec![GuiRibbonTab {
                name: "Assemble".into(),
                ..Default::default()
            }],
            ..Default::default()
        };
        select_available_ribbon_tab(&assembly, &mut workspace);
        assert_eq!(workspace.active_tab, "Assemble");
    }

    #[test]
    fn unavailable_ribbon_controls_are_drawn_without_a_host_action() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let draws = engine
            .frame(&ribbon_request(unavailable_control()))
            .unwrap()
            .draws;
        let baseline = engine
            .frame(&ribbon_request(GuiRibbonControl::default()))
            .unwrap()
            .draws;
        assert_ne!(draws, baseline);
    }

    #[test]
    fn narrow_ribbon_exposes_collapsed_panel_flyouts() {
        let mut engine = GuiFrameEngine::new().unwrap();
        let request = narrow_ribbon_request();
        engine.frame(&request).unwrap();
        let second = engine.frame(&request).unwrap();
        assert!(
            second
                .targets
                .iter()
                .any(|target| target.command_id == "RibbonPanel.Test.Third")
        );
    }

    fn unavailable_control() -> GuiRibbonControl {
        GuiRibbonControl {
            command_id: "Future.Unavailable".into(),
            display_name: "Unavailable".into(),
            enabled: false,
            button_style: 2,
            ..Default::default()
        }
    }

    fn ribbon_request(control: GuiRibbonControl) -> GuiFrameRequest {
        GuiFrameRequest {
            extent: [640, 360],
            commands: GuiCommandState {
                ribbon_tabs: vec![GuiRibbonTab {
                    name: "Test".into(),
                    panels: vec![GuiRibbonPanel {
                        name: "Panel".into(),
                        controls: vec![control],
                    }],
                }],
                ..Default::default()
            },
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1.; 2],
            events: Vec::new(),
            status: None,
        }
    }

    fn narrow_ribbon_request() -> GuiFrameRequest {
        let panel = |name: &str| GuiRibbonPanel {
            name: name.into(),
            controls: vec![GuiRibbonControl {
                command_id: format!("{name}.Action"),
                display_name: "Very Wide Ribbon Command".into(),
                enabled: true,
                button_style: 2,
                ..Default::default()
            }],
        };
        GuiFrameRequest {
            extent: [180, 360],
            commands: GuiCommandState {
                ribbon_tabs: vec![GuiRibbonTab {
                    name: "Test".into(),
                    panels: ["First", "Second", "Third"].map(panel).to_vec(),
                }],
                ..Default::default()
            },
            delta_seconds: 1. / 60.,
            framebuffer_scale: [1.; 2],
            events: Vec::new(),
            status: None,
        }
    }

    fn small(name: &str) -> GuiRibbonControl {
        GuiRibbonControl {
            display_name: name.into(),
            ..Default::default()
        }
    }

    fn large(name: &str) -> GuiRibbonControl {
        GuiRibbonControl {
            display_name: name.into(),
            button_style: 2,
            ..Default::default()
        }
    }
}
