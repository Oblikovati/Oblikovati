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
    match choice {
        GuiSketchConstraint::Parallel
        | GuiSketchConstraint::Perpendicular
        | GuiSketchConstraint::Collinear => operands::category(kind) == Some(Line),
        GuiSketchConstraint::Concentric => operands::category(kind) == Some(Circular),
        GuiSketchConstraint::Equal | GuiSketchConstraint::Tangent => {
            matches!(operands::category(kind), Some(Line | Circular))
        }
        _ => matches!(
            (choice, operands::category(kind)),
            (_, Some(Point | Line)) | (GuiSketchConstraint::Coincident, Some(Circular))
        ),
    }
}
impl SketchGeometricState {
    pub(crate) fn prompt(&self) -> &'static str {
        match self.choice {
            GuiSketchConstraint::Horizontal | GuiSketchConstraint::Vertical => {
                "Select a line or two points. Escape cancels."
            }
            GuiSketchConstraint::Parallel
            | GuiSketchConstraint::Perpendicular
            | GuiSketchConstraint::Collinear => "Select two lines. Escape cancels.",
            GuiSketchConstraint::Equal => "Select two lines, or two circles/arcs. Escape cancels.",
            GuiSketchConstraint::Concentric => "Select two circles or arcs. Escape cancels.",
            GuiSketchConstraint::Tangent => {
                "Select a line and circle/arc, or two circles/arcs. Escape cancels."
            }
            GuiSketchConstraint::Fix => "Select a point or line to fix. Escape cancels.",
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
    ) -> Result<Vec<AddConstraintArgs>, String> {
        let operands = operands::resolve(sketch, picks)?;
        let points = operands::ids(&operands, Point);
        let lines = operands::ids(&operands, Line);
        let circular = operands::ids(&operands, Circular);
        if self.choice == GuiSketchConstraint::Fix {
            return fixed_point_requests(sketch, &points, &lines, index);
        }
        let recipe = match self.choice {
            GuiSketchConstraint::Horizontal | GuiSketchConstraint::Vertical => {
                self.alignment_recipe(&points, &lines)
            }
            GuiSketchConstraint::Coincident => self.coincident_recipe(&points, &lines, &circular),
            GuiSketchConstraint::Parallel
            | GuiSketchConstraint::Perpendicular
            | GuiSketchConstraint::Collinear => self.line_pair_recipe(&lines),
            GuiSketchConstraint::Equal
            | GuiSketchConstraint::Concentric
            | GuiSketchConstraint::Tangent => curve_relation_recipe(self.choice, &lines, &circular),
            GuiSketchConstraint::Fix => None,
        };
        Ok(recipe
            .into_iter()
            .map(|(kind, entities)| AddConstraintArgs {
                sketch_index: index,
                kind: kind.0.to_string(),
                entities: Some(entities),
                ..Default::default()
            })
            .collect())
    }
    fn alignment_recipe(
        &self,
        points: &[SketchEntityId],
        lines: &[SketchEntityId],
    ) -> Option<(GeometricConstraintKind, Vec<u64>)> {
        let kind = match self.choice {
            GuiSketchConstraint::Horizontal => GEO_CONSTRAINT_HORIZONTAL,
            GuiSketchConstraint::Vertical => GEO_CONSTRAINT_VERTICAL,
            _ => return None,
        };
        if let Some(line) = lines.first() {
            return Some((kind, vec![line.0]));
        }
        let [first, second, ..] = points else {
            return None;
        };
        Some((kind, vec![first.0, second.0]))
    }
    fn line_pair_recipe(
        &self,
        lines: &[SketchEntityId],
    ) -> Option<(GeometricConstraintKind, Vec<u64>)> {
        let kind = match self.choice {
            GuiSketchConstraint::Parallel => GEO_CONSTRAINT_PARALLEL,
            GuiSketchConstraint::Perpendicular => GEO_CONSTRAINT_PERPENDICULAR,
            GuiSketchConstraint::Collinear => GEO_CONSTRAINT_COLLINEAR,
            _ => return None,
        };
        pair_recipe(kind, lines)
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

fn fixed_point_requests(
    sketch: &Sketch2D,
    points: &[SketchEntityId],
    lines: &[SketchEntityId],
    index: i64,
) -> Result<Vec<AddConstraintArgs>, String> {
    let mut fixed = points.to_vec();
    for line in lines {
        fixed.extend(
            sketch
                .line(*line)
                .map_err(|cause| cause.to_string())?
                .endpoints,
        );
    }
    let mut seen = BTreeSet::new();
    Ok(fixed
        .into_iter()
        .filter(|point| seen.insert(*point))
        .map(|point| AddConstraintArgs {
            sketch_index: index,
            kind: GEO_CONSTRAINT_FIX.0.to_string(),
            entities: Some(vec![point.0]),
            ..Default::default()
        })
        .collect())
}

fn pair_recipe(
    kind: GeometricConstraintKind,
    entities: &[SketchEntityId],
) -> Option<(GeometricConstraintKind, Vec<u64>)> {
    let [first, second, ..] = entities else {
        return None;
    };
    Some((kind, vec![first.0, second.0]))
}
fn curve_relation_recipe(
    choice: GuiSketchConstraint,
    lines: &[SketchEntityId],
    circular: &[SketchEntityId],
) -> Option<(GeometricConstraintKind, Vec<u64>)> {
    match choice {
        GuiSketchConstraint::Equal if lines.len() >= 2 => {
            pair_recipe(GEO_CONSTRAINT_EQUAL_LENGTH, lines)
        }
        GuiSketchConstraint::Equal => pair_recipe(GEO_CONSTRAINT_EQUAL_RADIUS, circular),
        GuiSketchConstraint::Concentric => pair_recipe(GEO_CONSTRAINT_CONCENTRIC, circular),
        GuiSketchConstraint::Tangent if !lines.is_empty() && !circular.is_empty() => {
            Some((GEO_CONSTRAINT_TANGENT, vec![lines[0].0, circular[0].0]))
        }
        GuiSketchConstraint::Tangent => pair_recipe(GEO_CONSTRAINT_TANGENT, circular),
        _ => None,
    }
}
