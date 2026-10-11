// SPDX-License-Identifier: GPL-2.0-only
use PlanarConstraint::*;
use oblikovati_api::{
    Error,
    types::{
        GEO_CONSTRAINT_COINCIDENT as COINCIDENT, GEO_CONSTRAINT_COLLINEAR as COLLINEAR,
        GEO_CONSTRAINT_CONCENTRIC as CONCENTRIC, GEO_CONSTRAINT_EQUAL_LENGTH as EQUAL_LENGTH,
        GEO_CONSTRAINT_EQUAL_RADIUS as EQUAL_RADIUS, GEO_CONSTRAINT_FIX as FIX,
        GEO_CONSTRAINT_HORIZONTAL as HORIZONTAL,
        GEO_CONSTRAINT_HORIZONTAL_ALIGN as HORIZONTAL_ALIGN, GEO_CONSTRAINT_MIDPOINT as MIDPOINT,
        GEO_CONSTRAINT_PARALLEL as PARALLEL, GEO_CONSTRAINT_PERPENDICULAR as PERPENDICULAR,
        GEO_CONSTRAINT_POINT_ON_CIRCLE as POINT_ON_CIRCLE,
        GEO_CONSTRAINT_POINT_ON_LINE as POINT_ON_LINE, GEO_CONSTRAINT_TANGENT as TANGENT,
        GEO_CONSTRAINT_VERTICAL as VERTICAL, GEO_CONSTRAINT_VERTICAL_ALIGN as VERTICAL_ALIGN,
        GeometricConstraintKind,
    },
};
use oblikovati_model::sketch::{LineRelation, PlanarConstraint, SketchEntityId};
type ConstraintShape = (GeometricConstraintKind, Vec<SketchEntityId>);
pub(in crate::sketches) fn require(relation: &PlanarConstraint) -> Result<ConstraintShape, Error> {
    classify(relation).ok_or_else(|| Error(format!(
        "geometric constraint representation {relation:?}: expected a supported authored geometric relation; dimensions and structural sources cannot be enumerated"
    )))
}
fn classify(relation: &PlanarConstraint) -> Option<ConstraintShape> {
    match relation {
        EllipseAxisAlignment {
            ellipse, vertical, ..
        } => single(axis_kind(*vertical), *ellipse),
        AxisCollinear { axes } => pair(COLLINEAR, axes.map(|axis| axis.entity())),
        AxisRelation {
            axes,
            perpendicular,
        } => pair(
            if *perpendicular {
                PERPENDICULAR
            } else {
                PARALLEL
            },
            axes.map(|axis| axis.entity()),
        ),
        _ => standard_shape(relation),
    }
}
fn standard_shape(relation: &PlanarConstraint) -> Option<ConstraintShape> {
    match relation {
        Coincident { points } => pair(COINCIDENT, *points),
        Horizontal { points } => pair(HORIZONTAL_ALIGN, *points),
        Vertical { points } => pair(VERTICAL_ALIGN, *points),
        LineHorizontal { line } => single(HORIZONTAL, *line),
        LineVertical { line } => single(VERTICAL, *line),
        Fix { point, .. } => single(FIX, *point),
        ArcMidpoint { point, arc } => pair(MIDPOINT, [*point, *arc]),
        Midpoint { point, line } => pair(MIDPOINT, [*point, *line]),
        PointOnLine { point, line } => pair(POINT_ON_LINE, [*point, *line]),
        Lines { lines, relation } => pair(line_kind(*relation), *lines),
        PointOnCircularCurve { point, curve } => pair(POINT_ON_CIRCLE, [*point, *curve]),
        Concentric { curves } => pair(CONCENTRIC, *curves),
        EqualRadius { curves } => pair(EQUAL_RADIUS, *curves),
        LineTangent { line, curve, .. } => pair(TANGENT, [*line, *curve]),
        CircularTangent { curves, .. } => pair(TANGENT, *curves),
        _ => None,
    }
}
fn line_kind(relation: LineRelation) -> GeometricConstraintKind {
    match relation {
        LineRelation::Parallel => PARALLEL,
        LineRelation::Perpendicular => PERPENDICULAR,
        LineRelation::Collinear => COLLINEAR,
        LineRelation::EqualLength => EQUAL_LENGTH,
    }
}
fn single(kind: GeometricConstraintKind, entity: SketchEntityId) -> Option<ConstraintShape> {
    Some((kind, vec![entity]))
}
fn pair(kind: GeometricConstraintKind, entities: [SketchEntityId; 2]) -> Option<ConstraintShape> {
    Some((kind, entities.to_vec()))
}

fn axis_kind(vertical: bool) -> GeometricConstraintKind {
    if vertical { VERTICAL } else { HORIZONTAL }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn dimensional_representation_is_a_named_decline_at_the_geometric_boundary() {
        let relation = Radius {
            curve: SketchEntityId(17),
            parameter: 99,
        };
        let cause = require(&relation).unwrap_err().to_string();
        assert!(cause.contains("17"));
        assert!(cause.contains("99"));
        assert!(cause.contains("expected a supported authored geometric relation"));
    }
}
