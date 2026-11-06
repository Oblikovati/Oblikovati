// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchEntityId, SketchError};
use std::collections::BTreeMap;
type CoordinateTargets = BTreeMap<usize, Option<f64>>;

pub(super) fn authored_constants(sketch: &Sketch2D) -> Result<Vec<(usize, f64)>, SketchError> {
    let mut targets = CoordinateTargets::new();
    for constraint in &sketch.constraints {
        record_relation(sketch, &mut targets, &constraint.relation)?;
    }
    // Disagreeing targets remain equations, so the shared solver reports every conflicting source.
    Ok(targets
        .into_iter()
        .filter_map(|(column, value)| value.map(|value| (column, value)))
        .collect())
}
fn record_relation(
    sketch: &Sketch2D,
    targets: &mut CoordinateTargets,
    relation: &PlanarConstraint,
) -> Result<(), SketchError> {
    match relation {
        PlanarConstraint::Fix { point, position } => {
            record_point(sketch, targets, *point, *position)?
        }
        PlanarConstraint::Ground { points } => {
            for point in points {
                record_point(sketch, targets, point.id, point.position)?;
            }
        }
        _ => {}
    }
    Ok(())
}

fn record_point(
    sketch: &Sketch2D,
    targets: &mut CoordinateTargets,
    point: SketchEntityId,
    position: [f64; 2],
) -> Result<(), SketchError> {
    let column = fixed_point_column(sketch, point)?;
    for (axis, value) in position.into_iter().enumerate() {
        record_coordinate(targets, column + axis, value);
    }
    Ok(())
}
fn fixed_point_column(sketch: &Sketch2D, point: SketchEntityId) -> Result<usize, SketchError> {
    sketch
        .points()
        .iter()
        .position(|owned| owned.id == point)
        .map(|index| index * 2)
        .ok_or_else(|| {
            SketchError::Definition(format!(
                "fixed point {point:?}: expected an owned sketch point"
            ))
        })
}
fn record_coordinate(targets: &mut CoordinateTargets, column: usize, value: f64) {
    targets
        .entry(column)
        .and_modify(|target| {
            if target.is_some_and(|previous| previous.to_bits() != value.to_bits()) {
                *target = None;
            }
        })
        .or_insert(Some(value));
}

pub(super) fn seed_authored(
    sketch: &Sketch2D,
    constants: &[(usize, f64)],
) -> Result<Sketch2D, SketchError> {
    let mut seed = sketch.clone();
    for &(column, value) in constants {
        let point = seed.points()[column / 2];
        let mut position = point.position;
        position[column % 2] = value;
        seed.set_position(point.id, position)?;
    }
    Ok(seed)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn absent_fixed_points_are_refused_and_conflict_classification_is_order_independent() {
        assert!(
            fixed_point_column(&Sketch2D::default(), SketchEntityId(9))
                .unwrap_err()
                .to_string()
                .contains("SketchEntityId(9)")
        );
        for values in [[2., 3., 2.], [3., 2., 2.], [2., 2., 3.]] {
            let mut targets = CoordinateTargets::new();
            for value in values {
                record_coordinate(&mut targets, 0, value);
            }
            assert_eq!(targets[&0], None);
        }
    }
}
