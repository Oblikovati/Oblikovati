// SPDX-License-Identifier: GPL-2.0-only
//! Convert a block placement to independent geometry as one document edit.
use super::{DocumentSnapshot, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        SketchCopyError, SketchCopyReport, SketchEntityId, SketchError, SketchSelectionTransform,
    },
};

impl DocumentSnapshot {
    /// Replace one instance with its placed members; example: `snapshot.explode_sketch_block(sketch, instance, reservations)?`.
    pub fn explode_sketch_block(
        &mut self,
        sketch: DocumentSketchId,
        instance_id: SketchEntityId,
        reservations: &mut RecipeReservations<'_>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let placed = self.sketches.get(sketch).map_err(SketchCopyError::Sketch)?;
        let instance = placed.block_instances().iter().find(|item| item.id() == instance_id).ok_or_else(|| {
            SketchCopyError::Sketch(SketchError::Definition(format!("block instance {instance_id:?} in sketch {sketch:?}: expected a current placed instance")))
        })?;
        let definition = self
            .sketches
            .block_definitions()
            .iter()
            .find(|item| item.name() == instance.definition())
            .ok_or_else(|| {
                SketchCopyError::Sketch(SketchError::Definition(format!(
                    "block instance {instance_id:?} definition {:?}: expected a current definition",
                    instance.definition()
                )))
            })?;
        let source = definition.geometry();
        let transform = SketchSelectionTransform::Similarity {
            position: instance.position(),
            radians: instance.rotation(),
            scale: instance.scale(),
        };
        let mut geometry = placed.geometry().clone();
        let report = geometry.copy_transformed_selection(
            source,
            source.entity_ids(),
            transform,
            &mut || reservations.allocate_sketch_entity(),
        )?;
        if let Some(color) = definition.color() {
            apply_definition_color(&mut geometry, &report, color)
                .map_err(SketchCopyError::Sketch)?;
        }
        let mut candidate = self.clone();
        candidate
            .sketches
            .remove_block_instances(sketch, &[instance_id])
            .map_err(SketchCopyError::Sketch)?;
        candidate
            .sketches
            .replace_geometry(sketch, geometry)
            .map_err(SketchCopyError::Sketch)?;
        *self = candidate;
        Ok(report)
    }
}

fn apply_definition_color(
    geometry: &mut crate::sketch::Sketch2D,
    report: &SketchCopyReport,
    color: &oblikovati_api::types::Color,
) -> Result<(), SketchError> {
    for entity in &report.copies {
        let mut format = geometry
            .entity_format(*entity)?
            .cloned()
            .unwrap_or_default();
        if format.override_color.source == oblikovati_api::types::OVERRIDE_COLOR_SOURCE {
            continue;
        }
        format.override_color = color.clone();
        geometry.set_entity_format(*entity, format)?;
    }
    Ok(())
}
