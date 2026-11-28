// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Delete Face (#3918).
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
    GuiDeleteFaceAction, GuiDeleteFaceEditSeed, GuiDeleteFacePick, GuiDeleteFacePreview,
    GuiDeleteFacePreviewState, GuiDeleteFaceRequest,
};

pub(crate) struct DeleteFaceDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) body: Option<Vec<u8>>,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) heal_remaining_faces: bool,
    pub(super) select_lump_or_void: bool,
    pub(super) input_geometry_open: bool,
    pub(super) behavior_open: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiDeleteFaceRequest>,
    pub(super) submitted: Option<GuiDeleteFaceRequest>,
    consumed_pick_seq: u64,
    pub(super) failure_modal: Option<String>,
}

impl Default for DeleteFaceDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            faces: Vec::new(),
            heal_remaining_faces: false,
            select_lump_or_void: false,
            input_geometry_open: true,
            behavior_open: true,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            consumed_pick_seq: 0,
            failure_modal: None,
        }
    }
}

impl DockableWindow for DeleteFaceDialog {}

impl DeleteFaceDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Delete Face", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.preview_hidden = false;
        self.failure_modal = None;
        self.input_geometry_open = true;
        self.behavior_open = true;
        self.consumed_pick_seq = state.delete_face_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.delete_face_edits.get(&id))
        {
            self.body = Some(seed.body.clone());
            self.faces = seed.faces.clone();
            self.heal_remaining_faces = seed.heal_remaining_faces;
            self.select_lump_or_void = seed.select_lump_or_void;
            return;
        }
        self.body = None;
        self.faces.clear();
        self.heal_remaining_faces = false;
        self.select_lump_or_void = false;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::DeleteFace(GuiDeleteFaceAction::Clear));
        }
        self.editing = None;
        self.body = None;
        self.faces.clear();
        self.attempted = None;
        self.submitted = None;
        self.failure_modal = None;
    }

    pub(crate) fn sync_before_docking(
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

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiDeleteFaceRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiDeleteFaceRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            faces: self.faces.clone(),
            heal_remaining_faces: self.heal_remaining_faces,
            select_lump_or_void: self.select_lump_or_void,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.delete_face_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        if self.body.as_deref() != Some(pick.body.as_slice()) {
            self.body = Some(pick.body.clone());
            self.faces.clear();
        }
        if !self.faces.iter().any(|face| face == &pick.face) {
            self.faces.push(pick.face.clone());
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        icons: &GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state);
        let height = 310.0_f32.min((area.bottom - area.top) - 24.0).max(240.0);
        widget_abi::fit_viewport_properties(ui, "Delete Face", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Delete Face", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Delete Face",
                "##delete-face-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            self.draw_input_geometry(ui);
            ui.separator();
            self.draw_behavior(ui);
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer(ui, state, actions);
        });
        self.draw_failure_modal(ui, actions);
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::DeleteFace(GuiDeleteFaceAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::DeleteFace(
                    GuiDeleteFaceAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::DeleteFace(GuiDeleteFaceAction::Clear)),
            }
        }
    }
}
