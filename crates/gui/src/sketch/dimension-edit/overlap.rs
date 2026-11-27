// SPDX-License-Identifier: GPL-2.0-only
//! Resolve text and line hits against the same screen positions used for drawing.
use super::GuiDimensionEdit;
use crate::{GuiCommandState, GuiDimensionLabel, dimension_labels::LabelPosition};
use imgui::Ui;

pub(super) enum DimensionHit {
    One(GuiDimensionEdit),
    Multiple(Vec<GuiDimensionEdit>),
}

pub(super) fn dimension_hits(
    mouse: [f32; 2],
    labels: &[GuiDimensionLabel],
    positions: &[Option<LabelPosition>],
    scale: [f32; 2],
) -> Option<DimensionHit> {
    let text = text_hits(mouse, labels, positions);
    match text.len() {
        0 => nearest_line(mouse, labels, positions, scale).map(DimensionHit::One),
        1 => text.into_iter().next().map(DimensionHit::One),
        _ => Some(DimensionHit::Multiple(text)),
    }
}
fn text_hits(
    mouse: [f32; 2],
    labels: &[GuiDimensionLabel],
    positions: &[Option<LabelPosition>],
) -> Vec<GuiDimensionEdit> {
    labels
        .iter()
        .zip(positions)
        .filter_map(|(label, position)| {
            let position = position.as_ref()?;
            if position.contains(mouse) {
                label.edit.clone()
            } else {
                None
            }
        })
        .collect()
}
fn nearest_line(
    mouse: [f32; 2],
    labels: &[GuiDimensionLabel],
    positions: &[Option<LabelPosition>],
    scale: [f32; 2],
) -> Option<GuiDimensionEdit> {
    labels
        .iter()
        .zip(positions)
        .enumerate()
        .filter_map(|(index, (label, position))| {
            let position = position.as_ref()?;
            let edit = label.edit.as_ref()?;
            let distance = dimension_line_distance(mouse, label, position, scale)?;
            (distance <= 5.).then(|| (distance, index, edit.clone()))
        })
        .min_by(|a, b| a.0.total_cmp(&b.0).then_with(|| b.1.cmp(&a.1)))
        .map(|(_, _, edit)| edit)
}
fn dimension_line_distance(
    mouse: [f32; 2],
    label: &GuiDimensionLabel,
    position: &LabelPosition,
    scale: [f32; 2],
) -> Option<f32> {
    let leader = (position.anchor != position.center)
        .then(|| line_distance(mouse, position.anchor, position.center));
    let dimension = label.line.map(|[a, b]| {
        let point = |p: [i32; 2]| [p[0] as f32 / scale[0], p[1] as f32 / scale[1]];
        line_distance(mouse, point(a), point(b))
    });
    match (leader, dimension) {
        (Some(a), Some(b)) => Some(a.min(b)),
        (Some(a), None) | (None, Some(a)) => Some(a),
        (None, None) => None,
    }
}
pub(super) fn line_distance(point: [f32; 2], from: [f32; 2], to: [f32; 2]) -> f32 {
    let direction = [to[0] - from[0], to[1] - from[1]];
    let length_squared = direction[0] * direction[0] + direction[1] * direction[1];
    if length_squared == 0. {
        return (point[0] - from[0]).hypot(point[1] - from[1]);
    }
    let along = (((point[0] - from[0]) * direction[0] + (point[1] - from[1]) * direction[1])
        / length_squared)
        .clamp(0., 1.);
    (point[0] - from[0] - along * direction[0]).hypot(point[1] - from[1] - along * direction[1])
}

#[derive(Default)]
pub(super) struct DimensionChooser {
    choices: Vec<GuiDimensionEdit>,
    requested: bool,
    position: [f32; 2],
    context: Option<(u64, u64)>,
}
impl DimensionChooser {
    pub(super) fn is_open(&self) -> bool {
        self.requested || !self.choices.is_empty()
    }
    pub(super) fn request(
        &mut self,
        choices: Vec<GuiDimensionEdit>,
        position: [f32; 2],
        state: &GuiCommandState,
    ) {
        self.choices = choices;
        self.position = position;
        self.context = dimension_context(state);
        self.requested = true;
    }
    pub(super) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
    ) -> Option<(GuiDimensionEdit, [f32; 2])> {
        const POPUP: &str = "Choose Dimension";
        if !self.valid_for(state) {
            self.choices.clear();
            self.requested = false;
            if let Some(_popup) = ui.begin_popup(POPUP) {
                ui.close_current_popup();
            }
            return None;
        }
        if std::mem::take(&mut self.requested) {
            ui.open_popup(POPUP);
        }
        let Some(_popup) = ui.begin_popup(POPUP) else {
            self.choices.clear();
            return None;
        };
        let mut selected = None;
        for edit in &self.choices {
            let label = format!(
                "{}: {}##dimension-{}-{}",
                edit.name, edit.expression, edit.sketch, edit.constraint
            );
            if ui.selectable_config(&label).build() {
                ui.close_current_popup();
                selected = Some((edit.clone(), self.position));
                break;
            }
        }
        if selected.is_some() {
            self.choices.clear();
        }
        selected
    }
    fn valid_for(&self, state: &GuiCommandState) -> bool {
        !self.choices.is_empty()
            && self.context.is_some()
            && self.context == dimension_context(state)
            && (state.sketch_at_ready || state.dimension_tool_active)
            && choices_are_current(&self.choices, &state.dimension_labels)
    }
}
fn choices_are_current(choices: &[GuiDimensionEdit], labels: &[GuiDimensionLabel]) -> bool {
    let mut current = labels.iter().filter_map(|label| label.edit.as_ref());
    choices
        .iter()
        .all(|edit| current.by_ref().any(|row| row == edit))
}
fn dimension_context(state: &GuiCommandState) -> Option<(u64, u64)> {
    state
        .document
        .zip(state.active_sketch.map(|sketch| sketch.0))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::GuiDimensionLabelRole;

    fn label(id: u64, line: Option<[[i32; 2]; 2]>) -> GuiDimensionLabel {
        GuiDimensionLabel {
            pixel: [100, 100],
            line,
            witnesses: None,
            unit: None,
            text: id.to_string(),
            role: GuiDimensionLabelRole::Driving,
            edit: Some(GuiDimensionEdit {
                sketch: 1,
                constraint: id,
                name: format!("d{id}"),
                expression: id.to_string(),
                driven: false,
            }),
        }
    }
    fn positioned(center: [f32; 2]) -> Option<LabelPosition> {
        Some(LabelPosition {
            center,
            size: [20., 10.],
            anchor: center,
        })
    }
    #[test]
    fn text_hit_takes_priority_over_a_crossing_dimension_line() {
        let labels = [label(1, None), label(2, Some([[90, 100], [110, 100]]))];
        let positions = [positioned([100., 100.]), positioned([200., 200.])];
        let Some(DimensionHit::One(edit)) =
            dimension_hits([100., 100.], &labels, &positions, [1., 1.])
        else {
            panic!("expected one text hit")
        };
        assert_eq!(edit.constraint, 1);
    }
    #[test]
    fn overlapping_text_offers_every_matching_dimension_in_order() {
        let labels = [label(1, None), label(2, None)];
        let positions = [positioned([100., 100.]), positioned([100., 100.])];
        let Some(DimensionHit::Multiple(edits)) =
            dimension_hits([100., 100.], &labels, &positions, [1., 1.])
        else {
            panic!("expected an ambiguity")
        };
        assert_eq!(
            edits.iter().map(|edit| edit.constraint).collect::<Vec<_>>(),
            [1, 2]
        );
    }
    #[test]
    fn displaced_text_and_its_leader_hit_the_same_dimension() {
        let labels = [label(1, None)];
        let positions = [Some(LabelPosition {
            center: [150., 100.],
            size: [20., 10.],
            anchor: [100., 100.],
        })];
        for point in [[150., 100.], [125., 100.]] {
            let Some(DimensionHit::One(edit)) =
                dimension_hits(point, &labels, &positions, [1., 1.])
            else {
                panic!("expected the displaced label at {point:?}")
            };
            assert_eq!(edit.constraint, 1);
        }
    }
    #[test]
    fn offscreen_label_does_not_accept_line_hits() {
        let labels = [label(1, Some([[90, 100], [110, 100]]))];
        assert!(dimension_hits([100., 100.], &labels, &[None], [1., 1.]).is_none());
    }

    #[test]
    fn chooser_rejects_changed_document_or_removed_dimension() {
        let labels = [label(1, None), label(2, None)];
        let mut state = GuiCommandState {
            document: Some(7),
            active_sketch: Some(oblikovati_model::document_sketch::DocumentSketchId(1)),
            sketch_at_ready: true,
            dimension_labels: labels.to_vec(),
            ..Default::default()
        };
        let mut chooser = DimensionChooser::default();
        let choices = labels
            .iter()
            .map(|label| label.edit.clone().unwrap())
            .collect();
        chooser.request(choices, [0., 0.], &state);
        assert!(chooser.valid_for(&state));
        state.document = Some(8);
        assert!(!chooser.valid_for(&state));
        state.document = Some(7);
        state.dimension_labels[0].edit.as_mut().unwrap().expression = "changed".into();
        assert!(!chooser.valid_for(&state));
        state.dimension_labels[0].edit = labels[0].edit.clone();
        state.dimension_labels.pop();
        assert!(!chooser.valid_for(&state));
    }
}
