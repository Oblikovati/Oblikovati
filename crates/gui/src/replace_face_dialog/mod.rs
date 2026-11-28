// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Surface Replace Face (#3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{FeatureId, replace_face::ReplaceFaceTarget};
pub use types::{
    GuiReplaceFaceAction, GuiReplaceFaceEditSeed, GuiReplaceFacePick, GuiReplaceFacePickMode,
    GuiReplaceFacePreview, GuiReplaceFacePreviewState, GuiReplaceFaceRequest,
};

pub(crate) struct ReplaceFaceDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) body: Option<Vec<u8>>,
    pub(super) existing_faces: Vec<Vec<u8>>,
    pub(super) new_faces: Vec<ReplaceFaceTarget>,
    pub(super) automatic_face_chain: bool,
    pub(super) pick_mode: GuiReplaceFacePickMode,
    pub(super) input_geometry_open: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiReplaceFaceRequest>,
    pub(super) submitted: Option<GuiReplaceFaceRequest>,
    consumed_pick_seq: u64,
    pub(super) failure_modal: Option<String>,
    pub(super) failure_modal_open: bool,
}

impl Default for ReplaceFaceDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            existing_faces: Vec::new(),
            new_faces: Vec::new(),
            automatic_face_chain: false,
            pick_mode: GuiReplaceFacePickMode::ExistingFaces,
            input_geometry_open: true,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            consumed_pick_seq: 0,
            failure_modal: None,
            failure_modal_open: false,
        }
    }
}

impl DockableWindow for ReplaceFaceDialog {}

impl ReplaceFaceDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Replace Face", &self.dock)
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
        self.failure_modal = None;
        self.failure_modal_open = false;
        self.input_geometry_open = true;
        self.pick_mode = GuiReplaceFacePickMode::ExistingFaces;
        self.consumed_pick_seq = state.replace_face_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.replace_face_edits.get(&id))
        {
            self.body = Some(seed.body.clone());
            self.existing_faces = seed.existing_faces.clone();
            self.new_faces = seed.new_faces.clone();
            self.automatic_face_chain = seed.automatic_face_chain;
        } else {
            self.body = None;
            self.existing_faces.clear();
            self.new_faces.clear();
            self.automatic_face_chain = false;
        }
        actions.push(GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Open));
        actions.push(GuiShellAction::ReplaceFace(
            GuiReplaceFaceAction::SetPickMode(self.pick_mode),
        ));
        actions.push(GuiShellAction::ReplaceFace(
            GuiReplaceFaceAction::SetAutomaticFaceChain(self.automatic_face_chain),
        ));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Clear));
        }
        self.editing = None;
        self.body = None;
        self.existing_faces.clear();
        self.new_faces.clear();
        self.automatic_face_chain = false;
        self.pick_mode = GuiReplaceFacePickMode::ExistingFaces;
        self.attempted = None;
        self.submitted = None;
        self.failure_modal = None;
        self.failure_modal_open = false;
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

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiReplaceFaceRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiReplaceFaceRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            existing_faces: self.existing_faces.clone(),
            new_faces: self.new_faces.clone(),
            automatic_face_chain: self.automatic_face_chain,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(pick) = &state.replace_face_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        let route_to_new = self.pick_mode == GuiReplaceFacePickMode::NewFaces
            || (pick.is_surface && !self.existing_faces.is_empty());
        if route_to_new {
            if self.pick_mode != GuiReplaceFacePickMode::NewFaces {
                self.pick_mode = GuiReplaceFacePickMode::NewFaces;
                actions.push(GuiShellAction::ReplaceFace(
                    GuiReplaceFaceAction::SetPickMode(self.pick_mode),
                ));
            }
            for face in &pick.faces {
                let target = ReplaceFaceTarget {
                    body: pick.body.clone(),
                    face: face.clone(),
                };
                if !self.new_faces.iter().any(|item| item.face == target.face)
                    && !self.existing_faces.iter().any(|item| item == &target.face)
                {
                    self.new_faces.push(target);
                }
            }
            return;
        }
        if self.body.as_deref() != Some(pick.body.as_slice()) {
            self.body = Some(pick.body.clone());
            self.existing_faces.clear();
        }
        for face in &pick.faces {
            if !self.existing_faces.iter().any(|item| item == face) {
                self.existing_faces.push(face.clone());
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
        self.consume_picks(state, actions);
        let height = 310.0_f32.min((area.bottom - area.top) - 24.0).max(240.0);
        widget_abi::fit_viewport_properties(ui, "Replace Face", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Replace Face", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Replace Face",
                "##replace-face-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            self.draw_input_geometry(ui, actions);
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
            GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::ReplaceFace(
                    GuiReplaceFaceAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::ReplaceFace(GuiReplaceFaceAction::Clear)),
            }
        }
    }
}
