// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::LineRelation;
fn incident_lines(shared: usize, reverse: [bool; 2], scale: f64) -> Sketch2D {
    let positions = [[0., 0.], [100., 20.], [140., 120.], [80., 40.]];
    let positions = positions.map(|point| point.map(|value| value * scale));
    let second = match shared {
        0 => [3, 4],
        1 => [2, 3],
        _ => [1, 2],
    };
    let ends = [[1, 2], second];
    let ends = std::array::from_fn(|index| {
        let mut endpoints = ends[index];
        if reverse[index] {
            endpoints.reverse();
        }
        endpoints
    });
    collinear_geometry(positions, ends)
}
fn collinear_geometry(positions: [[f64; 2]; 4], ends: [[u64; 2]; 2]) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (index, position) in positions.into_iter().enumerate() {
        sketch
            .add_point(entity(index as u64 + 1), position)
            .unwrap();
    }
    for (index, endpoints) in ends.into_iter().enumerate() {
        sketch
            .add_line(entity(index as u64 + 5), endpoints.map(entity))
            .unwrap();
    }
    relation(
        &mut sketch,
        1,
        PlanarConstraint::Lines {
            lines: [entity(5), entity(6)],
            relation: LineRelation::Collinear,
        },
    );
    sketch
}
fn assert_incidence_orientations(shared: usize, scale: f64) {
    for reverse in [[false, false], [true, false], [false, true], [true, true]] {
        let sketch = incident_lines(shared, reverse, scale);
        let report = sketch.analyze(&parameters(), 1e-9).unwrap();
        assert_eq!(
            report.equations,
            2 - shared,
            "shared={shared}, reverse={reverse:?}, scale={scale}"
        );
        assert_eq!(report.rank, 2 - shared);
        assert_eq!(report.degrees_of_freedom, 6 + shared);
        assert_eq!(report.redundant_equations, 0);
    }
}
#[test]
fn collinear_equation_count_uses_shared_identity_before_numeric_rank() {
    for shared in 0..=2 {
        for scale in [0.01, 1., 100.] {
            assert_incidence_orientations(shared, scale);
        }
    }
}
#[test]
fn collinear_shared_incidence_solves_without_changing_identity() {
    for shared in 0..=2 {
        for reverse in [[false, false], [true, false], [false, true], [true, true]] {
            let sketch = incident_lines(shared, reverse, 1.);
            let outcome = sketch
                .solve(&parameters(), SolveOptions::default(), || false)
                .unwrap();
            assert_eq!(outcome.numerical().termination, SolveTermination::Converged);
            assert_eq!(outcome.numerical().structure.degrees_of_freedom, 6 + shared);
            assert_eq!(outcome.candidate().lines(), sketch.lines());
            assert_eq!(outcome.candidate().constraints(), sketch.constraints());
        }
    }
}
#[test]
fn collinear_independent_coincident_points_retain_two_equations() {
    let sketch = collinear_geometry(
        [[0., 0.], [100., 20.], [100., 20.], [140., 120.]],
        [[1, 2], [3, 4]],
    );
    let report = sketch.analyze(&parameters(), 1e-9).unwrap();
    assert_eq!(report.equations, 2);
    assert_eq!(report.rank, 2);
    assert_eq!(report.degrees_of_freedom, 6);
}
#[test]
fn collinear_identity_does_not_admit_zero_length_directions() {
    for shared in 0..=2 {
        let sketch = incident_lines(shared, [false, false], 0.);
        let error = sketch.analyze(&parameters(), 1e-9).unwrap_err();
        assert!(error.to_string().contains("nonzero direction"), "{error}");
    }
}
