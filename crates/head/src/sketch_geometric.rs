// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadSketchEdit;
use oblikovati_gui::GuiSketchConstraint;
use oblikovati_model::sketch::SketchEntityId;
use std::collections::BTreeSet;

pub(crate) struct SketchGeometricState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    pub choice: GuiSketchConstraint,
    pub midpoint_lines: BTreeSet<SketchEntityId>,
}

use crate::sketch_operands as operands;
use oblikovati_api::{types::*, wire::AddConstraintArgs};
use oblikovati_model::sketch::Sketch2D;
use oblikovati_session::selection::SketchPick;
use operands::OperandCategory::{Circular, Line, Point};

pub(crate) fn accepts(choice: GuiSketchConstraint, kind: &SketchEntityKind) -> bool {
    matches!(
        (choice, operands::category(kind)),
        (_, Some(Point | Line)) | (GuiSketchConstraint::Coincident, Some(Circular))
    )
}
impl SketchGeometricState {
    pub(crate) fn prompt(&self) -> &'static str {
        match self.choice {
            GuiSketchConstraint::Horizontal | GuiSketchConstraint::Vertical => {
                "Select a line or two points. Escape cancels."
            }
            GuiSketchConstraint::Coincident => {
                "Select two points, or a point and a line, circle or arc. Escape cancels."
            }
        }
    }
    pub(crate) fn request(
        &self,
        sketch: &Sketch2D,
        picks: &[SketchPick],
        index: i64,
    ) -> Result<Option<AddConstraintArgs>, String> {
        let operands = operands::resolve(sketch, picks)?;
        let points = operands::ids(&operands, Point);
        let lines = operands::ids(&operands, Line);
        let circular = operands::ids(&operands, Circular);
        let recipe = match self.choice {
            GuiSketchConstraint::Horizontal | GuiSketchConstraint::Vertical => {
                self.alignment_recipe(&points, &lines)
            }
            GuiSketchConstraint::Coincident => self.coincident_recipe(&points, &lines, &circular),
        };
        Ok(recipe.map(|(kind, entities)| AddConstraintArgs {
            sketch_index: index,
            kind: kind.0.to_string(),
            entities: Some(entities),
            ..Default::default()
        }))
    }
    fn alignment_recipe(
        &self,
        points: &[SketchEntityId],
        lines: &[SketchEntityId],
    ) -> Option<(GeometricConstraintKind, Vec<u64>)> {
        let kind = match self.choice {
            GuiSketchConstraint::Horizontal => GEO_CONSTRAINT_HORIZONTAL,
            GuiSketchConstraint::Vertical => GEO_CONSTRAINT_VERTICAL,
            GuiSketchConstraint::Coincident => return None,
        };
        if let Some(line) = lines.first() {
            return Some((kind, vec![line.0]));
        }
        let [first, second, ..] = points else {
            return None;
        };
        Some((kind, vec![first.0, second.0]))
    }
    fn coincident_recipe(
        &self,
        points: &[SketchEntityId],
        lines: &[SketchEntityId],
        circular: &[SketchEntityId],
    ) -> Option<(GeometricConstraintKind, Vec<u64>)> {
        if let [first, second, ..] = points {
            return Some((GEO_CONSTRAINT_COINCIDENT, vec![first.0, second.0]));
        }
        let point = points.first()?;
        // Go applyCoincident gives a selected line priority over another circular operand.
        if let Some(line) = lines.first() {
            let kind = if self.midpoint_lines.contains(line) {
                GEO_CONSTRAINT_MIDPOINT
            } else {
                GEO_CONSTRAINT_POINT_ON_LINE
            };
            return Some((kind, vec![point.0, line.0]));
        }
        let circle = circular.first()?;
        Some((GEO_CONSTRAINT_POINT_ON_CIRCLE, vec![point.0, circle.0]))
    }
}

pub(crate) fn line_midpoint(
    sketch: &Sketch2D,
    entity: SketchEntityId,
) -> Result<Option<[f64; 2]>, String> {
    if sketch
        .entity_kind(entity)
        .map_err(|cause| cause.to_string())?
        != SKETCH_ENTITY_LINE
    {
        return Ok(None);
    }
    let line = sketch.line(entity).map_err(|cause| cause.to_string())?;
    let first = sketch
        .point(line.endpoints[0])
        .map_err(|cause| cause.to_string())?
        .position;
    let last = sketch
        .point(line.endpoints[1])
        .map_err(|cause| cause.to_string())?
        .position;
    Ok(Some(std::array::from_fn(|axis| {
        first[axis].midpoint(last[axis])
    })))
}
pub(crate) fn wire_sketch_index(
    sketches: &oblikovati_model::document_sketch::DocumentSketches,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
) -> Result<i64, String> {
    let index = sketches
        .sketches()
        .iter()
        .position(|placed| placed.id() == sketch)
        .ok_or_else(|| format!("constraint sketch {sketch:?}: expected a current sketch index"))?;
    i64::try_from(index).map_err(|_| format!("sketch index {index}: expected an i64 index"))
}
