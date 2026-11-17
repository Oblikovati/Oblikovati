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

#[test]
fn injected_corner_constructor_certifies_fillet_and_chamfer_in_both_orientations() {
    use oblikovati_model::sketch::{SketchCornerConstruction, SketchCornerSize};
    let backend = MartensiteSketchConstruction;
    for sign in [-1.0, 1.0] {
        let points = [[0., 0.], [4., 0.], [0., 5. * sign]];
        let bevel = backend
            .construct_corner(points, SketchCornerSize::Chamfer([1., 2.]), 0.)
            .unwrap();
        assert_eq!(bevel.endpoints, [[1., 0.], [0., 2. * sign]]);
        assert!(bevel.arc.is_none());
        let fillet = backend
            .construct_corner(points, SketchCornerSize::Fillet(1.), 0.)
            .unwrap();
        assert_eq!(fillet.endpoints, [[1., 0.], [0., sign]]);
        assert_eq!(fillet.arc, Some(([1., sign], sign < 0.)));
    }
}
#[test]
fn injected_corner_constructor_refuses_degenerate_and_unrepresentable_inputs() {
    use oblikovati_model::sketch::{SketchCornerConstruction, SketchCornerSize};
    let backend = MartensiteSketchConstruction;
    let points = [[0., 0.], [4., 0.], [0., 5.]];
    for size in [
        SketchCornerSize::Fillet(0.),
        SketchCornerSize::Fillet(5.),
        SketchCornerSize::Chamfer([1., 5.]),
        SketchCornerSize::Chamfer([f64::NAN, 1.]),
    ] {
        assert!(backend.construct_corner(points, size, 0.).is_err());
    }
    assert!(
        backend
            .construct_corner(
                [[0., 0.], [1., 0.], [2., 0.]],
                SketchCornerSize::Fillet(1.),
                0.
            )
            .is_err()
    );
    assert!(
        backend
            .construct_corner(points, SketchCornerSize::Fillet(1.), f64::NAN)
            .is_err()
    );
}

#[test]
fn oblique_corner_seed_checks_tangent_geometry_and_rejects_lost_precision() {
    use oblikovati_model::sketch::{SketchCornerConstruction, SketchCornerSize};
    let backend = MartensiteSketchConstruction;
    let points = [[0., 0.], [5., 0.], [3., 4.]];
    let seed = backend
        .construct_corner(points, SketchCornerSize::Fillet(1.), 1e-12)
        .unwrap();
    let (center, _) = seed.arc.unwrap();
    for endpoint in seed.endpoints {
        assert!(((endpoint[0] - center[0]).hypot(endpoint[1] - center[1]) - 1.).abs() < 1e-12);
    }
    let distant = [[1e16, 1e16], [1e16 + 10., 1e16], [1e16, 1e16 + 10.]];
    assert!(
        backend
            .construct_corner(distant, SketchCornerSize::Fillet(1.), 0.)
            .is_err()
    );
}
