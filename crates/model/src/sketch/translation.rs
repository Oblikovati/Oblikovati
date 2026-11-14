// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchEntityId, SketchError, SketchPoint2D, SketchSelectionTransform,
    entity_geometry, geometry::require_position, selection_transform::SelectionAffine,
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
        self.transform_entities(entities, SketchSelectionTransform::Move(vector))
    }
    /// Transform unique defining points and selected intrinsic axes atomically.
    /// Example: `sketch.transform_entities(&[line], SketchSelectionTransform::Rotate { center: [0.0; 2], radians: 1.0 })?;`.
    pub fn transform_entities(
        &mut self,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let affine = transform.affine()?;
        let selected = self.selected_defining_points(entities, transform)?;
        let prepared = self.transformed_points(&selected, &affine, transform)?;
        let mut candidate = self.clone();
        for point in prepared {
            candidate.set_position(point.id, point.position)?;
        }
        candidate.transform_selected_frames(entities, &affine, transform)?;
        *self = candidate;
        Ok(selected)
    }
    fn selected_defining_points(
        &self,
        entities: &[SketchEntityId],
        transform: SketchSelectionTransform,
    ) -> Result<Vec<SketchEntityId>, SketchError> {
        let catalogue = entity_geometry::catalogue(self);
        let mut selected = BTreeSet::new();
        for id in entities {
            let shape = catalogue
                .get(id)
                .ok_or_else(|| selection_geometry_error(*id))?;
            selected.extend(shape.transform_points(transform)?);
        }
        // #3563: constructor-owned defining points are hidden from the visible entity list.
        Ok(self
            .owned_entity_ids()
            .into_iter()
            .filter(|id| selected.contains(id))
            .collect())
    }
    fn transformed_points(
        &self,
        selected: &[SketchEntityId],
        affine: &SelectionAffine,
        transform: SketchSelectionTransform,
    ) -> Result<Vec<SketchPoint2D>, SketchError> {
        selected
            .iter()
            .map(|id| {
                let position = affine.point(self.point(*id)?.position);
                require_position(position).map_err(|cause| {
                    SketchError::Definition(format!(
                        "transform point {id:?} {transform:?}: {cause}"
                    ))
                })?;
                Ok(SketchPoint2D { id: *id, position })
            })
            .collect()
    }
    fn transform_selected_frames(
        &mut self,
        entities: &[SketchEntityId],
        affine: &SelectionAffine,
        transform: SketchSelectionTransform,
    ) -> Result<(), SketchError> {
        if matches!(transform, SketchSelectionTransform::Move(_)) {
            return Ok(());
        }
        let selected: BTreeSet<_> = entities.iter().copied().collect();
        self.transform_selected_axes(&selected, affine)?;
        if matches!(transform, SketchSelectionTransform::Reflect { .. }) {
            self.reverse_selected_arcs(&selected);
        }
        Ok(())
    }
    fn transform_selected_axes(
        &mut self,
        selected: &BTreeSet<SketchEntityId>,
        affine: &SelectionAffine,
    ) -> Result<(), SketchError> {
        for ellipse in &mut self.ellipses {
            if selected.contains(&ellipse.id()) {
                *ellipse = ellipse.with_transformed_axis(affine.direction(ellipse.major_axis()))?;
            }
        }
        Ok(())
    }
    fn reverse_selected_arcs(&mut self, selected: &BTreeSet<SketchEntityId>) {
        // #3563: reflection reverses orientation; retaining the source flag selects the other sweep.
        for arc in &mut self.arcs {
            if selected.contains(&arc.id) {
                arc.counter_clockwise = !arc.counter_clockwise;
            }
        }
    }
}

fn selection_geometry_error(id: SketchEntityId) -> SketchError {
    SketchError::Definition(format!(
        "transform entity {id:?}: expected an owned sketch entity with defining points"
    ))
}
