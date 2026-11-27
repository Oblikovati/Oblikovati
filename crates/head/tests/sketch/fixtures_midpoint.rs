// SPDX-License-Identifier: GPL-2.0-only
//! Shared midpoint-line fixtures for sketch scenarios (#3996).
use super::super::ApplicationWorld;
use oblikovati_model::sketch::{
    PlanarConstraint, Sketch2D, SketchGeometricConstraintCreation, SketchPointInput,
    SketchPrimitiveCreation,
};

impl ApplicationWorld {
    pub fn create_midpoint_line(&mut self) {
        let origin = self
            .edited_geometry()
            .points()
            .first()
            .expect("sketch origin")
            .id;
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let mut commands = self.app().owner.document_commands(edit.document);
        let line = commands
            .create_sketch_primitive(edit.sketch, &midpoint_line())
            .unwrap();
        let relation = SketchGeometricConstraintCreation::Relation(PlanarConstraint::Midpoint {
            point: origin,
            line: line.entity,
        });
        commands
            .create_sketch_geometric_constraint(edit.sketch, &relation)
            .unwrap();
        assert_midpoint_relation(&self.edited_geometry());
    }

    pub fn expect_midpoint_and_circle_restored(&mut self) {
        let current = self.edited_geometry();
        let before = &self
            .observed_sketch
            .as_ref()
            .expect("recorded fixture")
            .geometry;
        assert_retained_midpoint(before, &current);
        assert_retained_circle(before, &current);
        let witness = self.line_point(0, 0.3);
        self.expect_segment_drawn(witness, witness);
    }
}

fn assert_retained_midpoint(before: &Sketch2D, current: &Sketch2D) {
    assert_midpoint_relation(before);
    assert_midpoint_relation(current);
    assert_eq!(
        current.lines(),
        before.lines(),
        "retained midpoint line identity"
    );
    assert_eq!(
        midpoint_constraint(current),
        midpoint_constraint(before),
        "retained midpoint relation"
    );
    assert_retained_line_positions(before, current);
}

fn assert_retained_line_positions(before: &Sketch2D, current: &Sketch2D) {
    for point in before.lines()[0].endpoints {
        let expected = before.point(point).unwrap().position;
        let actual = current.point(point).unwrap().position;
        for axis in 0..2 {
            assert!(
                (actual[axis] - expected[axis]).abs() < 1e-9,
                "line endpoint {point:?} axis {axis}"
            );
        }
    }
}

fn midpoint_line() -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([-2., 1.5]),
            SketchPointInput::New([2., -1.5]),
        ],
        construction: false,
        centerline: false,
    }
}

fn midpoint_constraint(geometry: &Sketch2D) -> &oblikovati_model::sketch::SketchConstraint {
    let matches: Vec<_> = geometry
        .constraints()
        .iter()
        .filter(|c| matches!(c.relation, PlanarConstraint::Midpoint { .. }))
        .collect();
    assert_eq!(matches.len(), 1, "expected one midpoint relation");
    matches[0]
}

fn assert_midpoint_relation(geometry: &Sketch2D) {
    let PlanarConstraint::Midpoint { point, line } = midpoint_constraint(geometry).relation else {
        unreachable!()
    };
    let line = geometry.line(line).expect("midpoint line exists");
    let center = geometry
        .point(point)
        .expect("midpoint point exists")
        .position;
    let ends = line
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    for axis in 0..2 {
        assert!(
            (center[axis] - (ends[0][axis] + ends[1][axis]) / 2.).abs() < 1e-9,
            "midpoint relation coordinate {axis}"
        );
    }
}

fn assert_retained_circle(before: &Sketch2D, current: &Sketch2D) {
    assert_eq!(before.circles().len(), 1, "baseline circle count");
    assert_eq!(
        current.circles(),
        before.circles(),
        "retained circle identity and radius"
    );
    let center = before.circles()[0].center;
    assert_eq!(
        current.point(center).unwrap(),
        before.point(center).unwrap(),
        "retained circle center"
    );
}

#[test]
#[should_panic(expected = "expected one midpoint relation")]
fn midpoint_check_rejects_a_plain_line() {
    use oblikovati_model::sketch::SketchEntityId;
    let mut geometry = Sketch2D::default();
    let first = SketchEntityId(1);
    let second = SketchEntityId(2);
    geometry.add_point(first, [-2., 1.5]).unwrap();
    geometry.add_point(second, [2., -1.5]).unwrap();
    geometry
        .add_line(SketchEntityId(3), [first, second])
        .unwrap();
    assert_midpoint_relation(&geometry);
}
