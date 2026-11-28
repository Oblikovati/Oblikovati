// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Finish (#3918).
mod catalog;
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::FeatureId;
pub use types::{
    GuiFinishAction, GuiFinishEditSeed, GuiFinishPick, GuiFinishPreview, GuiFinishPreviewState,
    GuiFinishRequest, GuiFinishSelectionMode, GuiFinishType,
};

pub(crate) struct FinishDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) body: Option<Vec<u8>>,
    pub(super) included_faces: Vec<Vec<u8>>,
    pub(super) excluded_faces: Vec<Vec<u8>>,
    pub(super) selection_mode: GuiFinishSelectionMode,
    pub(super) finish_type: GuiFinishType,
    pub(super) process: String,
    pub(super) appearance: String,
    pub(super) thickness: String,
    pub(super) hardness: String,
    pub(super) depth: String,
    pub(super) description: String,
    pub(super) short_description: String,
    pub(super) comment: String,
    pub(super) export_all_parameters: bool,
    pub(super) last_area_mm2: f64,
    pub(super) input_geometry_open: bool,
    pub(super) behavior_open: bool,
    pub(super) advanced_open: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiFinishRequest>,
    pub(super) submitted: Option<GuiFinishRequest>,
    consumed_pick_seq: u64,
}

impl Default for FinishDialog {
    fn default() -> Self {
        let preset = catalog::default_preset(GuiFinishType::Appearance);
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            included_faces: Vec::new(),
            excluded_faces: Vec::new(),
            selection_mode: GuiFinishSelectionMode::Include,
            finish_type: GuiFinishType::Appearance,
            process: preset.process.to_owned(),
            appearance: preset.appearance.to_owned(),
            thickness: preset.thickness.to_owned(),
            hardness: preset.hardness.to_owned(),
            depth: preset.depth.to_owned(),
            description: preset.description.to_owned(),
            short_description: preset.short_description.to_owned(),
            comment: String::new(),
            export_all_parameters: false,
            last_area_mm2: 0.0,
            input_geometry_open: true,
            behavior_open: true,
            advanced_open: true,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            consumed_pick_seq: 0,
        }
    }
}

impl DockableWindow for FinishDialog {}

impl FinishDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Finish", &self.dock)
    }

    fn apply_preset(&mut self, preset: &catalog::FinishPreset) {
        self.process = preset.process.to_owned();
        self.appearance = preset.appearance.to_owned();
        self.thickness = preset.thickness.to_owned();
        self.hardness = preset.hardness.to_owned();
        self.depth = preset.depth.to_owned();
        self.description = preset.description.to_owned();
        self.short_description = preset.short_description.to_owned();
    }

    pub(super) fn select_finish_type(&mut self, kind: GuiFinishType) {
        self.finish_type = kind;
        let preset = catalog::default_preset(kind);
        self.apply_preset(preset);
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        edit: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.preview_hidden = false;
        self.selection_mode = GuiFinishSelectionMode::Include;
        self.input_geometry_open = true;
        self.behavior_open = true;
        self.advanced_open = true;
        self.consumed_pick_seq = state.finish_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.finish_edits.get(&id))
        {
            self.body = seed.body.clone();
            self.included_faces = seed.included_faces.clone();
            self.excluded_faces = seed.excluded_faces.clone();
            self.finish_type = seed.finish_type;
            self.process = seed.process.clone();
            self.appearance = seed.appearance.clone();
            self.thickness = seed.thickness.clone();
            self.hardness = seed.hardness.clone();
            self.depth = seed.depth.clone();
            self.description = seed.description.clone();
            self.short_description = seed.short_description.clone();
            self.comment = seed.comment.clone();
            self.export_all_parameters = seed.export_all_parameters;
            self.last_area_mm2 = seed.area_mm2;
            actions.push(GuiShellAction::Finish(GuiFinishAction::OpenEdit(id)));
            return;
        }
        self.body = None;
        self.included_faces.clear();
        self.excluded_faces.clear();
        self.comment.clear();
        self.export_all_parameters = false;
        self.last_area_mm2 = 0.0;
        self.select_finish_type(GuiFinishType::Appearance);
        if self.document.is_some() {
            actions.push(GuiShellAction::Finish(GuiFinishAction::Open));
        }
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Finish(GuiFinishAction::Clear));
        }
        self.editing = None;
        self.body = None;
        self.included_faces.clear();
        self.excluded_faces.clear();
        self.attempted = None;
        self.submitted = None;
        self.last_area_mm2 = 0.0;
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document || state.authoring != GuiAuthoringContext::Part)
        {
            self.close(actions);
            return;
        }
        if let Some(submitted) = &self.submitted
            && state.document_generation != Some(submitted.generation)
        {
            self.close(actions);
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiFinishRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiFinishRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            included_faces: self.included_faces.clone(),
            excluded_faces: self.excluded_faces.clone(),
            finish_type: self.finish_type,
            process: self.process.clone(),
            appearance: self.appearance.clone(),
            thickness: self.thickness.clone(),
            hardness: self.hardness.clone(),
            depth: self.depth.clone(),
            description: self.description.clone(),
            short_description: self.short_description.clone(),
            comment: self.comment.clone(),
            export_all_parameters: self.export_all_parameters,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.finish_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        match self.selection_mode {
            GuiFinishSelectionMode::Include => {
                if self.body.as_deref() != Some(pick.body.as_slice()) {
                    self.body = Some(pick.body.clone());
                    self.included_faces.clear();
                    self.excluded_faces.clear();
                }
                self.excluded_faces.retain(|face| face != &pick.face);
                if !self.included_faces.iter().any(|face| face == &pick.face) {
                    self.included_faces.push(pick.face.clone());
                }
            }
            GuiFinishSelectionMode::Exclude => {
                // Reject excluding a face if it is not among a multi-face included set
                // (e.g. clicking the sole included face leaves Exclude at "Select Faces").
                let effective_after = self
                    .included_faces
                    .iter()
                    .filter(|face| {
                        *face != &pick.face && !self.excluded_faces.iter().any(|ex| ex == *face)
                    })
                    .count();
                let is_included = self.included_faces.iter().any(|face| face == &pick.face);
                if is_included
                    && effective_after >= 1
                    && !self.excluded_faces.iter().any(|face| face == &pick.face)
                {
                    self.excluded_faces.push(pick.face.clone());
                }
            }
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
        self.synchronize(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state);
        if let Some(preview) = &state.finish_preview
            && let GuiFinishPreview::Ready { area_mm2 } = preview.result
        {
            self.last_area_mm2 = area_mm2;
        }
        let height = 580.0_f32.min((area.bottom - area.top) - 24.0).max(320.0);
        widget_abi::fit_viewport_properties(ui, "Finish", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Finish", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let title = self
                .editing
                .and_then(|id| {
                    state.browser.as_ref().and_then(|b| {
                        b.features
                            .iter()
                            .find(|f| f.id == id)
                            .map(|f| f.name.as_str())
                    })
                })
                .unwrap_or("Finish");
            let header = draw_properties_header(
                ui,
                icons,
                title,
                "##finish-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            self.draw_input_geometry(ui, state);
            ui.separator();
            self.draw_behavior(ui);
            ui.separator();
            self.draw_advanced_properties(ui);
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Finish(GuiFinishAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Finish(GuiFinishAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Finish(GuiFinishAction::Clear)),
            }
        }
    }
}
