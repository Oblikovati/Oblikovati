// SPDX-License-Identifier: GPL-2.0-only
//! Import file picker, STL Import Options modal, and dockable STEP Import panel.
mod picker;
mod stl_options;
mod types;
pub use types::{
    GuiImportAction, GuiImportEditSeed, GuiImportPreview, GuiImportPreviewState, GuiImportRequest,
    GuiImportSourceSummary,
};

use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    imported_model::{ImportMode, ImportedFileKind, StepImportOptions, StlImportOptions},
};
use picker::{ImportPicker, ImportPickerOutcome, is_stl_path};
use std::path::Path;
use stl_options::{StlOptionsModal, StlOptionsOutcome};

const SURFACE_MODES: [&str; 3] = ["Composite", "Individual Surfaces", "Stitch Surfaces"];

#[derive(Default)]
pub(crate) struct ImportDialog {
    dock: crate::dockable_lifecycle::DockableState,
    picker: ImportPicker,
    stl_options_modal: StlOptionsModal,
    document: Option<u64>,
    submitted: Option<GuiImportRequest>,
    feature: Option<FeatureId>,
    source_path: String,
    source_name: String,
    file_kind: ImportedFileKind,
    import_mode: ImportMode,
    body_names: Vec<String>,
    stl_options: StlImportOptions,
    step_options: StepImportOptions,
    model_loaded: bool,
    presets_shown: bool,
    popup_was_open: bool,
}

impl DockableWindow for ImportDialog {}

impl ImportDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Import", &self.dock)
    }

    pub(crate) fn is_opening(&self) -> bool {
        self.picker.is_opening()
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        feature: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = state.document else {
            return;
        };
        if let Some(id) = feature {
            let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.import_edits.get(&id))
                .cloned()
            else {
                return;
            };
            self.picker.close();
            self.stl_options_modal.close();
            self.document = Some(document);
            self.feature = Some(id);
            self.source_path = seed.source_path.clone();
            self.source_name = seed.source_name;
            self.file_kind = seed.file_kind;
            self.import_mode = seed.import_mode;
            self.body_names = seed.body_names;
            self.stl_options = seed.stl_options;
            self.step_options = seed.step_options;
            self.model_loaded = true;
            self.submitted = None;
            self.dock.set_open(true);
            actions.push(GuiShellAction::Import(GuiImportAction::InspectSource {
                document,
                path: seed.source_path,
            }));
            return;
        }
        self.document = Some(document);
        self.feature = None;
        self.source_path.clear();
        self.source_name.clear();
        self.body_names.clear();
        self.stl_options = StlImportOptions {
            template_units: template_unit_label(state),
            ..StlImportOptions::default()
        };
        self.step_options = StepImportOptions::default();
        self.import_mode = ImportMode::ConvertModel;
        self.model_loaded = false;
        self.submitted = None;
        self.dock.set_open(false);
        self.stl_options_modal.close();
        let initial_dir = state
            .browser
            .as_ref()
            .and_then(|b| b.path.as_deref())
            .and_then(|p| Path::new(p).parent());
        self.picker.open_in(initial_dir);
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        let was_active = self.document.take().is_some()
            || self.picker.is_open()
            || self.stl_options_modal.is_open();
        self.picker.close();
        self.stl_options_modal.close();
        self.submitted = None;
        self.dock.set_open(false);
        if was_active {
            actions.push(GuiShellAction::Import(GuiImportAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_none() {
            return;
        }
        if self.document != state.document || !GuiImportAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if let Some(summary) = &state.import_source
            && summary.source_path == self.source_path
        {
            self.source_name = summary.source_name.clone();
            self.body_names = summary.solid_names.clone();
            if summary.file_kind == ImportedFileKind::Stl {
                let units = template_unit_label(state);
                self.stl_options.bounding_box = summary.bounding_box;
                self.stl_options.template_units = units.clone();
                self.stl_options_modal
                    .update_inspection(summary.bounding_box, units);
            }
        }
        if !self.dock.is_open() || self.source_path.is_empty() {
            return;
        }
        let Some(request) = self.request(state) else {
            return;
        };
        if self.submitted.as_ref() != Some(&request) {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::Import(GuiImportAction::Preview(Box::new(
                request,
            ))));
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_none() {
            return;
        }
        if self.document != state.document || !GuiImportAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        let Some(document) = self.document else {
            return;
        };
        match self.picker.draw(ui) {
            ImportPickerOutcome::Pending => {}
            ImportPickerOutcome::OpenOptions(path) => {
                self.source_path = path.clone();
                self.file_kind = ImportedFileKind::Stl;
                self.stl_options.template_units = template_unit_label(state);
                actions.push(GuiShellAction::Import(GuiImportAction::InspectSource {
                    document,
                    path,
                }));
                self.stl_options_modal.open(&self.stl_options);
            }
            ImportPickerOutcome::OpenFile(path) => {
                self.source_path = path.clone();
                self.source_name = Path::new(&path)
                    .file_name()
                    .and_then(|s| s.to_str())
                    .unwrap_or("ImportedModel")
                    .to_owned();
                if is_stl_path(&path) {
                    self.file_kind = ImportedFileKind::Stl;
                    self.import_mode = ImportMode::ConvertModel;
                    if let Some(req) = self.request(state) {
                        actions.push(GuiShellAction::Import(GuiImportAction::Commit(Box::new(
                            req,
                        ))));
                    }
                    self.document = None;
                    self.submitted = None;
                    self.dock.set_open(false);
                    return;
                }
                self.file_kind = ImportedFileKind::Step;
                self.import_mode = ImportMode::ConvertModel;
                self.model_loaded = false;
                self.submitted = None;
                actions.push(GuiShellAction::Import(GuiImportAction::InspectSource {
                    document,
                    path,
                }));
                self.dock.set_open(true);
            }
            ImportPickerOutcome::Cancelled => {
                self.close(actions);
                return;
            }
        }
        match self.stl_options_modal.draw(ui) {
            StlOptionsOutcome::Pending => {}
            StlOptionsOutcome::Accepted(updated) => {
                self.stl_options = updated;
                self.picker.reopen_current();
            }
            StlOptionsOutcome::Cancelled => {
                self.picker.reopen_current();
            }
        }
        if !self.dock.is_open() {
            return;
        }
        self.draw_step_panel(ui, state, area, icons, actions);
    }

    fn draw_step_panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        icons: &GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !self.popup_was_open && ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }
        let mut keep_open = true;
        let mut commit = false;
        let mut cancel = false;
        let request = self.request(state);
        let ready = request
            .as_ref()
            .is_some_and(|req| GuiImportAction::Commit(Box::new(req.clone())).enabled(state));
        let preview_error = request.as_ref().and_then(|req| {
            state
                .import_preview
                .as_ref()
                .filter(|preview| preview.request == *req)
                .and_then(|preview| match &preview.result {
                    GuiImportPreview::Failed(msg) => Some(msg.as_str()),
                    GuiImportPreview::Ready => None,
                })
        });
        widget_abi::fit_viewport_properties(ui, "Import", area, [324., 472.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Import", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let title = if self.source_name.is_empty() {
                "Import".to_string()
            } else {
                format!("Import: {}", self.source_name)
            };
            draw_properties_header(
                ui,
                icons,
                &title,
                "##import-advanced-settings",
                &mut self.presets_shown,
                None,
                |_| {},
            );
            let footer_height = 44.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(160.);
            {
                let body = widget_abi::scroll_region(ui, "##import-body", [0., body_height]);
                if body.visible {
                    self.draw_tabs(ui, preview_error);
                }
            }
            let footer = draw_properties_footer(ui, ready);
            commit = footer.ok;
            cancel = footer.cancel;
        });
        self.popup_was_open = widget_abi::any_popup_open(ui);
        if commit && let Some(req) = request {
            actions.push(GuiShellAction::Import(GuiImportAction::Commit(Box::new(
                req,
            ))));
            self.document = None;
            self.submitted = None;
            self.dock.set_open(false);
        } else if cancel || !keep_open {
            self.close(actions);
        }
    }

    fn draw_tabs(&mut self, ui: &Ui, preview_error: Option<&str>) {
        let Some(_tabs) = ui.tab_bar("##import-tabs") else {
            return;
        };
        if let Some(_tab) = ui.tab_item("Options") {
            self.draw_options_tab(ui);
        }
        if let Some(_tab) = ui.tab_item("Select") {
            self.draw_select_tab(ui);
        }
        if let Some(error) = preview_error {
            ui.spacing();
            ui.text_wrapped(error);
        }
    }

    fn draw_options_tab(&mut self, ui: &Ui) {
        if ui.collapsing_header("Import Type", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            let _ = ui.radio_button(
                "Convert Model",
                &mut self.import_mode,
                ImportMode::ConvertModel,
            );
            let _ = ui.radio_button(
                "Reference Model",
                &mut self.import_mode,
                ImportMode::ReferenceModel,
            );
        }
        if ui.collapsing_header("Object Filters", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            ui.checkbox("Solid Bodies", &mut self.step_options.import_solids);
            ui.same_line_with_spacing(0., 16.);
            ui.checkbox("Surface Bodies", &mut self.step_options.import_surfaces);
            ui.checkbox("Wires", &mut self.step_options.import_wires);
            ui.same_line_with_spacing(0., 50.);
            ui.checkbox("Points", &mut self.step_options.import_points);
        }
        if ui.collapsing_header("Part Options", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            if self.import_mode == ImportMode::ReferenceModel {
                ui.text_disabled("Reference Mode: Associative Link");
                ui.text_wrapped(&self.source_name);
            }
            crate::theme::property_label(ui, "Surfaces");
            ui.set_next_item_width(156.);
            if let Some(_combo) =
                ui.begin_combo("##import-surfaces", &self.step_options.surfaces_mode)
            {
                for mode in SURFACE_MODES {
                    if ui
                        .selectable_config(mode)
                        .selected(self.step_options.surfaces_mode == mode)
                        .build()
                    {
                        self.step_options.surfaces_mode = mode.to_owned();
                    }
                }
            }
        }
    }

    fn draw_select_tab(&mut self, ui: &Ui) {
        if widget_abi::button(ui, "Load Model", [112., 26.]) {
            self.model_loaded = true;
        }
        ui.separator();
        if self.model_loaded {
            if let Some(_root) = ui
                .tree_node_config(&self.source_name)
                .default_open(true)
                .push()
            {
                let names = if self.body_names.is_empty() {
                    vec!["Solid1".to_string()]
                } else {
                    self.body_names.clone()
                };
                for name in names {
                    ui.bullet_text(&name);
                }
            }
        } else {
            ui.text_disabled("Click Load Model to inspect solids and preview geometry.");
        }
    }

    fn request(&self, state: &GuiCommandState) -> Option<GuiImportRequest> {
        Some(GuiImportRequest {
            document: self.document?,
            generation: state.document_generation?,
            feature: self.feature,
            source_path: self.source_path.clone(),
            source_name: self.source_name.clone(),
            file_kind: self.file_kind,
            import_mode: self.import_mode,
            body_names: self.body_names.clone(),
            stl_options: self.stl_options.clone(),
            step_options: self.step_options.clone(),
            show_viewport_preview: self.model_loaded
                || self.import_mode == ImportMode::ReferenceModel,
        })
    }
}

fn template_unit_label(state: &GuiCommandState) -> String {
    state
        .document_units
        .as_ref()
        .map(|u| u.length_unit.clone())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| "millimeter".into())
}
