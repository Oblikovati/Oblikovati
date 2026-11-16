// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
use oblikovati_model::sketch::{SketchCornerPredicate, SketchCornerTurn};

#[test]
fn corner_orientation_classifies_each_exact_branch_and_rejects_nonfinite_inputs() {
    let predicate = MartensiteSketchConstruction;
    let cases = [
        (
            [[0., 0.], [1., 0.], [0., 1.]],
            SketchCornerTurn::CounterClockwise,
        ),
        ([[0., 0.], [1., 0.], [0., -1.]], SketchCornerTurn::Clockwise),
        ([[0., 0.], [1., 0.], [2., 0.]], SketchCornerTurn::Collinear),
        (
            [[0., 0.], [1., 1e-200], [2., 3e-200]],
            SketchCornerTurn::CounterClockwise,
        ),
    ];
    for (points, expected) in cases {
        assert_eq!(predicate.turn(points).unwrap(), expected);
    }
    assert!(
        predicate
            .turn([[0., 0.], [1., 0.], [f64::NAN, 1.]])
            .is_err()
    );
}
