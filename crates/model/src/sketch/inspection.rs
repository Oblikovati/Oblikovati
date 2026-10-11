// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchEntityId, SketchError, SketchPoint2D, grounded};
use crate::parameter::ParameterCollection;
use oblikovati_api::types::{
    GeometryMoveableStatus, MOVEABLE_BY_DIMENSION_CHANGE, MOVEABLE_FIXED, MOVEABLE_FREE,
    SKETCH_ENTITY_ARC, SKETCH_ENTITY_CIRCLE, SKETCH_ENTITY_LINE, SKETCH_ENTITY_POINT,
    SketchEntityKind,
};
use oblikovati_solve::{
    diagnostics::coordinate_freedoms,
    nonlinear::{SolveError, SolveOptions},
    residual::evaluate_residuals,
};
use std::collections::{BTreeMap, BTreeSet};
mod geometry;
use geometry::InspectionGeometry;
/// Current analytic shape and movement classification in document working units.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchEntityInspection {
    pub id: SketchEntityId,
    pub kind: SketchEntityKind,
    pub points: Vec<[f64; 2]>,
    pub radius: f64,
    pub movement: GeometryMoveableStatus,
    pub construction: bool,
}
/// Separate rank conditioning and squared local-freedom policy.
#[derive(Clone, Copy, Debug)]
pub struct SketchMovementOptions {
    pub rank_tolerance: f64,
    pub squared_freedom_tolerance: f64,
}
impl Default for SketchMovementOptions {
    fn default() -> Self {
        Self {
            rank_tolerance: SolveOptions::default().rank_tolerance,
            // Go #1417 retains this squared projection policy after analytic derivatives replaced differences.
            squared_freedom_tolerance: 1e-6,
        }
    }
}
impl Sketch2D {
    /// Inspect all entities against one current Jacobian, with grounded references held exact.
    /// Example: `sketch.inspect_entities(&parameters, &anchors, Default::default())?;`.
    pub fn inspect_entities(
        &self,
        parameters: &ParameterCollection<u64>,
        anchors: &[SketchPoint2D],
        options: SketchMovementOptions,
    ) -> Result<Vec<SketchEntityInspection>, SketchError> {
        let prepared = grounded::prepare(self, parameters, anchors)?;
        let evaluated = evaluate_residuals(&prepared.values, &prepared.blocks())
            .map_err(|cause| super::source::source_failure(&prepared.origins, cause.into()))?;
        let free = coordinate_freedoms(
            &evaluated,
            options.rank_tolerance,
            options.squared_freedom_tolerance,
        )
        .map_err(SolveError::from)?;
        let seed = self.candidate_values(&prepared.restore(&prepared.values)?)?;
        let free = prepared.restore_freedoms(&free)?;
        let fixed = fixed_points(self, anchors);
        inspect_in_order(&seed, &free, &fixed)
    }
}
fn inspect_in_order(
    sketch: &Sketch2D,
    free: &[bool],
    fixed: &BTreeSet<SketchEntityId>,
) -> Result<Vec<SketchEntityInspection>, SketchError> {
    let shapes = geometry::catalogue(sketch);
    sketch
        .entity_ids()
        .iter()
        .map(|id| {
            let shape = shapes.get(id).ok_or_else(|| {
                SketchError::Definition(format!(
                    "sketch entity {id:?}: expected an owned point, line, circle, or arc"
                ))
            })?;
            inspect_shape(sketch, *shape, free, fixed)
        })
        .collect()
}
fn fixed_points(sketch: &Sketch2D, anchors: &[SketchPoint2D]) -> BTreeSet<SketchEntityId> {
    sketch
        .constraints()
        .iter()
        .filter_map(|constraint| match constraint.relation {
            PlanarConstraint::Fix { point, .. } => Some(point),
            _ => None,
        })
        .chain(anchors.iter().map(|point| point.id))
        .collect()
}
fn inspect_shape(
    sketch: &Sketch2D,
    shape: InspectionGeometry,
    free: &[bool],
    fixed: &BTreeSet<SketchEntityId>,
) -> Result<SketchEntityInspection, SketchError> {
    let point_ids = shape.points();
    let columns = shape.columns(sketch)?;
    let points = point_ids
        .iter()
        .map(|id| Ok(sketch.point(*id)?.position))
        .collect::<Result<_, SketchError>>()?;
    Ok(SketchEntityInspection {
        id: shape.id(),
        kind: shape.kind(),
        points,
        radius: shape.radius(sketch)?,
        movement: movement(&columns, &point_ids, free, fixed),
        construction: sketch.is_construction(shape.id())?,
    })
}
fn movement(
    columns: &[usize],
    points: &[SketchEntityId],
    free: &[bool],
    fixed: &BTreeSet<SketchEntityId>,
) -> GeometryMoveableStatus {
    if columns.iter().any(|column| free[*column]) {
        return MOVEABLE_FREE;
    }
    if points.iter().any(|point| fixed.contains(point)) {
        return MOVEABLE_FIXED;
    }
    MOVEABLE_BY_DIMENSION_CHANGE
}
