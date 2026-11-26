// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingAnnotationDefinition, DrawingDocument, DrawingError};

/// Sheet-space text and leader geometry regenerated from a drawing note.
pub struct DrawingNotePresentation<'a> {
    pub text: &'a str,
    pub anchor_mm: [f64; 2],
    pub leader_segments_mm: Vec<[[f64; 2]; 2]>,
}

impl DrawingAnnotationDefinition {
    /// Resolve independent sheet markup, e.g. `annotation.note_presentation()`.
    pub fn note_presentation(&self) -> Option<DrawingNotePresentation<'_>> {
        let Self::Note {
            x_mm,
            y_mm,
            leader_x_mm,
            leader_y_mm,
            text,
            ..
        } = self
        else {
            return None;
        };
        let anchor_mm = [*x_mm, *y_mm];
        Some(DrawingNotePresentation {
            text,
            anchor_mm,
            leader_segments_mm: leader_segments(anchor_mm, [*leader_x_mm, *leader_y_mm]),
        })
    }
}

impl DrawingDocument {
    /// Place sheet text and return its definition, e.g. `drawing.add_note("Note", [30., 50.], "Deburr", [0., 0.])?`.
    pub fn add_note(
        &mut self,
        requested_name: &str,
        [x_mm, y_mm]: [f64; 2],
        text: &str,
        [leader_x_mm, leader_y_mm]: [f64; 2],
    ) -> Result<DrawingAnnotationDefinition, DrawingError> {
        let sheet = self.active_sheet();
        let name = unique_name(requested_name, &self.sheets()[sheet].annotations);
        let note = DrawingAnnotationDefinition::Note {
            name,
            x_mm,
            y_mm,
            leader_x_mm,
            leader_y_mm,
            text: text.into(),
            characteristic: None,
            material_removal: None,
        };
        self.add_annotation_definition(sheet, note.clone())?;
        Ok(note)
    }
}

fn unique_name(requested: &str, annotations: &[DrawingAnnotationDefinition]) -> String {
    let exists = |name: &str| {
        annotations
            .iter()
            .any(|annotation| annotation.name() == name)
    };
    if !requested.is_empty() && !exists(requested) {
        return requested.into();
    }
    let mut number = annotations.len() + 1;
    loop {
        let name = format!("NOTE:{number}");
        if !exists(&name) {
            return name;
        }
        number += 1;
    }
}

fn leader_segments(anchor: [f64; 2], target: [f64; 2]) -> Vec<[[f64; 2]; 2]> {
    if target == [0., 0.] {
        return Vec::new();
    }
    let [dx, dy] = [anchor[0] - target[0], anchor[1] - target[1]];
    let distance = dx.hypot(dy);
    let mut segments = vec![[anchor, target]];
    if distance >= 1e-9 && distance.is_finite() {
        let [ux, uy] = [dx / distance, dy / distance];
        let [x, y] = [target[0] + 3. * ux, target[1] + 3. * uy];
        segments.extend([
            [target, [x - 1.2 * uy, y + 1.2 * ux]],
            [target, [x + 1.2 * uy, y - 1.2 * ux]],
        ]);
    }
    segments
}
