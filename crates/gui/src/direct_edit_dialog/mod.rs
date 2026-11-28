// SPDX-License-Identifier: GPL-2.0-only
//! Floating in-canvas Direct Edit toolbar (Move, Size, Scale, Rotate, Delete, #3918).
mod controls;
mod manipulator;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    direct_edit::{
        DirectEditCoordinateMode, DirectEditMode, DirectEditOperationKind, DirectEditScaleMode,
        DirectEditSelectionFilter,
    },
};
pub use types::{
    GuiDirectEditAction, GuiDirectEditEditSeed, GuiDirectEditPick, GuiDirectEditPreview,
    GuiDirectEditPreviewState, GuiDirectEditRequest,
};

pub(crate) const DIRECT_EDIT_WINDOW: &str = "Direct Edit";
const SNAP_PARALLEL_PROMPT: &str = "Select a face to specify the rotation angle.";

pub(crate) struct DirectEditDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) editing_op_index: Option<usize>,
    pub(super) mode: DirectEditMode,
    pub(super) filter: DirectEditSelectionFilter,
    pub(super) body: Option<Vec<u8>>,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) scale_mode: DirectEditScaleMode,
    pub(super) uniform_factor: String,
    pub(super) scale_factors: [String; 3],
    pub(super) coupled_from_uniform: bool,
    pub(super) coordinate_mode: DirectEditCoordinateMode,
    pub(super) snap_parallel: bool,
    attempted: Option<GuiDirectEditRequest>,
    pub(super) submitted_generation: Option<u64>,
    consumed_pick: Option<GuiDirectEditPick>,
}

impl Default for DirectEditDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            editing_op_index: None,
            mode: DirectEditMode::Move,
            filter: DirectEditSelectionFilter::Faces,
            body: None,
            faces: Vec::new(),
            scale_mode: DirectEditScaleMode::Uniform,
            uniform_factor: "1".into(),
            scale_factors: ["1".into(), "1".into(), "1".into()],
            coupled_from_uniform: false,
            coordinate_mode: DirectEditCoordinateMode::Local,
            snap_parallel: false,
            attempted: None,
            submitted_generation: None,
            consumed_pick: None,
        }
    }
}

impl DockableWindow for DirectEditDialog {}

impl DirectEditDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(DIRECT_EDIT_WINDOW, &self.dock)
    }

    pub(crate) fn active(&self) -> bool {
        self.document.is_some()
    }

    pub(crate) fn status_prompt(&self) -> Option<&'static str> {
        (self.document.is_some() && self.snap_parallel).then_some(SNAP_PARALLEL_PROMPT)
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        edit: Option<(FeatureId, Option<usize>)>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted_generation = None;
        self.consumed_pick = state.direct_edit_pick.clone();
        self.snap_parallel = false;
        self.coupled_from_uniform = false;

        if let Some((feature_id, op_idx)) = edit {
            self.editing = Some(feature_id);
            self.editing_op_index = op_idx;
            if let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.direct_edit_edits.get(&feature_id))
                && let Some(idx) = op_idx
                && let Some(op) = seed.operations.get(idx)
            {
                self.hydrate_operation(&seed.body, &op.kind);
                actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::OpenEdit {
                    feature: feature_id,
                    operation: op_idx,
                }));
                actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::SetFilter(
                    self.filter,
                )));
                return;
            }
            self.body = None;
            self.faces.clear();
            self.mode = DirectEditMode::Move;
            self.filter = DirectEditSelectionFilter::Faces;
            actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::OpenEdit {
                feature: feature_id,
                operation: op_idx,
            }));
            return;
        }

        self.editing = None;
        self.editing_op_index = None;
        self.mode = DirectEditMode::Move;
        self.filter = DirectEditSelectionFilter::Faces;
        self.body = None;
        self.faces.clear();
        self.scale_mode = DirectEditScaleMode::Uniform;
        self.uniform_factor = "1".into();
        self.scale_factors = ["1".into(), "1".into(), "1".into()];
        self.coordinate_mode = DirectEditCoordinateMode::Local;
        actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::Open));
    }

    fn hydrate_operation(&mut self, body: &[u8], kind: &DirectEditOperationKind) {
        self.body = Some(body.to_vec());
        self.faces.clear();
        self.coupled_from_uniform = false;
        self.snap_parallel = false;
        match kind {
            DirectEditOperationKind::Move { filter, faces, .. } => {
                self.mode = DirectEditMode::Move;
                self.filter = *filter;
                self.faces = faces.clone();
            }
            DirectEditOperationKind::Size { faces, .. } => {
                self.mode = DirectEditMode::Size;
                self.filter = DirectEditSelectionFilter::Faces;
                self.faces = faces.clone();
            }
            DirectEditOperationKind::Scale {
                mode,
                uniform,
                factors,
            } => {
                self.mode = DirectEditMode::Scale;
                self.filter = DirectEditSelectionFilter::Solids;
                self.scale_mode = *mode;
                self.uniform_factor = format_factor(*uniform);
                self.scale_factors = factors.map(format_factor);
            }
            DirectEditOperationKind::Rotate {
                filter,
                faces,
                coordinate,
                ..
            } => {
                self.mode = DirectEditMode::Rotate;
                self.filter = *filter;
                self.faces = faces.clone();
                self.coordinate_mode = *coordinate;
            }
            DirectEditOperationKind::Delete { faces } => {
                self.mode = DirectEditMode::Delete;
                self.filter = DirectEditSelectionFilter::Faces;
                self.faces = faces.clone();
            }
        }
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::Clear));
        }
        self.editing = None;
        self.editing_op_index = None;
        self.body = None;
        self.faces.clear();
        self.snap_parallel = false;
        self.coupled_from_uniform = false;
        self.attempted = None;
        self.submitted_generation = None;
    }

    pub(super) fn has_geometry_selection(&self) -> bool {
        match self.filter {
            DirectEditSelectionFilter::Faces => !self.faces.is_empty(),
            DirectEditSelectionFilter::Solids => self.body.is_some(),
        }
    }

    pub(super) fn has_active_selection(&self) -> bool {
        self.has_geometry_selection() || self.snap_parallel
    }

    pub(super) fn switch_mode(&mut self, mode: DirectEditMode, actions: &mut Vec<GuiShellAction>) {
        self.mode = mode;
        self.snap_parallel = false;
        if mode == DirectEditMode::Scale && self.filter != DirectEditSelectionFilter::Solids {
            self.set_filter(DirectEditSelectionFilter::Solids, actions);
        } else if mode == DirectEditMode::Size && self.filter != DirectEditSelectionFilter::Faces {
            self.set_filter(DirectEditSelectionFilter::Faces, actions);
        }
    }

    pub(super) fn set_filter(
        &mut self,
        filter: DirectEditSelectionFilter,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.filter = filter;
        self.body = None;
        self.faces.clear();
        self.snap_parallel = false;
        self.attempted = None;
        actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::SetFilter(
            filter,
        )));
        actions.push(GuiShellAction::DirectEdit(
            GuiDirectEditAction::ClearSelection,
        ));
    }

    pub(super) fn handle_red_x(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.has_active_selection() {
            self.body = None;
            self.faces.clear();
            self.snap_parallel = false;
            self.editing_op_index = None;
            self.attempted = None;
            actions.push(GuiShellAction::DirectEdit(
                GuiDirectEditAction::ClearSelection,
            ));
        } else {
            self.close(actions);
        }
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
        if let Some(submitted) = self.submitted_generation
            && state.document_generation != Some(submitted)
        {
            self.submitted_generation = None;
            if let Some(committed) = state
                .direct_edit_preview
                .as_ref()
                .and_then(|p| p.last_committed_feature)
                .or_else(|| {
                    state
                        .browser
                        .as_ref()
                        .and_then(|b| b.direct_edit_edits.keys().next_back().copied())
                })
            {
                self.editing = Some(committed);
            }
            self.editing_op_index = None;
            self.body = None;
            self.faces.clear();
            self.snap_parallel = false;
            self.coupled_from_uniform = false;
            self.attempted = None;
            actions.push(GuiShellAction::DirectEdit(
                GuiDirectEditAction::ClearSelection,
            ));
        }
    }

    fn effective_scale_factors(&self) -> [String; 3] {
        if self.scale_mode == DirectEditScaleMode::NonUniform && self.coupled_from_uniform {
            let x = self.scale_factors[0].clone();
            [x.clone(), x.clone(), x]
        } else {
            self.scale_factors.clone()
        }
    }

    pub(super) fn preview_request(&self, state: &GuiCommandState) -> Option<GuiDirectEditRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiDirectEditRequest {
            document,
            generation,
            feature: self.editing,
            editing_op_index: self.editing_op_index,
            body: self.body.clone(),
            faces: self.faces.clone(),
            mode: self.mode,
            filter: self.filter,
            scale_mode: self.scale_mode,
            uniform_factor: self.uniform_factor.clone(),
            scale_factors: self.scale_factors.clone(),
            coordinate_mode: self.coordinate_mode,
            snap_parallel: self.snap_parallel,
        })
    }

    pub(super) fn commit_request(&self, state: &GuiCommandState) -> Option<GuiDirectEditRequest> {
        let mut req = self.preview_request(state)?;
        req.scale_factors = self.effective_scale_factors();
        Some(req)
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.direct_edit_pick else {
            return;
        };
        if self.consumed_pick.as_ref() == Some(pick) {
            return;
        }
        self.consumed_pick = Some(pick.clone());
        self.body = Some(pick.body.clone());
        match pick.filter {
            DirectEditSelectionFilter::Faces => {
                if let Some(face) = &pick.face {
                    self.faces = vec![face.clone()];
                }
            }
            DirectEditSelectionFilter::Solids => {
                self.faces.clear();
            }
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state);
        manipulator::draw_manipulator(ui, state, area, self.mode, self.has_geometry_selection());

        let width = 344.0;
        let height = if self.mode == DirectEditMode::Scale {
            240.0
        } else {
            204.0
        };
        widget_abi::fit_viewport_properties(ui, DIRECT_EDIT_WINDOW, area, [width, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, DIRECT_EDIT_WINDOW, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.draw_mode_bar(ui, actions);
            ui.separator();
            self.draw_selection_row(ui, actions);
            ui.spacing();
            self.draw_mode_controls(ui);
            ui.separator();
            self.refresh_preview(state, actions);
            self.draw_action_bar(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.commit_request(state).filter(|req| {
            GuiShellAction::DirectEdit(GuiDirectEditAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            if let Some(req) = request {
                actions.push(GuiShellAction::DirectEdit(GuiDirectEditAction::Preview(
                    Box::new(req),
                )));
            }
        }
    }
}

fn format_factor(value: f64) -> String {
    let text = format!("{value:.4}");
    text.trim_end_matches('0').trim_end_matches('.').to_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn red_x_clears_selection_first_then_closes_toolbar() {
        let mut dialog = DirectEditDialog::default();
        let state = GuiCommandState {
            document: Some(1),
            document_generation: Some(1),
            authoring: GuiAuthoringContext::Part,
            ..Default::default()
        };
        let mut actions = Vec::new();
        dialog.open(&state, None, &mut actions);
        dialog.faces = vec![vec![1, 2, 3]];
        dialog.body = Some(vec![4, 5, 6]);
        dialog.handle_red_x(&mut actions);
        assert!(dialog.active());
        assert!(dialog.faces.is_empty());
        assert!(dialog.body.is_none());
        dialog.handle_red_x(&mut actions);
        assert!(!dialog.active());
    }

    #[test]
    fn switching_uniform_to_non_uniform_initializes_xyz_and_couples_x_only_edit() {
        let mut dialog = DirectEditDialog::default();
        let state = GuiCommandState {
            document: Some(1),
            document_generation: Some(2),
            authoring: GuiAuthoringContext::Part,
            ..Default::default()
        };
        let mut actions = Vec::new();
        dialog.open(&state, None, &mut actions);
        dialog.uniform_factor = "2".into();
        dialog.scale_mode = DirectEditScaleMode::NonUniform;
        dialog.scale_factors = ["2".into(), "2".into(), "2".into()];
        dialog.coupled_from_uniform = true;
        dialog.scale_factors[0] = "1".into();
        let req = dialog.commit_request(&state).unwrap();
        assert_eq!(req.scale_factors, ["1", "1", "1"]);
    }
}
