// SPDX-License-Identifier: GPL-2.0-only
//! Placement of geometric-constraint marks in sketch coordinates.
use oblikovati_model::sketch::{
    LineRelation, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
};

#[derive(Debug, PartialEq)]
pub(super) struct RelationMark {
    pub id: SketchConstraintId,
    pub at: [f64; 2],
    pub text: String,
    pub related: Vec<SketchEntityId>,
}

pub(super) fn relation_marks(geometry: &Sketch2D) -> Vec<RelationMark> {
    geometry
        .constraints()
        .iter()
        .filter_map(|constraint| {
            let (at, text, related) = relation_mark(geometry, &constraint.relation)?;
            Some(RelationMark {
                id: constraint.id,
                at,
                text: text.into(),
                related,
            })
        })
        .collect()
}
fn relation_mark(
    geometry: &Sketch2D,
    relation: &PlanarConstraint,
) -> Option<([f64; 2], &'static str, Vec<SketchEntityId>)> {
    let (at, text, related) = match relation {
        PlanarConstraint::Coincident { points } => {
            (point_at(geometry, points[0])?, "C", points.to_vec())
        }
        PlanarConstraint::LineHorizontal { line } => {
            (line_middle(geometry, *line)?, "H", vec![*line])
        }
        PlanarConstraint::LineVertical { line } => {
            (line_middle(geometry, *line)?, "V", vec![*line])
        }
        PlanarConstraint::Lines {
            lines,
            relation: LineRelation::Parallel,
        } => (line_middle(geometry, lines[0])?, "//", lines.to_vec()),
        PlanarConstraint::Lines { lines, relation } => (
            line_middle(geometry, lines[0])?,
            line_relation_text(*relation),
            lines.to_vec(),
        ),
        _ => return None,
    };
    Some((at, text, related))
}
fn line_relation_text(relation: LineRelation) -> &'static str {
    match relation {
        LineRelation::Parallel => "//",
        LineRelation::Perpendicular => "⊥",
        LineRelation::Collinear => "Col",
        LineRelation::EqualLength => "=",
    }
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
            [RelationMark {
                id: oblikovati_model::sketch::SketchConstraintId(1),
                at: [2., 0.],
                text: "H".to_owned(),
                related: vec![SketchEntityId(3)]
            }]
        );
    }
    #[test]
    fn a_relation_with_no_mark_yet_is_left_out() {
        let sketch = Sketch2D::default();
        assert!(relation_marks(&sketch).is_empty());
    }
    #[test]
    fn parallel_mark_keeps_constraint_and_both_related_lines() {
        let mut sketch = Sketch2D::default();
        for (id, point) in [(1, [0., 0.]), (2, [4., 0.]), (3, [0., 2.]), (4, [4., 2.])] {
            sketch.add_point(SketchEntityId(id), point).unwrap();
        }
        let lines = [SketchEntityId(5), SketchEntityId(6)];
        sketch
            .add_line(lines[0], [SketchEntityId(1), SketchEntityId(2)])
            .unwrap();
        sketch
            .add_line(lines[1], [SketchEntityId(3), SketchEntityId(4)])
            .unwrap();
        let id = SketchConstraintId(7);
        sketch
            .add_constraint(
                id,
                PlanarConstraint::Lines {
                    lines,
                    relation: LineRelation::Parallel,
                },
            )
            .unwrap();
        assert_eq!(
            relation_marks(&sketch),
            [RelationMark {
                id,
                at: [2., 0.],
                text: "//".into(),
                related: lines.to_vec()
            }]
        );
    }
    #[test]
    fn nonparallel_relations_do_not_claim_parallel_hover() {
        for relation in [
            LineRelation::Perpendicular,
            LineRelation::Collinear,
            LineRelation::EqualLength,
        ] {
            assert_ne!(line_relation_text(relation), "//");
        }
    }
}
