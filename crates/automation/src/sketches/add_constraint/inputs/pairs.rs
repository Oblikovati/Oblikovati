// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn classify(
    kind: &GeometricConstraintKind,
    first: &Operand,
    second: &Operand,
) -> Option<Creation> {
    let refs = [first.id, second.id];
    match (&first.kind, &second.kind) {
        (a, b) if *a == SKETCH_ENTITY_POINT && *b == SKETCH_ENTITY_POINT => points(kind, refs),
        (a, b) if *a == SKETCH_ENTITY_LINE && *b == SKETCH_ENTITY_LINE => lines(kind, refs),
        (a, b) if *a == SKETCH_ENTITY_POINT && *b == SKETCH_ENTITY_LINE => point_line(kind, refs),
        (a, b) if *a == SKETCH_ENTITY_POINT && circular(b) => {
            point_circular(kind, refs, *b == SKETCH_ENTITY_ARC)
        }
        (a, b) if *a == SKETCH_ENTITY_LINE && circular(b) && *kind == GEO_CONSTRAINT_TANGENT => {
            Some(Creation::LineTangent {
                line: first.id,
                curve: second.id,
            })
        }
        (a, b) if circular(a) && circular(b) => circles(kind, refs),
        _ => None,
    }
}
fn points(kind: &GeometricConstraintKind, points: [SketchEntityId; 2]) -> Option<Creation> {
    match kind {
        k if *k == GEO_CONSTRAINT_COINCIDENT => relation(Coincident { points }),
        k if *k == GEO_CONSTRAINT_HORIZONTAL || *k == GEO_CONSTRAINT_HORIZONTAL_ALIGN => {
            relation(Horizontal { points })
        }
        k if *k == GEO_CONSTRAINT_VERTICAL || *k == GEO_CONSTRAINT_VERTICAL_ALIGN => {
            relation(Vertical { points })
        }
        _ => None,
    }
}
fn lines(kind: &GeometricConstraintKind, lines: [SketchEntityId; 2]) -> Option<Creation> {
    let selected = match kind {
        k if *k == GEO_CONSTRAINT_PARALLEL => LineRelation::Parallel,
        k if *k == GEO_CONSTRAINT_PERPENDICULAR => LineRelation::Perpendicular,
        k if *k == GEO_CONSTRAINT_COLLINEAR => LineRelation::Collinear,
        k if *k == GEO_CONSTRAINT_EQUAL_LENGTH => LineRelation::EqualLength,
        _ => return None,
    };
    relation(Lines {
        lines,
        relation: selected,
    })
}
fn point_line(kind: &GeometricConstraintKind, refs: [SketchEntityId; 2]) -> Option<Creation> {
    let [point, line] = refs;
    match kind {
        k if *k == GEO_CONSTRAINT_POINT_ON_LINE => relation(PointOnLine { point, line }),
        k if *k == GEO_CONSTRAINT_MIDPOINT => relation(Midpoint { point, line }),
        _ => None,
    }
}
fn point_circular(
    kind: &GeometricConstraintKind,
    refs: [SketchEntityId; 2],
    arc: bool,
) -> Option<Creation> {
    let [point, curve] = refs;
    match kind {
        k if *k == GEO_CONSTRAINT_POINT_ON_CIRCLE => {
            relation(PointOnCircularCurve { point, curve })
        }
        k if *k == GEO_CONSTRAINT_MIDPOINT && arc => relation(ArcMidpoint { point, arc: curve }),
        _ => None,
    }
}
fn circles(kind: &GeometricConstraintKind, curves: [SketchEntityId; 2]) -> Option<Creation> {
    match kind {
        k if *k == GEO_CONSTRAINT_CONCENTRIC => relation(Concentric { curves }),
        k if *k == GEO_CONSTRAINT_EQUAL_RADIUS => relation(EqualRadius { curves }),
        k if *k == GEO_CONSTRAINT_TANGENT => Some(Creation::CircularTangent { curves }),
        _ => None,
    }
}
