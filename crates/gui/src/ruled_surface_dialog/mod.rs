// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Surface Ruled Surface (#3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    ruled_surface::{RuledSurfaceType, RuledSurfaceVector},
    work_axis::WorkAxisReference,
};
pub use types::{
    GuiRuledSurfaceAction, GuiRuledSurfaceArrow, GuiRuledSurfaceEditSeed, GuiRuledSurfacePick,
    GuiRuledSurfacePickTarget, GuiRuledSurfacePreview, GuiRuledSurfacePreviewState,
    GuiRuledSurfaceRequest,
};

pub(super) const PANEL_WINDOW_TITLE: &str = "Ruled Surface";
const ARROW_COLOR: [f32; 4] = [1.0, 0.78, 0.16, 1.0];
const ARROW_FLIPPED_COLOR: [f32; 4] = [0.22, 0.82, 0.98, 1.0];

pub(crate) struct RuledSurfaceDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) target_body: Option<Vec<u8>>,
    pub(super) edges: Vec<Vec<u8>>,
    pub(super) ruled_type: RuledSurfaceType,
    pub(super) distance_text: String,
    pub(super) angle_text: String,
    pub(super) flip: bool,
    pub(super) automatic_edge_chain: bool,
    pub(super) vector: Option<RuledSurfaceVector>,
    pub(super) pick_target: GuiRuledSurfacePickTarget,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiRuledSurfaceRequest>,
    pub(super) submitted: Option<GuiRuledSurfaceRequest>,
    pub(super) apply_keep_open: bool,
    consumed_pick: Option<GuiRuledSurfacePick>,
}

impl Default for RuledSurfaceDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            target_body: None,
            edges: Vec::new(),
            ruled_type: RuledSurfaceType::Normal,
            distance_text: "10 mm".to_string(),
            angle_text: "0 deg".to_string(),
            flip: false,
            automatic_edge_chain: true,
            vector: None,
            pick_target: GuiRuledSurfacePickTarget::Edges,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            apply_keep_open: false,
            consumed_pick: None,
        }
    }
}

impl DockableWindow for RuledSurfaceDialog {}

impl RuledSurfaceDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(PANEL_WINDOW_TITLE, &self.dock)
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
        self.apply_keep_open = false;
        self.preview_hidden = false;
        self.consumed_pick = state.ruled_surface_pick.clone();
        self.pick_target = GuiRuledSurfacePickTarget::Edges;

        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.ruled_surface_edits.get(&id))
        {
            self.target_body = Some(seed.target_body.clone());
            self.edges = seed.edges.clone();
            self.ruled_type = seed.ruled_type;
            self.distance_text = format_mm_distance(seed.distance);
            self.angle_text = format_deg_angle(seed.angle_deg);
            self.flip = seed.flip;
            self.automatic_edge_chain = seed.automatic_edge_chain;
            self.vector = seed.vector.clone();
            self.sync_owner_flags(actions);
            return;
        }
        self.target_body = None;
        self.edges.clear();
        self.ruled_type = RuledSurfaceType::Normal;
        self.distance_text = "10 mm".to_string();
        self.angle_text = "0 deg".to_string();
        self.flip = false;
        self.automatic_edge_chain = true;
        self.vector = None;
        self.sync_owner_flags(actions);
    }

    fn reset_after_apply(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.editing = None;
        self.target_body = None;
        self.edges.clear();
        self.vector = None;
        self.flip = false;
        self.pick_target = GuiRuledSurfacePickTarget::Edges;
        self.attempted = None;
        self.submitted = None;
        self.consumed_pick = state.ruled_surface_pick.clone();
        actions.push(GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Open));
        self.sync_owner_flags(actions);
    }

    pub(super) fn sync_owner_flags(&self, actions: &mut Vec<GuiShellAction>) {
        actions.push(GuiShellAction::RuledSurface(
            GuiRuledSurfaceAction::SetAutomaticEdgeChain(self.automatic_edge_chain),
        ));
        actions.push(GuiShellAction::RuledSurface(
            GuiRuledSurfaceAction::SetPickTarget(self.pick_target),
        ));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.apply_keep_open = false;
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
            if self.apply_keep_open {
                self.apply_keep_open = false;
                self.reset_after_apply(state, actions);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn parsed_distance_cm(&self) -> Option<f64> {
        let cm = parse_distance_cm(&self.distance_text)?;
        (cm.is_finite() && cm > 0.0).then_some(cm)
    }

    pub(super) fn parsed_angle_deg(&self) -> Option<f64> {
        let deg = parse_angle_deg(&self.angle_text)?;
        (deg.is_finite() && (-180.0..=180.0).contains(&deg)).then_some(deg)
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiRuledSurfaceRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiRuledSurfaceRequest {
            document,
            generation,
            feature: self.editing,
            target_body: self.target_body.clone(),
            edges: self.edges.clone(),
            ruled_type: self.ruled_type,
            distance: self.parsed_distance_cm(),
            angle_deg: self.parsed_angle_deg(),
            flip: self.flip,
            automatic_edge_chain: self.automatic_edge_chain,
            vector: if self.ruled_type == RuledSurfaceType::Sweep {
                self.vector.clone()
            } else {
                None
            },
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, picked_axis: Option<WorkAxisReference>) {
        if self.ruled_type == RuledSurfaceType::Sweep
            && let Some(axis) = picked_axis
        {
            self.vector = Some(RuledSurfaceVector::from_work_axis_reference(axis));
        }
        let Some(pick) = &state.ruled_surface_pick else {
            return;
        };
        if self.consumed_pick.as_ref() == Some(pick) {
            return;
        }
        self.consumed_pick = Some(pick.clone());
        match pick.target {
            GuiRuledSurfacePickTarget::Edges => {
                let Some(body) = &pick.target_body else {
                    return;
                };
                if self.target_body.as_ref() != Some(body) {
                    self.target_body = Some(body.clone());
                    self.edges = pick.edges.clone();
                } else {
                    for edge in &pick.edges {
                        if !self.edges.iter().any(|existing| existing == edge) {
                            self.edges.push(edge.clone());
                        }
                    }
                }
            }
            GuiRuledSurfacePickTarget::Vector => {
                if let Some(vec) = &pick.vector {
                    self.vector = Some(vec.clone());
                }
            }
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons, picked_axis): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
            Option<WorkAxisReference>,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.synchronize(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, picked_axis);
        if let Some(preview) = &state.ruled_surface_preview
            && let Some(arrow) = preview.arrow
            && !self.preview_hidden
        {
            draw_direction_arrow(ui, arrow);
        }
        let height = ((area.bottom - area.top) - 16.0).clamp(440.0, 580.0);
        widget_abi::fit_viewport_properties(ui, PANEL_WINDOW_TITLE, area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, PANEL_WINDOW_TITLE, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                PANEL_WINDOW_TITLE,
                "##ruled-surface-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(160.0);
            ui.child_window("##ruled-surface-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_body(ui, state, actions);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn draw_body(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.draw_type_controls(ui, actions);
        ui.separator();
        self.draw_edge_controls(ui, actions);
        if self.ruled_type == RuledSurfaceType::Sweep {
            ui.separator();
            self.draw_vector_controls(ui, actions);
        }
        ui.separator();
        self.draw_dimension_controls(ui);
        if let Some(preview) = &state.ruled_surface_preview
            && let GuiRuledSurfacePreview::Failed(msg) = &preview.result
        {
            ui.spacing();
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::RuledSurface(
                    GuiRuledSurfaceAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::RuledSurface(GuiRuledSurfaceAction::Clear)),
            }
        }
    }
}

fn draw_direction_arrow(ui: &Ui, arrow: GuiRuledSurfaceArrow) {
    let (Some(origin_px), Some(tip_px)) = (arrow.origin_pixel, arrow.tip_pixel) else {
        return;
    };
    let scale = ui.io().display_framebuffer_scale;
    let sx = if scale[0] > 0.0 { scale[0] } else { 1.0 };
    let sy = if scale[1] > 0.0 { scale[1] } else { 1.0 };
    let origin = [origin_px[0] as f32 / sx, origin_px[1] as f32 / sy];
    let tip = [tip_px[0] as f32 / sx, tip_px[1] as f32 / sy];
    let color = if arrow.flipped {
        ARROW_FLIPPED_COLOR
    } else {
        ARROW_COLOR
    };
    let list = ui.get_background_draw_list();
    list.add_line(origin, tip, color).thickness(3.0).build();
    let dx = tip[0] - origin[0];
    let dy = tip[1] - origin[1];
    let len = dx.hypot(dy);
    if len > 4.0 {
        let ux = dx / len;
        let uy = dy / len;
        let px = -uy;
        let py = ux;
        let base = [tip[0] - ux * 11.0, tip[1] - uy * 11.0];
        let left = [base[0] + px * 5.5, base[1] + py * 5.5];
        let right = [base[0] - px * 5.5, base[1] - py * 5.5];
        list.add_triangle(tip, left, right, color)
            .filled(true)
            .build();
    }
}

pub(super) fn parse_distance_cm(raw: &str) -> Option<f64> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return None;
    }
    let lower = trimmed.to_ascii_lowercase();
    let (num_str, scale_to_cm) = if let Some(prefix) = lower.strip_suffix("mm") {
        (prefix.trim(), 0.1)
    } else if let Some(prefix) = lower.strip_suffix("cm") {
        (prefix.trim(), 1.0)
    } else if let Some(prefix) = lower.strip_suffix("in") {
        (prefix.trim(), 2.54)
    } else {
        (trimmed, 0.1)
    };
    let val = num_str.parse::<f64>().ok()?;
    val.is_finite().then_some(val * scale_to_cm)
}

pub(super) fn parse_angle_deg(raw: &str) -> Option<f64> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return None;
    }
    let lower = trimmed.to_ascii_lowercase();
    let num_str = lower
        .strip_suffix("deg")
        .or_else(|| lower.strip_suffix("degrees"))
        .or_else(|| lower.strip_suffix("degree"))
        .or_else(|| trimmed.strip_suffix('°'))
        .unwrap_or(trimmed)
        .trim();
    let val = num_str.parse::<f64>().ok()?;
    val.is_finite().then_some(val)
}

pub(super) fn format_mm_distance(cm: f64) -> String {
    let mm = cm * 10.0;
    if (mm - mm.round()).abs() < 1.0e-6 {
        format!("{} mm", mm.round() as i64)
    } else {
        let formatted = format!("{mm:.3}");
        let trimmed = formatted.trim_end_matches('0').trim_end_matches('.');
        format!("{trimmed} mm")
    }
}

pub(super) fn format_deg_angle(deg: f64) -> String {
    if (deg - deg.round()).abs() < 1.0e-6 {
        format!("{} deg", deg.round() as i64)
    } else {
        let formatted = format!("{deg:.3}");
        let trimmed = formatted.trim_end_matches('0').trim_end_matches('.');
        format!("{trimmed} deg")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn distance_and_angle_validation_match_ruled_surface_rules() {
        let mut dialog = RuledSurfaceDialog::default();
        assert_eq!(dialog.parsed_distance_cm(), Some(1.0));
        assert_eq!(dialog.parsed_angle_deg(), Some(0.0));
        dialog.distance_text = "0".into();
        assert!(dialog.parsed_distance_cm().is_none());
        dialog.distance_text = "-1".into();
        assert!(dialog.parsed_distance_cm().is_none());
        dialog.distance_text = "1".into();
        assert_eq!(dialog.parsed_distance_cm(), Some(0.1));
        dialog.distance_text = "20".into();
        assert_eq!(dialog.parsed_distance_cm(), Some(2.0));
        dialog.angle_text = "15".into();
        assert_eq!(dialog.parsed_angle_deg(), Some(15.0));
        dialog.angle_text = "180".into();
        assert_eq!(dialog.parsed_angle_deg(), Some(180.0));
        assert_eq!(format_deg_angle(180.0), "180 deg");
        dialog.angle_text = "181".into();
        assert!(dialog.parsed_angle_deg().is_none());
    }
}
