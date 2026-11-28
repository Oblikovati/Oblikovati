// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Mark (#3918).
mod controls;
mod modals;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use modals::{MarkModalOutcome, MarkModals};
use oblikovati_model::feature::{
    FeatureId,
    mark::{MarkDirection, MarkMethod},
};
pub use types::{
    GuiMarkAction, GuiMarkEditSeed, GuiMarkEligibility, GuiMarkFacePick, GuiMarkGeometrySet,
    GuiMarkPreview, GuiMarkPreviewState, GuiMarkRequest, GuiMarkSketchPick,
};

pub(crate) struct MarkDialog {
    dock: crate::dockable_lifecycle::DockableState,
    pub(super) document: Option<u64>,
    editing: Option<FeatureId>,
    body: Option<Vec<u8>>,
    pub(super) geometry_sets: Vec<GuiMarkGeometrySet>,
    pub(super) active_set: usize,
    pub(super) method: MarkMethod,
    pub(super) seed_face: Option<Vec<u8>>,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) automatic_face_chain: bool,
    pub(super) direction: MarkDirection,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiMarkRequest>,
    pub(super) submitted: Option<GuiMarkRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_face_pick: Option<GuiMarkFacePick>,
    consumed_sketch_pick: Option<GuiMarkSketchPick>,
    pub(in crate::mark_dialog) modals: MarkModals,
}

impl Default for MarkDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            geometry_sets: vec![GuiMarkGeometrySet::default()],
            active_set: 0,
            method: MarkMethod::ProjectToClosestFace,
            seed_face: None,
            faces: Vec::new(),
            automatic_face_chain: false,
            direction: MarkDirection::Positive,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_face_pick: None,
            consumed_sketch_pick: None,
            modals: MarkModals::default(),
        }
    }
}

impl DockableWindow for MarkDialog {}

impl MarkDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Mark", &self.dock)
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
        self.consumed_face_pick = state.mark_face_pick.clone();
        self.consumed_sketch_pick = state.mark_sketch_pick.clone();
        if edit.is_none() {
            match state.mark_eligibility {
                GuiMarkEligibility::MultipleSolidBodies => {
                    self.document = None;
                    self.editing = None;
                    self.dock.set_open(false);
                    self.modals.show_multi_body();
                    return;
                }
                GuiMarkEligibility::NoVolume => {
                    self.document = None;
                    self.editing = None;
                    self.dock.set_open(false);
                    self.modals.show_no_volume();
                    return;
                }
                GuiMarkEligibility::Eligible => {}
            }
        }
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.mark_edits.get(&id))
        {
            self.body = Some(seed.body.clone());
            self.geometry_sets = if seed.geometry_sets.is_empty() {
                vec![GuiMarkGeometrySet::default()]
            } else {
                seed.geometry_sets.clone()
            };
            self.active_set = 0;
            self.method = seed.method;
            self.seed_face = seed.faces.first().cloned();
            self.faces = seed.faces.clone();
            self.automatic_face_chain = seed.automatic_face_chain;
            self.direction = seed.direction;
            actions.push(GuiShellAction::Mark(GuiMarkAction::OpenEdit(id)));
            return;
        }
        self.body = None;
        self.geometry_sets = vec![GuiMarkGeometrySet::default()];
        self.active_set = 0;
        self.method = MarkMethod::ProjectToClosestFace;
        self.seed_face = None;
        self.faces.clear();
        self.automatic_face_chain = false;
        self.direction = MarkDirection::Positive;
        actions.push(GuiShellAction::Mark(GuiMarkAction::Open));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Mark(GuiMarkAction::Clear));
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

    pub(super) fn has_any_geometry(&self) -> bool {
        self.geometry_sets
            .iter()
            .any(|set| set.sketch.is_some() && !set.curves.is_empty())
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiMarkRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiMarkRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            geometry_sets: self.geometry_sets.clone(),
            method: self.method,
            faces: self.faces.clone(),
            automatic_face_chain: self.automatic_face_chain,
            direction: self.direction,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        if let Some(pick) = &state.mark_face_pick
            && self.consumed_face_pick.as_ref() != Some(pick)
        {
            self.consumed_face_pick = Some(pick.clone());
            self.body = Some(pick.body.clone());
            self.seed_face = Some(pick.seed_face.clone());
            self.faces = pick.faces.clone();
            self.automatic_face_chain = pick.automatic_face_chain;
        }
        if let Some(pick) = &state.mark_sketch_pick
            && self.consumed_sketch_pick.as_ref() != Some(pick)
        {
            self.consumed_sketch_pick = Some(pick.clone());
            if let Some(set) = self.geometry_sets.get_mut(self.active_set) {
                set.sketch = Some(pick.sketch);
                set.curves = pick.curves.clone();
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
            self.modals.draw(ui);
            return;
        }
        self.consume_picks(state);
        let height = ((area.bottom - area.top) - 24.0).clamp(320.0, 560.0);
        widget_abi::fit_viewport_properties(ui, "Mark", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let popup_active = widget_abi::any_popup_open(ui);
        Self::dockable_window(ui, "Mark", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Mark",
                "##mark-adv",
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
            ui.child_window("##mark-body")
                .size([0.0, body_h])
                .build(|| {
                    if let Some(preview) = &state.mark_preview
                        && let GuiMarkPreview::Failed(msg) = &preview.result
                    {
                        ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
                    }
                    self.draw_selection_sets_section(ui);
                    self.draw_behavior_section(ui, state, actions);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(ui, state, popup_active, actions);
        });
        if matches!(self.modals.draw(ui), MarkModalOutcome::CloseMark) || !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Mark(GuiMarkAction::Preview(Box::new(req.clone()))).dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => {
                    actions.push(GuiShellAction::Mark(GuiMarkAction::Preview(Box::new(req))))
                }
                None => actions.push(GuiShellAction::Mark(GuiMarkAction::Clear)),
            }
        }
    }
}
