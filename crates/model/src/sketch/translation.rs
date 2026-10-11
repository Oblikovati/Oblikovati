// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityId, SketchError, SketchPoint2D, entity_geometry,
    geometry::require_position,
};
use std::collections::BTreeSet;
impl Sketch2D {
    /// Translate each defining point once; prepare every finite output before changing the owner.
    /// Example: `let points = sketch.translate_entities(&[line, arc], [2.0, 3.0])?;`.
    pub fn translate_entities(
        &mut self,
        entities: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        require_position(vector)?;
        let selected = self.selected_defining_points(entities)?;
        let prepared = self.translated_points(&selected, vector)?;
        let mut candidate = self.clone();
        for point in prepared {
            candidate.set_position(point.id, point.position)?;
        }
        *self = candidate;
        Ok(selected)
    }
    fn selected_defining_points(
        &self,
        entities: &[SketchEntityId],
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let catalogue = entity_geometry::catalogue(self);
        let mut selected = BTreeSet::new();
        for id in entities {
            let shape = catalogue.get(id).ok_or_else(|| {
                SketchError::Definition(format!(
                    "translation entity {id:?}: expected an owned point, line, circle, arc, or ellipse"
                ))
            })?;
            selected.extend(shape.points());
        }
        Ok(self
            .entity_ids()
            .iter()
            .copied()
            .filter(|id| selected.contains(id))
            .collect())
    }
    fn translated_points(
        &self,
        selected: &[SketchEntityId],
        vector: [f64; 2],
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        selected
            .iter()
            .map(|id| {
                let position = translated_position(self.point(*id)?.position, vector);
                require_position(position).map_err(|cause| {
                    SketchError::Definition(format!(
                        "translation point {id:?} vector {vector:?}: {cause}"
                    ))
                })?;
                Ok(SketchPoint2D { id: *id, position })
            })
            .collect()
    }
}
fn translated_position(position: [f64; 2], vector: [f64; 2]) -> [f64; 2] {
    // #3770: preserve Go affine2.point arithmetic, including signed-zero behavior.
    [
        1.0 * position[0] + 0.0 * position[1] + vector[0],
        0.0 * position[0] + 1.0 * position[1] + vector[1],
    ]
}
