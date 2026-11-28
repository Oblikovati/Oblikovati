// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Surface Boundary Patch (#3918).
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
    GuiBoundaryCondition, GuiBoundaryCurveRef, GuiBoundaryEdgeSet, GuiBoundaryPatchAction,
    GuiBoundaryPatchEditSeed, GuiBoundaryPatchPick, GuiBoundaryPatchPreview,
    GuiBoundaryPatchPreviewState, GuiBoundaryPatchRequest, GuiBoundaryPatchSelectionMode,
};

pub(super) struct DraftEdgeSet {
    pub(super) edges: Vec<GuiBoundaryCurveRef>,
    pub(super) condition: GuiBoundaryCondition,
    pub(super) weight_text: String,
}

impl DraftEdgeSet {
    pub(super) fn parsed_weight(&self) -> Option<f64> {
        let trimmed = self.weight_text.trim();
        let value: f64 = trimmed.parse().ok()?;
        (value.is_finite() && value >= 0.0).then_some(value)
    }

    fn to_edge_set(&self) -> Option<GuiBoundaryEdgeSet> {
        let weight = if self.condition.supports_weight() {
            self.parsed_weight()?
        } else {
            self.parsed_weight().unwrap_or(1.0)
        };
        Some(GuiBoundaryEdgeSet {
            edges: self.edges.clone(),
            condition: self.condition,
            weight,
        })
    }
}

pub(crate) struct BoundaryPatchDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) automatic_edge_chain: bool,
    pub(super) selection_mode: GuiBoundaryPatchSelectionMode,
    pub(super) edge_sets: Vec<DraftEdgeSet>,
    pub(super) guide_rails: Vec<GuiBoundaryCurveRef>,
    pub(super) boundary_open: bool,
    pub(super) guide_rails_open: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiBoundaryPatchRequest>,
    pub(super) submitted: Option<GuiBoundaryPatchRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick_seq: u64,
}

impl Default for BoundaryPatchDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            automatic_edge_chain: true,
            selection_mode: GuiBoundaryPatchSelectionMode::Boundary,
            edge_sets: Vec::new(),
            guide_rails: Vec::new(),
            boundary_open: true,
            guide_rails_open: true,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick_seq: 0,
        }
    }
}

impl DockableWindow for BoundaryPatchDialog {}

pub(super) fn format_weight(value: f64) -> String {
    let raw = format!("{value:.4}");
    let trimmed = raw.trim_end_matches('0').trim_end_matches('.');
    if trimmed.is_empty() {
        "0".to_string()
    } else {
        trimmed.to_string()
    }
}

impl BoundaryPatchDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Boundary Patch", &self.dock)
    }

    pub(super) fn set_selection_mode(
        &mut self,
        mode: GuiBoundaryPatchSelectionMode,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.selection_mode = mode;
        actions.push(GuiShellAction::BoundaryPatch(
            GuiBoundaryPatchAction::SetSelectionMode(mode),
        ));
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
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.boundary_open = true;
        self.guide_rails_open = true;
        self.selection_mode = GuiBoundaryPatchSelectionMode::Boundary;
        self.consumed_pick_seq = state.boundary_patch_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.boundary_patch_edits.get(&id))
        {
            self.automatic_edge_chain = seed.automatic_edge_chain;
            self.edge_sets = seed
                .edge_sets
                .iter()
                .map(|set| DraftEdgeSet {
                    edges: set.edges.clone(),
                    condition: set.condition,
                    weight_text: format_weight(set.weight),
                })
                .collect();
            self.guide_rails = seed.guide_rails.clone();
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::OpenEdit(id),
            ));
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::SetAutomaticEdgeChain(self.automatic_edge_chain),
            ));
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::SetSelectionMode(self.selection_mode),
            ));
            return;
        }
        self.automatic_edge_chain = true;
        self.edge_sets.clear();
        self.guide_rails.clear();
        if self.document.is_some() {
            actions.push(GuiShellAction::BoundaryPatch(GuiBoundaryPatchAction::Open));
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::SetAutomaticEdgeChain(true),
            ));
            actions.push(GuiShellAction::BoundaryPatch(
                GuiBoundaryPatchAction::SetSelectionMode(GuiBoundaryPatchSelectionMode::Boundary),
            ));
        }
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::BoundaryPatch(GuiBoundaryPatchAction::Clear));
        }
        self.editing = None;
        self.edge_sets.clear();
        self.guide_rails.clear();
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
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

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiBoundaryPatchRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        let edge_sets: Option<Vec<GuiBoundaryEdgeSet>> = self
            .edge_sets
            .iter()
            .map(DraftEdgeSet::to_edge_set)
            .collect();
        Some(GuiBoundaryPatchRequest {
            document,
            generation,
            feature: self.editing,
            automatic_edge_chain: self.automatic_edge_chain,
            edge_sets: edge_sets?,
            guide_rails: self.guide_rails.clone(),
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.boundary_patch_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        if pick.edges.is_empty() {
            return;
        }
        match pick.mode {
            GuiBoundaryPatchSelectionMode::GuideRails => {
                for &curve in &pick.edges {
                    let in_boundary = self.edge_sets.iter().any(|set| set.edges.contains(&curve));
                    if !in_boundary && !self.guide_rails.contains(&curve) {
                        self.guide_rails.push(curve);
                    }
                }
            }
            GuiBoundaryPatchSelectionMode::Boundary => {
                let fresh: Vec<GuiBoundaryCurveRef> = pick
                    .edges
                    .iter()
                    .copied()
                    .filter(|curve| {
                        !self.edge_sets.iter().any(|set| set.edges.contains(curve))
                            && !self.guide_rails.contains(curve)
                    })
                    .collect();
                if !fresh.is_empty() {
                    self.edge_sets.push(DraftEdgeSet {
                        edges: fresh,
                        condition: GuiBoundaryCondition::Free,
                        weight_text: format_weight(1.0),
                    });
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
        let height = 460.0_f32.min((area.bottom - area.top) - 24.0).max(280.0);
        widget_abi::fit_viewport_properties(ui, "Boundary Patch", area, [296.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Boundary Patch", &mut open).build(|| {
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
                .unwrap_or("Boundary Patch");
            let header = draw_properties_header(
                ui,
                icons,
                title,
                "##boundary-patch-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            self.draw_boundary_section(ui, actions);
            ui.separator();
            self.draw_guide_rails_section(ui, actions);
            ui.separator();
            self.draw_output_and_status(ui, state);
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
            GuiShellAction::BoundaryPatch(GuiBoundaryPatchAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::BoundaryPatch(
                    GuiBoundaryPatchAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::BoundaryPatch(GuiBoundaryPatchAction::Clear)),
            }
        }
    }
}
