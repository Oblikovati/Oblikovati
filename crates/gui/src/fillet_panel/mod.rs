// SPDX-License-Identifier: GPL-2.0-only
//! Upgraded v1.1 Properties panel state and lifecycle for Part Modify Fillet.
mod controls;
pub mod types;

use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    fillet_options::{FilletAdvancedOptions, FilletContinuity, FilletMode},
};
pub use types::{
    GuiFilletCornerSetback, GuiFilletEdgeSet, GuiFilletSelectionMode, GuiFilletSharedVertex,
    GuiFilletV2Action, GuiFilletV2EditSeed, GuiFilletV2Pick, GuiFilletV2Preview,
    GuiFilletV2PreviewState, GuiFilletV2Request, GuiFilletVariablePoint,
};

#[derive(Clone, Debug)]
pub(crate) struct PartialFilletModal {
    pub(crate) failed_count: usize,
    pub(crate) total_count: usize,
    pub(crate) kept_sets: Vec<GuiFilletEdgeSet>,
    pub(crate) accept_successful: bool,
}

#[derive(Default)]
pub(crate) struct FilletV2State {
    pub(crate) body: Option<Vec<u8>>,
    pub(crate) mode: FilletMode,
    pub(crate) selection_mode: GuiFilletSelectionMode,
    pub(crate) continuity: FilletContinuity,
    pub(crate) smooth_radius_transition: bool,
    pub(crate) edge_sets: Vec<GuiFilletEdgeSet>,
    pub(crate) active_set: usize,
    pub(crate) variable_points: Vec<GuiFilletVariablePoint>,
    pub(crate) corner_setback: Option<GuiFilletCornerSetback>,
    pub(crate) advanced: FilletAdvancedOptions,
    pub(crate) advanced_open: bool,
    pub(crate) solid_selected: bool,
    pub(crate) used_v2: bool,
    pub(crate) status_notice: Option<String>,
    pub(crate) consumed_pick_seq: u64,
    pub(crate) attempted_v2: Option<GuiFilletV2Request>,
    pub(crate) submitted_v2: Option<GuiFilletV2Request>,
    pub(crate) partial_modal: Option<PartialFilletModal>,
    pub(crate) pending_commit_after_partial: bool,
}

impl FilletV2State {
    pub(crate) fn reset_new(&mut self, radius: &str, pick_seq: u64) {
        self.body = None;
        self.mode = FilletMode::Constant;
        self.selection_mode = GuiFilletSelectionMode::Edge;
        self.continuity = FilletContinuity::Tangent;
        self.smooth_radius_transition = true;
        self.edge_sets = vec![GuiFilletEdgeSet {
            edges: Vec::new(),
            radius: radius.into(),
            is_loop: false,
        }];
        self.active_set = 0;
        self.variable_points = vec![GuiFilletVariablePoint {
            position: "0.0".into(),
            radius: radius.into(),
        }];
        self.corner_setback = None;
        self.advanced = FilletAdvancedOptions::default();
        self.advanced_open = false;
        self.solid_selected = false;
        self.used_v2 = false;
        self.status_notice = None;
        self.consumed_pick_seq = pick_seq;
        self.attempted_v2 = None;
        self.submitted_v2 = None;
        self.partial_modal = None;
        self.pending_commit_after_partial = false;
    }

    pub(crate) fn hydrate_edit(&mut self, seed: &GuiFilletV2EditSeed, pick_seq: u64) {
        self.body = Some(seed.body.clone());
        self.mode = seed.mode;
        self.selection_mode = GuiFilletSelectionMode::Edge;
        self.continuity = seed.continuity;
        self.smooth_radius_transition = seed.smooth_radius_transition;
        self.edge_sets = if seed.edge_sets.is_empty() {
            vec![GuiFilletEdgeSet {
                edges: Vec::new(),
                radius: "2 mm".into(),
                is_loop: false,
            }]
        } else {
            seed.edge_sets.clone()
        };
        self.active_set = 0;
        self.variable_points = if seed.variable_points.is_empty() {
            let r = self
                .edge_sets
                .first()
                .map_or("2 mm", |s| s.radius.as_str())
                .to_owned();
            vec![GuiFilletVariablePoint {
                position: "0.0".into(),
                radius: r,
            }]
        } else {
            seed.variable_points.clone()
        };
        self.corner_setback = seed.corner_setback.clone();
        self.advanced = seed.advanced;
        self.advanced_open = false;
        self.solid_selected = false;
        self.used_v2 = seed.mode != FilletMode::Constant
            || seed.continuity != FilletContinuity::Tangent
            || !seed.smooth_radius_transition
            || seed.edge_sets.len() > 1
            || !seed.variable_points.is_empty()
            || seed.corner_setback.is_some()
            || seed.advanced != FilletAdvancedOptions::default();
        self.status_notice = None;
        self.consumed_pick_seq = pick_seq;
        self.attempted_v2 = None;
        self.submitted_v2 = None;
        self.partial_modal = None;
        self.pending_commit_after_partial = false;
    }

    pub(crate) fn is_v2_active(&self) -> bool {
        self.used_v2
            || self.mode != FilletMode::Constant
            || self.selection_mode != GuiFilletSelectionMode::Edge
            || self.continuity != FilletContinuity::Tangent
            || !self.smooth_radius_transition
            || self.edge_sets.len() > 1
            || self.corner_setback.is_some()
            || self.solid_selected
            || self.advanced != FilletAdvancedOptions::default()
    }

    pub(crate) fn sync_ambient_edges(
        &mut self,
        state: &GuiCommandState,
        radius: &str,
        is_edit: bool,
    ) {
        if is_edit || self.solid_selected || self.selection_mode != GuiFilletSelectionMode::Edge {
            return;
        }
        if self.edge_sets.is_empty() {
            self.edge_sets.push(GuiFilletEdgeSet {
                edges: Vec::new(),
                radius: radius.into(),
                is_loop: false,
            });
        }
        if self.edge_sets.len() == 1
            && !self.edge_sets[0].is_loop
            && let Some((_, ambient)) = &state.fillet_selection
        {
            self.edge_sets[0].edges = ambient.clone();
            self.edge_sets[0].radius = radius.into();
        }
    }

    pub(crate) fn consume_pick(&mut self, state: &GuiCommandState, radius: &str) {
        let Some(pick) = &state.fillet_v2_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        self.used_v2 = true;
        if !pick.body.is_empty() {
            self.body = Some(pick.body.clone());
        }
        match pick.mode {
            GuiFilletSelectionMode::CornerVertex => {
                if let Some(v) = &pick.vertex {
                    let count = v.incident_edges.len().max(3);
                    let default_r = self
                        .edge_sets
                        .first()
                        .map_or(radius, |s| s.radius.as_str())
                        .to_owned();
                    self.corner_setback = Some(GuiFilletCornerSetback {
                        vertex: v.key.clone(),
                        minimal: false,
                        distances: vec![default_r; count],
                    });
                    self.status_notice = None;
                }
            }
            GuiFilletSelectionMode::Solid => {
                self.solid_selected = true;
                self.edge_sets = pick
                    .edge_groups
                    .iter()
                    .map(|edges| GuiFilletEdgeSet {
                        edges: edges.clone(),
                        radius: radius.into(),
                        is_loop: false,
                    })
                    .collect();
                self.active_set = 0;
            }
            GuiFilletSelectionMode::Loop => {
                self.solid_selected = false;
                let edges = pick.edge_groups.first().cloned().unwrap_or_default();
                let r = self
                    .edge_sets
                    .get(self.active_set)
                    .map_or(radius, |s| s.radius.as_str())
                    .to_owned();
                if self.edge_sets.is_empty() {
                    self.edge_sets.push(GuiFilletEdgeSet {
                        edges,
                        radius: r,
                        is_loop: true,
                    });
                } else {
                    let idx = self.active_set.min(self.edge_sets.len() - 1);
                    self.edge_sets[idx].edges = edges;
                    self.edge_sets[idx].is_loop = true;
                }
            }
            GuiFilletSelectionMode::Feature | GuiFilletSelectionMode::Edge => {
                self.solid_selected = false;
                let edges = pick.edge_groups.first().cloned().unwrap_or_default();
                if self.edge_sets.is_empty() {
                    self.edge_sets.push(GuiFilletEdgeSet {
                        edges,
                        radius: radius.into(),
                        is_loop: false,
                    });
                } else {
                    let idx = self.active_set.min(self.edge_sets.len() - 1);
                    self.edge_sets[idx].edges = edges;
                    self.edge_sets[idx].is_loop = false;
                }
            }
        }
    }

    pub(crate) fn request(
        &self,
        document: u64,
        state: &GuiCommandState,
        feature: Option<FeatureId>,
    ) -> Option<GuiFilletV2Request> {
        let generation = state.document_generation?;
        Some(GuiFilletV2Request {
            document,
            generation,
            feature,
            body: self.body.clone(),
            mode: self.mode,
            continuity: self.continuity,
            smooth_radius_transition: self.smooth_radius_transition,
            edge_sets: self.edge_sets.clone(),
            variable_points: if self.mode == FilletMode::Variable {
                self.variable_points.clone()
            } else {
                Vec::new()
            },
            corner_setback: self.corner_setback.clone(),
            advanced: self.advanced,
            solid_selected: self.solid_selected,
        })
    }

    pub(crate) fn local_error(&self, radius: &str) -> Option<&'static str> {
        if self.mode == FilletMode::Variable {
            for pt in &self.variable_points {
                if parse_pos(&pt.position).is_none() {
                    return Some("Position must be between 0.0 and 1.0");
                }
                if parse_positive_length(&pt.radius).is_none() {
                    return Some("Radius must be greater than zero");
                }
            }
            return None;
        }
        if self.edge_sets.len() > 1 {
            for set in &self.edge_sets {
                if parse_positive_length(&set.radius).is_none() {
                    return Some("Radius must be greater than zero");
                }
            }
        } else if parse_positive_length(radius).is_none() {
            return Some("Radius must be greater than zero");
        }
        if let Some(sb) = &self.corner_setback
            && !sb.minimal
        {
            for dist in &sb.distances {
                if parse_positive_length(dist).is_none() {
                    return Some("Setback distance must be greater than zero");
                }
            }
        }
        None
    }

    pub(crate) fn draw_body(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        radius: &mut String,
        is_edit: bool,
        edited_edges: usize,
        actions: &mut Vec<GuiShellAction>,
    ) {
        controls::draw_fillet_body(self, ui, state, radius, is_edit, edited_edges, actions);
    }

    pub(crate) fn draw_partial_modal(
        &mut self,
        ui: &Ui,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        controls::draw_partial_modal(self, ui, actions)
    }
}

pub(super) fn parse_positive_length(text: &str) -> Option<f64> {
    let trimmed = text.trim().replace(',', ".");
    let num_str = trimmed
        .strip_suffix("mm")
        .or_else(|| trimmed.strip_suffix("cm"))
        .or_else(|| trimmed.strip_suffix("in"))
        .unwrap_or(&trimmed)
        .trim();
    let val: f64 = num_str.parse().ok()?;
    (val.is_finite() && val > 0.0).then_some(val)
}

pub(super) fn parse_pos(text: &str) -> Option<f64> {
    let val: f64 = text.trim().replace(',', ".").parse().ok()?;
    (val.is_finite() && (0.0..=1.0).contains(&val)).then_some(val)
}
