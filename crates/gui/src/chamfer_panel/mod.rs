// SPDX-License-Identifier: GPL-2.0-only
//! Upgraded v1.1 Properties panel state and lifecycle for Part Modify Chamfer (#4046).
mod controls;
pub mod types;

use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    chamfer_options::{ChamferDrivenDimension, ChamferMode},
};
pub use types::{
    GuiChamferPartialState, GuiChamferPickMode, GuiChamferTab, GuiChamferV2Action,
    GuiChamferV2EditSeed, GuiChamferV2Pick, GuiChamferV2Preview, GuiChamferV2PreviewState,
    GuiChamferV2Request,
};

#[derive(Default)]
pub(crate) struct ChamferV2State {
    pub(crate) tab: GuiChamferTab,
    pub(crate) body: Option<Vec<u8>>,
    pub(crate) edges: Vec<Vec<u8>>,
    pub(crate) mode: ChamferMode,
    pub(crate) distance2: String,
    pub(crate) angle: String,
    pub(crate) reference_face: Option<Vec<u8>>,
    pub(crate) face_pick_active: bool,
    pub(crate) reversed: bool,
    pub(crate) edge_chain_second: bool,
    pub(crate) setback_second: bool,
    pub(crate) partial: Option<GuiChamferPartialState>,
    pub(crate) driven_choice: ChamferDrivenDimension,
    pub(crate) consumed_pick_seq: u64,
    pub(crate) attempted: Option<GuiChamferV2Request>,
    pub(crate) submitted_close: Option<GuiChamferV2Request>,
    pub(crate) submitted_apply: Option<GuiChamferV2Request>,
    pub(crate) failure_modal: Option<String>,
}

impl ChamferV2State {
    pub(crate) fn reset_new(&mut self, pick_seq: u64) {
        self.tab = GuiChamferTab::Chamfer;
        self.body = None;
        self.edges.clear();
        self.mode = ChamferMode::EqualDistance;
        self.distance2 = "2 mm".into();
        self.angle = "45 deg".into();
        self.reference_face = None;
        self.face_pick_active = false;
        self.reversed = false;
        self.edge_chain_second = false;
        self.setback_second = false;
        self.partial = None;
        self.driven_choice = ChamferDrivenDimension::Chamfer;
        self.consumed_pick_seq = pick_seq;
        self.attempted = None;
        self.submitted_close = None;
        self.submitted_apply = None;
        self.failure_modal = None;
    }

    pub(crate) fn reset_after_apply(&mut self) {
        self.body = None;
        self.edges.clear();
        self.reference_face = None;
        self.partial = None;
        self.attempted = None;
        self.submitted_apply = None;
        self.failure_modal = None;
    }

    pub(crate) fn hydrate_edit(&mut self, seed: &GuiChamferV2EditSeed, pick_seq: u64) {
        self.tab = GuiChamferTab::Chamfer;
        self.body = Some(seed.body.clone());
        self.edges = seed.edges.clone();
        self.mode = seed.mode;
        self.distance2 = seed.distance2.clone();
        self.angle = seed.angle.clone();
        self.reference_face = seed.reference_face.clone();
        self.face_pick_active = false;
        self.reversed = seed.reversed;
        self.edge_chain_second = seed.edge_chain_second;
        self.setback_second = seed.setback_second;
        self.partial = seed.partial.clone();
        self.driven_choice = seed
            .partial
            .as_ref()
            .map_or(ChamferDrivenDimension::Chamfer, |p| p.driven);
        self.consumed_pick_seq = pick_seq;
        self.attempted = None;
        self.submitted_close = None;
        self.submitted_apply = None;
        self.failure_modal = None;
    }

    pub(crate) fn sync_ambient_edges(&mut self, state: &GuiCommandState, is_edit: bool) {
        if is_edit {
            return;
        }
        if let Some((_, ambient)) = &state.fillet_selection
            && self.edges != *ambient
        {
            self.edges = ambient.clone();
            if self.edges.is_empty() {
                self.partial = None;
            }
        }
    }

    pub(crate) fn consume_pick(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.chamfer_v2_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        if !pick.body.is_empty() {
            self.body = Some(pick.body.clone());
        }
        match pick.mode {
            GuiChamferPickMode::Face => {
                if let Some(face) = &pick.face {
                    self.reference_face = Some(face.clone());
                }
                self.face_pick_active = false;
            }
            GuiChamferPickMode::PartialEdge => {
                self.activate_partial_from_edge(state);
            }
            GuiChamferPickMode::None => {}
        }
    }

    pub(crate) fn activate_partial_from_edge(&mut self, state: &GuiCommandState) {
        if self.edges.is_empty() {
            return;
        }
        let edge_len_cm = state
            .chamfer_v2_preview
            .as_ref()
            .and_then(|p| p.edge_length_cm)
            .filter(|len| len.is_finite() && *len > 1e-4)
            .unwrap_or(0.8);
        let len_mm = edge_len_cm * 10.0;
        let to_end_mm = (len_mm * 0.25)
            .clamp(0.5, 2.0)
            .min((len_mm * 0.45).max(0.1));
        let chamfer_mm = (len_mm - to_end_mm).max(0.1);
        self.partial = Some(GuiChamferPartialState {
            edge_length_cm: edge_len_cm,
            to_start: format_mm(0.0),
            chamfer_length: format_mm(chamfer_mm),
            to_end: format_mm(to_end_mm),
            driven: self.driven_choice,
        });
    }

    pub(crate) fn recalculate_partial(&mut self) {
        let Some(partial) = self.partial.as_mut() else {
            return;
        };
        partial.driven = self.driven_choice;
        let total_mm = partial.edge_length_cm * 10.0;
        let s = parse_non_negative_mm(&partial.to_start).unwrap_or(0.0);
        let c = parse_non_negative_mm(&partial.chamfer_length).unwrap_or(0.0);
        let e = parse_non_negative_mm(&partial.to_end).unwrap_or(0.0);
        match partial.driven {
            ChamferDrivenDimension::Chamfer => {
                let new_c = (total_mm - s - e).max(0.0);
                partial.chamfer_length = format_mm(new_c);
            }
            ChamferDrivenDimension::ToEnd => {
                let new_e = (total_mm - s - c).max(0.0);
                partial.to_end = format_mm(new_e);
            }
            ChamferDrivenDimension::ToStart => {
                let new_s = (total_mm - c - e).max(0.0);
                partial.to_start = format_mm(new_s);
            }
        }
    }

    pub(crate) fn request(
        &self,
        document: u64,
        state: &GuiCommandState,
        feature: Option<FeatureId>,
        distance: &str,
    ) -> Option<GuiChamferV2Request> {
        let generation = state.document_generation?;
        Some(GuiChamferV2Request {
            document,
            generation,
            feature,
            body: self.body.clone(),
            edges: self.edges.clone(),
            mode: self.mode,
            distance: distance.to_owned(),
            distance2: self.distance2.clone(),
            angle: self.angle.clone(),
            reference_face: self.reference_face.clone(),
            reversed: self.reversed,
            edge_chain_second: self.edge_chain_second,
            setback_second: self.setback_second,
            partial: self.partial.clone(),
        })
    }

    pub(crate) fn local_error(&self, distance: &str) -> Option<&'static str> {
        if parse_positive_mm(distance).is_none() {
            return Some("Distance must be greater than zero");
        }
        match self.mode {
            ChamferMode::EqualDistance => {}
            ChamferMode::DistanceAndAngle => {
                let angle = parse_angle_deg(&self.angle)?;
                if angle <= 0.0 || angle >= 90.0 {
                    return Some("Angle must be between 0 and 90 degrees");
                }
            }
            ChamferMode::TwoDistances => {
                if parse_positive_mm(&self.distance2).is_none() {
                    return Some("Distance2 must be greater than zero");
                }
            }
        }
        if let Some(p) = &self.partial {
            let s = parse_non_negative_mm(&p.to_start)?;
            let c = parse_positive_mm(&p.chamfer_length)?;
            let e = parse_non_negative_mm(&p.to_end)?;
            if s + c + e > p.edge_length_cm * 10.0 + 0.05 {
                return Some("Partial lengths exceed edge length");
            }
        }
        None
    }

    pub(crate) fn fitted_height(&self, ui: &Ui) -> f32 {
        let row = ui.frame_height_with_spacing();
        let pad = ui.clone_style().window_padding[1] * 2.0;
        let rows = match self.tab {
            GuiChamferTab::Chamfer => match self.mode {
                ChamferMode::EqualDistance => 6.8,
                ChamferMode::DistanceAndAngle => 8.8,
                ChamferMode::TwoDistances => 8.8,
            },
            GuiChamferTab::Partial => {
                if self.partial.is_some() {
                    9.0
                } else {
                    6.8
                }
            }
        };
        rows * row + pad + 44.0
    }

    pub(crate) fn draw_body(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        distance: &mut String,
        is_edit: bool,
        edited_edges: usize,
        actions: &mut Vec<GuiShellAction>,
    ) {
        controls::draw_chamfer_body(self, ui, state, distance, is_edit, edited_edges, actions);
    }

    pub(crate) fn draw_failure_modal(&mut self, ui: &Ui) {
        controls::draw_failure_modal(self, ui);
    }
}

pub(super) fn format_mm(mm: f64) -> String {
    let rounded = (mm * 100.0).round() / 100.0;
    if (rounded - rounded.round()).abs() < 1e-6 {
        format!("{} mm", rounded.round() as i64)
    } else {
        format!("{rounded:.2} mm")
    }
}

pub(super) fn parse_positive_mm(text: &str) -> Option<f64> {
    let v = parse_non_negative_mm(text)?;
    (v > 0.0).then_some(v)
}

pub(super) fn parse_non_negative_mm(text: &str) -> Option<f64> {
    let trimmed = text.trim().replace(',', ".");
    let (num_str, factor) = if let Some(s) = trimmed.strip_suffix("mm") {
        (s.trim(), 1.0)
    } else if let Some(s) = trimmed.strip_suffix("cm") {
        (s.trim(), 10.0)
    } else if let Some(s) = trimmed.strip_suffix("in") {
        (s.trim(), 25.4)
    } else {
        (trimmed.as_str(), 1.0)
    };
    let val: f64 = num_str.parse().ok()?;
    (val.is_finite() && val >= 0.0).then_some(val * factor)
}

pub(super) fn parse_angle_deg(text: &str) -> Option<f64> {
    let trimmed = text.trim().replace(',', ".");
    let num_str = trimmed
        .strip_suffix("deg")
        .or_else(|| trimmed.strip_suffix('°'))
        .unwrap_or(&trimmed)
        .trim();
    let val: f64 = num_str.parse().ok()?;
    val.is_finite().then_some(val)
}
