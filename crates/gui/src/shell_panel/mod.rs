// SPDX-License-Identifier: GPL-2.0-only
//! Upgraded Properties panel state and lifecycle for Part Modify Shell (#4106).
mod controls;
pub mod types;

use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    shell::{
        ShellApproximationTolerance, ShellApproximationType, ShellDirection, ShellOptions,
        ShellType,
    },
};
pub use types::{
    GuiShellOverrideSet, GuiShellPickMode, GuiShellV2Action, GuiShellV2EditSeed, GuiShellV2Pick,
    GuiShellV2Preview, GuiShellV2PreviewState, GuiShellV2Request,
};

#[derive(Default)]
pub(crate) struct ShellV2State {
    pub(crate) feature: Option<FeatureId>,
    pub(crate) body: Option<Vec<u8>>,
    pub(crate) openings: Vec<Vec<u8>>,
    pub(crate) direction: ShellDirection,
    pub(crate) shell_type: ShellType,
    pub(crate) overrides: Vec<GuiShellOverrideSet>,
    pub(crate) allow_approximation: bool,
    pub(crate) approximation_type: ShellApproximationType,
    pub(crate) approximation_tolerance: ShellApproximationTolerance,
    pub(crate) approximation_percent: String,
    pub(crate) pick_mode: GuiShellPickMode,
    pub(crate) pick_mode_dirty: bool,
    pub(crate) consumed_pick_seq: u64,
    pub(crate) attempted: Option<GuiShellV2Request>,
    pub(crate) submitted: Option<GuiShellV2Request>,
}

impl ShellV2State {
    pub(crate) fn reset_new(&mut self, state: &GuiCommandState) {
        self.feature = None;
        self.body = None;
        self.openings = seed_selected_faces(state);
        self.direction = ShellDirection::Inside;
        self.shell_type = ShellType::Sharp;
        self.overrides.clear();
        self.allow_approximation = false;
        self.approximation_type = ShellApproximationType::Mean;
        self.approximation_tolerance = ShellApproximationTolerance::Optimized;
        self.approximation_percent = "10.0 %".into();
        self.pick_mode = GuiShellPickMode::Remove;
        self.pick_mode_dirty = true;
        self.consumed_pick_seq = state.shell_v2_pick.as_ref().map_or(0, |p| p.sequence);
        self.attempted = None;
        self.submitted = None;
    }

    pub(crate) fn hydrate_edit(
        &mut self,
        feature: FeatureId,
        seed: &GuiShellV2EditSeed,
        pick_seq: u64,
    ) {
        self.feature = Some(feature);
        self.body = Some(seed.body.clone());
        self.openings = seed.openings.clone();
        self.direction = seed.direction;
        self.shell_type = seed.options.shell_type;
        self.overrides = seed.overrides.clone();
        self.allow_approximation = seed.options.allow_approximation;
        self.approximation_type = seed.options.approximation_type;
        self.approximation_tolerance = seed.options.approximation_tolerance;
        self.approximation_percent = format_percent(seed.options.approximation_percent);
        self.pick_mode = GuiShellPickMode::Remove;
        self.pick_mode_dirty = true;
        self.consumed_pick_seq = pick_seq;
        self.attempted = None;
        self.submitted = None;
    }

    pub(crate) fn needs_clear(&self) -> bool {
        self.attempted.is_some()
            || self.pick_mode != GuiShellPickMode::None
            || self.submitted.is_some()
    }

    pub(crate) fn reset_closed(&mut self) {
        self.feature = None;
        self.pick_mode = GuiShellPickMode::None;
        self.pick_mode_dirty = false;
        self.attempted = None;
        self.submitted = None;
    }

    pub(crate) fn sync_pick_mode(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.pick_mode_dirty {
            self.pick_mode_dirty = false;
            actions.push(GuiShellAction::ShellV2(GuiShellV2Action::SetPickMode(
                self.pick_mode,
            )));
        }
    }

    pub(crate) fn set_pick_mode(
        &mut self,
        mode: GuiShellPickMode,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.pick_mode = mode;
        self.pick_mode_dirty = false;
        actions.push(GuiShellAction::ShellV2(GuiShellV2Action::SetPickMode(mode)));
    }

    pub(crate) fn consume_pick(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.shell_v2_pick else {
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
            GuiShellPickMode::Remove => {
                for face in &pick.faces {
                    if !self.openings.iter().any(|existing| existing == face) {
                        self.openings.push(face.clone());
                    }
                }
            }
            GuiShellPickMode::Override(index) => {
                if let Some(face) = pick.faces.first()
                    && !self.openings.iter().any(|opening| opening == face)
                    && let Some(slot) = self.overrides.get_mut(index)
                {
                    slot.face = Some(face.clone());
                }
            }
            GuiShellPickMode::None => {}
        }
    }

    pub(crate) fn local_error(&self, thickness: &str) -> Option<&'static str> {
        if parse_positive_mm(thickness).is_none() {
            return Some("Thickness must be greater than zero");
        }
        if self.shell_type == ShellType::Sharp {
            for ov in &self.overrides {
                if ov.face.is_some() && parse_positive_mm(&ov.thickness).is_none() {
                    return Some("Override thickness must be greater than zero");
                }
            }
            if self.allow_approximation
                && self.approximation_tolerance == ShellApproximationTolerance::Specified
                && parse_positive_percent(&self.approximation_percent).is_none()
            {
                return Some("Approximation tolerance percent must be greater than zero");
            }
        }
        None
    }

    pub(crate) fn request(
        &self,
        document: u64,
        state: &GuiCommandState,
        thickness: &str,
    ) -> Option<GuiShellV2Request> {
        let generation = state.document_generation?;
        let overrides = if self.shell_type == ShellType::Sharp {
            self.overrides
                .iter()
                .filter_map(|ov| Some((ov.face.clone()?, ov.thickness.clone())))
                .collect()
        } else {
            Vec::new()
        };
        let approx_pct = parse_positive_percent(&self.approximation_percent).unwrap_or(10.0);
        let options = ShellOptions {
            shell_type: self.shell_type,
            allow_approximation: self.allow_approximation,
            approximation_type: self.approximation_type,
            approximation_tolerance: self.approximation_tolerance,
            approximation_percent: approx_pct,
        };
        Some(GuiShellV2Request {
            document,
            generation,
            feature: self.feature,
            body: self.body.clone(),
            openings: self.openings.clone(),
            thickness: thickness.to_owned(),
            direction: self.direction,
            overrides,
            options,
        })
    }

    pub(crate) fn fitted_height(&self, ui: &Ui) -> f32 {
        let row = ui.frame_height_with_spacing();
        let pad = ui.clone_style().window_padding[1] * 2.0;
        let base_rows = 6.6;
        let extra_rows = if self.shell_type == ShellType::Sharp {
            let override_rows = 1.6 + self.overrides.len() as f32 * 1.25;
            let approx_rows =
                if self.approximation_tolerance == ShellApproximationTolerance::Specified {
                    5.4
                } else {
                    4.2
                };
            override_rows + approx_rows
        } else {
            0.0
        };
        (base_rows + extra_rows) * row + pad + 44.0
    }

    pub(crate) fn draw_body(
        &mut self,
        ui: &Ui,
        thickness: &mut String,
        actions: &mut Vec<GuiShellAction>,
    ) {
        controls::draw_shell_body(self, ui, thickness, actions);
    }
}

fn seed_selected_faces(state: &GuiCommandState) -> Vec<Vec<u8>> {
    let mut faces = Vec::new();
    for (hex, _) in &state.selected_face_appearances {
        if let Some(bytes) = decode_hex_key(hex)
            && !faces.iter().any(|existing| existing == &bytes)
        {
            faces.push(bytes);
        }
    }
    if faces.is_empty()
        && let Some(face) = &state.selected_model_face
        && !face.is_empty()
    {
        faces.push(face.clone());
    }
    faces
}

fn decode_hex_key(hex: &str) -> Option<Vec<u8>> {
    let bytes = hex.as_bytes();
    if bytes.is_empty() || !bytes.len().is_multiple_of(2) {
        return None;
    }
    let mut out = Vec::with_capacity(bytes.len() / 2);
    for &[hi, lo] in bytes.as_chunks::<2>().0 {
        out.push((hex_nibble(hi)? << 4) | hex_nibble(lo)?);
    }
    Some(out)
}

fn hex_nibble(b: u8) -> Option<u8> {
    match b {
        b'0'..=b'9' => Some(b - b'0'),
        b'a'..=b'f' => Some(b - b'a' + 10),
        b'A'..=b'F' => Some(b - b'A' + 10),
        _ => None,
    }
}

pub(super) fn parse_positive_mm(text: &str) -> Option<f64> {
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
    (val.is_finite() && val > 0.0).then_some(val * factor)
}

pub(super) fn parse_positive_percent(text: &str) -> Option<f64> {
    let trimmed = text.trim().replace(',', ".");
    let num_str = trimmed
        .strip_suffix('%')
        .or_else(|| trimmed.strip_suffix("percent"))
        .unwrap_or(&trimmed)
        .trim();
    let val: f64 = num_str.parse().ok()?;
    (val.is_finite() && val > 0.0).then_some(val)
}

pub(super) fn format_percent(pct: f64) -> String {
    format!("{pct:.1} %")
}
