// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Thread (#3918).
mod catalog;
mod controls;
mod modals;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use catalog::{default_for_face, validate_fields};
use imgui::Ui;
use modals::{ThreadModalOutcome, ThreadModals};
use oblikovati_model::feature::{
    FeatureId,
    thread::{ThreadHandedness, ThreadSpecification},
};
pub use types::{
    GuiThreadAction, GuiThreadEditSeed, GuiThreadPick, GuiThreadPreview, GuiThreadPreviewState,
    GuiThreadRequest,
};

pub(crate) struct ThreadDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    editing: Option<FeatureId>,
    body: Option<Vec<u8>>,
    pub(super) face: Option<Vec<u8>>,
    pub(super) full_depth: bool,
    pub(super) measured_depth_mm: f64,
    pub(super) depth: String,
    pub(super) offset: String,
    pub(super) display_in_model: bool,
    pub(super) spec: ThreadSpecification,
    pub(super) face_conical: bool,
    pub(super) face_internal: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiThreadRequest>,
    pub(super) submitted: Option<GuiThreadRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick: Option<GuiThreadPick>,
    modals: ThreadModals,
}

impl Default for ThreadDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            face: None,
            full_depth: false,
            measured_depth_mm: 10.0,
            depth: "10 mm".into(),
            offset: "0 mm".into(),
            display_in_model: true,
            spec: ThreadSpecification {
                thread_type: "ANSI Metric M Profile".into(),
                size: "9".into(),
                designation: "M9x1.25".into(),
                class: "6H".into(),
                handedness: ThreadHandedness::RightHand,
            },
            face_conical: false,
            face_internal: true,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick: None,
            modals: ThreadModals::default(),
        }
    }
}

impl DockableWindow for ThreadDialog {}

impl ThreadDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Thread", &self.dock)
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        edit: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.modals.reset();
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.consumed_pick = state.thread_pick.clone();
        if edit.is_none() && !state.has_threadable_faces {
            self.document = None;
            self.editing = None;
            self.dock.set_open(false);
            self.modals.show_no_faces();
            return;
        }
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.thread_edits.get(&id))
        {
            self.body = Some(seed.body.clone());
            self.face = Some(seed.face.clone());
            self.full_depth = seed.full_depth;
            self.measured_depth_mm = seed.measured_depth_mm;
            self.depth = seed.depth.clone();
            self.offset = seed.offset.clone();
            self.display_in_model = seed.display_in_model;
            self.spec = seed.spec.clone();
            self.face_conical = seed.face_conical;
            self.face_internal = seed.face_internal;
            actions.push(GuiShellAction::Thread(GuiThreadAction::OpenEdit(id)));
            return;
        }
        self.body = None;
        self.face = None;
        self.full_depth = false;
        self.measured_depth_mm = 10.0;
        self.depth = "10 mm".into();
        self.offset = "0 mm".into();
        self.display_in_model = true;
        self.face_conical = false;
        self.face_internal = true;
        actions.push(GuiShellAction::Thread(GuiThreadAction::Open));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Thread(GuiThreadAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.modals.reset();
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
            if self.reopen_after_commit {
                self.reopen_after_commit = false;
                self.open(state, None, actions);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiThreadRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiThreadRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            face: self.face.clone(),
            full_depth: self.full_depth,
            depth: self.depth.clone(),
            offset: self.offset.clone(),
            display_in_model: self.display_in_model,
            spec: self.spec.clone(),
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_pick(&mut self, state: &GuiCommandState) {
        if self.editing.is_some() {
            return;
        }
        if let Some(pick) = &state.thread_pick
            && self.consumed_pick.as_ref() != Some(pick)
        {
            self.consumed_pick = Some(pick.clone());
            self.body = Some(pick.body.clone());
            self.face = Some(pick.face.clone());
            self.measured_depth_mm = pick.full_depth_mm;
            self.face_conical = pick.conical;
            self.face_internal = pick.internal;
            self.full_depth = false;
            self.offset = "0 mm".into();
            let (spec, depth) = default_for_face(
                pick.diameter_mm,
                pick.full_depth_mm,
                pick.conical,
                pick.internal,
            );
            self.spec = spec;
            self.depth = depth;
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
            self.modals.draw(ui);
            return;
        }
        self.consume_pick(state);
        let height = ((area.bottom - area.top) - 24.0).clamp(340.0, 560.0);
        widget_abi::fit_viewport_properties(ui, "Thread", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut enter_commit = false;
        let popup_active = widget_abi::any_popup_open(ui);
        Self::dockable_window(ui, "Thread", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Thread",
                "##thread-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(120.0);
            ui.child_window("##thread-body")
                .size([0.0, body_h])
                .build(|| {
                    enter_commit = self.draw_body(ui, state);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(ui, state, enter_commit, popup_active, actions);
        });
        if matches!(self.modals.draw(ui), ThreadModalOutcome::CloseThread) || !open {
            self.close(actions);
        }
    }

    fn draw_body(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let local_error = validate_fields(self.full_depth, &self.depth, &self.offset);
        if let Some(error) = local_error {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {error}"));
        } else if let Some(preview) = &state.thread_preview
            && let GuiThreadPreview::Failed(msg) = &preview.result
        {
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
        let enter_commit = self.draw_location_section(ui, local_error);
        self.draw_specification_section(ui);
        enter_commit
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Thread(GuiThreadAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Thread(GuiThreadAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Thread(GuiThreadAction::Clear)),
            }
        }
    }
}
