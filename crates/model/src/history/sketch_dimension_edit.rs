// SPDX-License-Identifier: GPL-2.0-only
use super::DocumentSnapshot;
use crate::{
    document_sketch::DocumentSketchId,
    sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchDimensionLimits, SketchEntityId,
        SketchError, SketchPoint2D,
    },
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
/// Optional authored changes; omitted fields preserve the existing dimension state.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct SketchDimensionEdit {
    pub target_expression: Option<String>,
    pub reference: Option<bool>,
    pub drive_limits: Option<SketchDimensionLimits>,
}
impl DocumentSnapshot {
    /// Edit one dimension on an isolated candidate; expression drive stores a clamped constant as in Go.
    /// Example: `snapshot.edit_sketch_dimension(sketch, dimension, &edit)?;` inside an edit.
    pub fn edit_sketch_dimension(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), SketchError> {
        let mut candidate = self.clone();
        candidate.apply_dimension_edit(sketch, dimension, edit)?;
        self.parameters = candidate.parameters;
        self.sketches = candidate.sketches;
        Ok(())
    }
    fn apply_dimension_edit(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        edit: &SketchDimensionEdit,
    ) -> Result<(), SketchError> {
        self.sketches
            .get(sketch)?
            .geometry()
            .dimension_source(dimension)?;
        if let Some(limits) = edit.drive_limits {
            self.sketches
                .set_dimension_limits(sketch, dimension, limits)?;
        }
        if let Some(reference) = edit.reference {
            self.sketches
                .set_dimension_reference(sketch, dimension, reference)?;
        }
        let Some(source) = &edit.target_expression else {
            return Ok(());
        };
        self.drive_dimension_target(sketch, dimension, source)?;
        self.hold_measured_start(sketch, dimension)
    }
    /// A distance grows from its first point, an arc's turn from its center and start, a radius
    /// from its center: those hold, and what hangs off the rest follows. With them held the
    /// sketch may have no solution; the ordinary recompute then decides.
    fn hold_measured_start(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
    ) -> Result<(), SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        let held = held_while_driven(geometry, dimension);
        if held.is_empty() {
            return Ok(());
        }
        let mut anchors = self.resolve_sketch_projected_points(sketch)?;
        for point in held {
            if !anchors.iter().any(|anchor| anchor.id == point) {
                anchors.push(*geometry.point(point)?);
            }
        }
        let value = self.driven_value(sketch, dimension)?;
        for end in value.map_or(Vec::new(), |value| {
            arc_ends_at_radius(geometry, dimension, value)
        }) {
            if !anchors.iter().any(|anchor| anchor.id == end.id) {
                anchors.push(end);
            }
        }
        let solved = geometry.solve_with_grounded_points(
            &self.parameters,
            &anchors,
            SolveOptions::default(),
            || false,
        );
        let Ok(outcome) = solved else {
            return Ok(());
        };
        if outcome.numerical().termination != SolveTermination::Converged {
            return Ok(());
        }
        let candidate = outcome.candidate().clone();
        self.sketches.replace_geometry(sketch, candidate)
    }
    /// The value a just-driven dimension now holds; example: an arc's new radius.
    fn driven_value(
        &self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
    ) -> Result<Option<f64>, SketchError> {
        let (parameter, _) = self
            .sketches
            .get(sketch)?
            .geometry()
            .dimension_source(dimension)?;
        Ok(self
            .parameters
            .get(parameter)
            .and_then(|state| state.current_value())
            .map(|value| value.value))
    }
    fn drive_dimension_target(
        &mut self,
        sketch: DocumentSketchId,
        dimension: SketchConstraintId,
        source: &str,
    ) -> Result<(), SketchError> {
        let geometry = self.sketches.get(sketch)?.geometry();
        let (parameter, category) = geometry.dimension_source(dimension)?;
        self.parameters
            .validate_owned_scalar(parameter)
            .map_err(|cause| drive_error(parameter, source, cause))?;
        let mut target = self
            .parameters
            .evaluate_scalar_expression(source, category)
            .map_err(|cause| drive_error(parameter, source, cause))?;
        if let Some(limits) = geometry.dimension_limits(dimension)? {
            target.value = limits.clamp(target.value);
        }
        self.parameters
            .set_value(parameter, target)
            .map_err(|cause| drive_error(parameter, source, cause))
    }
}
fn held_while_driven(geometry: &Sketch2D, dimension: SketchConstraintId) -> Vec<SketchEntityId> {
    let Some(constraint) = geometry
        .constraints()
        .iter()
        .find(|known| known.id == dimension)
    else {
        return Vec::new();
    };
    let center_of = |curve: SketchEntityId| {
        geometry
            .circle(curve)
            .map(|circle| circle.center)
            .or_else(|_| geometry.arc(curve).map(|arc| arc.center))
            .map_or(Vec::new(), |center| vec![center])
    };
    match constraint.relation {
        PlanarConstraint::Distance { points, .. } => vec![points[0]],
        PlanarConstraint::Radius { curve, .. } | PlanarConstraint::Diameter { curve, .. } => {
            center_of(curve)
        }
        PlanarConstraint::ArcSweep { arc, .. } => geometry
            .arc(arc)
            .map_or(Vec::new(), |arc| vec![arc.center, arc.start]),
        _ => Vec::new(),
    }
}
/// An arc keeps its turn as its size changes: both ends slide out along their own radii.
fn arc_ends_at_radius(
    geometry: &Sketch2D,
    dimension: SketchConstraintId,
    value: f64,
) -> Vec<SketchPoint2D> {
    let relation = geometry
        .constraints()
        .iter()
        .find(|known| known.id == dimension)
        .map(|constraint| &constraint.relation);
    let (curve, radius) = match relation {
        Some(PlanarConstraint::Radius { curve, .. }) => (*curve, value),
        Some(PlanarConstraint::Diameter { curve, .. }) => (*curve, value / 2.),
        _ => return Vec::new(),
    };
    let Ok(arc) = geometry.arc(curve) else {
        return Vec::new();
    };
    let Ok(center) = geometry.point(arc.center).map(|point| point.position) else {
        return Vec::new();
    };
    let slid = |id| {
        let at = geometry.point(id).ok()?.position;
        let reach = (at[0] - center[0]).hypot(at[1] - center[1]);
        (reach > 0.).then(|| SketchPoint2D {
            id,
            position: std::array::from_fn(|axis| {
                center[axis] + (at[axis] - center[axis]) * radius / reach
            }),
        })
    };
    [arc.start, arc.end].into_iter().filter_map(slid).collect()
}
fn drive_error(parameter: u64, source: &str, cause: impl std::fmt::Display) -> SketchError {
    SketchError::Definition(format!(
        "dimension parameter {parameter} drive expression {source:?}: expected a finite compatible owned Model scalar target: {cause}"
    ))
}
