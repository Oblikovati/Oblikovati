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

pub(crate) struct WorkspaceShellState {
    pub(crate) active_tab: RibbonTab,
    pub(crate) preferences_open: bool,
    pub(crate) selection_open: bool,
    pub(crate) history_open: bool,
    pub(crate) units_open: bool,
    pub(crate) model_browser_open: bool,
    pub(crate) command_panel_open: bool,
    selection_filter: SelectionFilterState,
    history_documents: Vec<u64>,
    icons: crate::icon_assets::GuiIconCache,
}

impl Default for WorkspaceShellState {
    fn default() -> Self {
        Self {
            active_tab: Default::default(),
            preferences_open: false,
            selection_open: false,
            history_open: false,
            units_open: false,
            model_browser_open: true,
            command_panel_open: true,
            selection_filter: Default::default(),
            history_documents: Vec::new(),
            icons: Default::default(),
        }
    }
}
impl WorkspaceShellState {
    pub(crate) fn textures(&self) -> &[crate::GuiTexture] {
        self.icons.textures()
    }
}

struct SelectionFilterState {
    entries: Vec<SelectionFilterEntry>,
}
impl SelectionFilterState {
    fn move_entry(&mut self, index: usize, offset: isize) {
        let Some(target) = index.checked_add_signed(offset) else {
            return;
        };
        if target < self.entries.len() {
            self.entries.swap(index, target);
        }
    }
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
    draw_menu_bar(ui, state, selected, parameters, workspace);
    draw_ribbon(ui, state, selected, parameters, workspace);
    workspace_panels::draw_document_tabs(ui, &state.documents, selected);
    if workspace.model_browser_open {
        workspace_panels::draw_model_browser(
            ui,
            state.model_tree.as_ref(),
            selected,
            &mut workspace.model_browser_open,
        );
    }
    if workspace.command_panel_open {
        workspace_panels::draw_command_window(ui, status, &mut workspace.command_panel_open);
    }
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
        &mut workspace.selection_filter,
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
                RibbonTab::View => draw_view_tab(ui, state, selected, parameters, workspace),
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
    workspace: &mut WorkspaceShellState,
) {
    ribbon_action(
        ui,
        "Home",
        GuiShellAction::Home,
        state,
        selected,
        &mut workspace.icons,
    );
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

fn draw_selection_filter(ui: &Ui, open: &mut bool, filter: &mut SelectionFilterState) {
    if !*open {
        return;
    }
    ui.window("Selection Filter")
        .opened(open)
        .size([300., 360.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Tick to allow picking. Earlier rows have higher priority.");
            ui.separator();
            for index in 0..filter.entries.len() {
                draw_selection_filter_entry(ui, filter, index);
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

fn draw_selection_filter_entry(ui: &Ui, filter: &mut SelectionFilterState, index: usize) {
    let entry = &mut filter.entries[index];
    ui.checkbox(entry.label, &mut entry.enabled);
    ui.same_line();
    let move_up = ui.small_button(format!("↑##selection-up-{index}"));
    ui.same_line();
    let move_down = ui.small_button(format!("↓##selection-down-{index}"));
    if move_up {
        filter.move_entry(index, -1);
    }
    if move_down {
        filter.move_entry(index, 1);
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
    if let Some(id) = icons.texture_id(icon_for_action(&action)) {
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

fn icon_for_action(action: &GuiShellAction) -> &'static str {
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
        GuiShellAction::CloseDocument(_) => "close",
        GuiShellAction::JumpHistoryTo { .. } => "history",
        GuiShellAction::SetLengthUnit(_)
        | GuiShellAction::SetAngleUnit(_)
        | GuiShellAction::SetMassUnit(_)
        | GuiShellAction::SetTimeUnit(_)
        | GuiShellAction::SetLengthPrecision(_)
        | GuiShellAction::SetAnglePrecision(_)
        | GuiShellAction::SetLengthFormat(_) => "settings",
        GuiShellAction::SetBodyVisible { .. } => "visibility",
        GuiShellAction::SetGridSpacingCm(_)
        | GuiShellAction::SetGridVisible(_)
        | GuiShellAction::SetGridMajorEvery(_)
        | GuiShellAction::SetHeadsUpDisplay(_)
        | GuiShellAction::SetPointerInput(_)
        | GuiShellAction::SetDimensionInput(_)
        | GuiShellAction::SetSnapToPoints(_)
        | GuiShellAction::SetSnapToGrid(_)
        | GuiShellAction::SetRelaxMode(_)
        | GuiShellAction::SetAutoProjectOrigin(_)
        | GuiShellAction::SetSuppressFormatOverrides(_) => "settings",
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
    use super::{SelectionFilterState, history_entry_label, set_history_document};
    use crate::GuiHistoryEntry;

    #[test]
    fn selection_filter_defaults_to_the_host_pick_order() {
        let filter = SelectionFilterState::default();
        assert_eq!(filter.entries.len(), 12);
        assert!(filter.entries.iter().all(|entry| entry.enabled));
        assert_eq!(filter.entries[0].label, "Work points");
        assert_eq!(filter.entries[11].label, "Sketch entities");
    }

    #[test]
    fn selection_filter_moves_entries_without_changing_their_enabled_state() {
        let mut filter = SelectionFilterState::default();
        filter.entries[1].enabled = false;
        filter.move_entry(1, -1);
        assert_eq!(filter.entries[0].label, "Work axes");
        assert!(!filter.entries[0].enabled);
        filter.move_entry(0, -1);
        assert_eq!(filter.entries[0].label, "Work axes");
    }

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
}
