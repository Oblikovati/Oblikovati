// SPDX-License-Identifier: GPL-2.0-only
//! Placement of geometric-constraint marks in sketch coordinates.
use oblikovati_model::sketch::{PlanarConstraint, Sketch2D, SketchEntityId};

pub(super) fn relation_marks(geometry: &Sketch2D) -> Vec<([f64; 2], String)> {
    geometry
        .constraints()
        .iter()
        .filter_map(|constraint| relation_mark(geometry, &constraint.relation))
        .collect()
}
fn relation_mark(geometry: &Sketch2D, relation: &PlanarConstraint) -> Option<([f64; 2], String)> {
    let (at, text) = match relation {
        PlanarConstraint::Coincident { points } => (point_at(geometry, points[0])?, "C"),
        PlanarConstraint::LineHorizontal { line } => (line_middle(geometry, *line)?, "H"),
        PlanarConstraint::LineVertical { line } => (line_middle(geometry, *line)?, "V"),
        PlanarConstraint::Lines { lines, .. } => (line_middle(geometry, lines[0])?, "//"),
        _ => return None,
    };
    Some((at, text.to_owned()))
}
fn point_at(geometry: &Sketch2D, point: SketchEntityId) -> Option<[f64; 2]> {
    geometry.point(point).ok().map(|point| point.position)
}
fn line_middle(geometry: &Sketch2D, line: SketchEntityId) -> Option<[f64; 2]> {
    let [start, end] = geometry.line(line).ok()?.endpoints;
    let (start, end) = (point_at(geometry, start)?, point_at(geometry, end)?);
    Some([(start[0] + end[0]) * 0.5, (start[1] + end[1]) * 0.5])
}

#[cfg(test)]
mod tests {
    use super::*;
    fn sketch_with_a_horizontal_line() -> Sketch2D {
        let mut sketch = Sketch2D::default();
        let [start, end, line] = [1, 2, 3].map(SketchEntityId);
        sketch.add_point(start, [0., 0.]).unwrap();
        sketch.add_point(end, [4., 0.]).unwrap();
        sketch.add_line(line, [start, end]).unwrap();
        sketch
            .add_constraint(
                oblikovati_model::sketch::SketchConstraintId(1),
                PlanarConstraint::LineHorizontal { line },
            )
            .unwrap();
        sketch
    }
    #[test]
    fn a_horizontal_relation_marks_the_middle_of_its_line() {
        assert_eq!(
            relation_marks(&sketch_with_a_horizontal_line()),
            [([2., 0.], "H".to_owned())]
        );
    }
    #[test]
    fn a_relation_with_no_mark_yet_is_left_out() {
        let sketch = Sketch2D::default();
        assert!(relation_marks(&sketch).is_empty());
    }
}
