// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::evaluate_planar_spline;
use oblikovati_model::sketch::Sketch2D;
#[path = "../planar_spline/go_recipe_fixture.rs"]
mod fixture;
use fixture::{FIT_POINTS, GoSplineOracle};
fn owned_spline(points: &[[f64; 2]], kind: SketchSplineKind, closed: bool) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    let ids: Vec<_> = (0..points.len())
        .map(|index| SketchEntityId(index as u64 + 10))
        .collect();
    for (&id, &position) in ids.iter().zip(points) {
        sketch.add_defining_point(id, position).unwrap();
    }
    sketch
        .add_spline(SketchSpline2D::new(SketchEntityId(1), ids, closed, kind).unwrap())
        .unwrap();
    sketch
}
fn resolved(sketch: &Sketch2D) -> ResolvedSketchCurve {
    resolve(
        &sketch.read_index(),
        sketch.spline(SketchEntityId(1)).unwrap(),
        1e-8,
    )
    .unwrap()
}
fn definition(curve: &ResolvedSketchCurve) -> &PlanarSplineDefinition {
    let PlanarGeometry::Spline(definition) = &curve.definition.geometry else {
        panic!("expected analytic spline")
    };
    definition
}
fn fit_kind(index: usize) -> SketchSplineKind {
    SketchSplineKind::Fit {
        method: [SPLINE_FIT_SMOOTH, SPLINE_FIT_CHORD, SPLINE_FIT_SWEET][index],
    }
}
fn close_values(actual: impl Iterator<Item = f64>, expected: impl Iterator<Item = f64>) {
    let actual: Vec<_> = actual.collect();
    let expected: Vec<_> = expected.collect();
    assert_eq!(actual.len(), expected.len());
    for (actual, expected) in actual.into_iter().zip(expected) {
        let bound = 1024. * f64::EPSILON * expected.abs().max(1.);
        assert!(
            (actual - expected).abs() <= bound,
            "{actual}: expected {expected} within {bound}"
        );
    }
}
#[test]
fn owned_fit_inputs_reconstruct_all_six_go_recipes_and_135_samples() {
    let records: Vec<GoSplineOracle> =
        serde_json::from_str(include_str!("../planar_spline/go-fit-oracle.json")).unwrap();
    assert_eq!(records.len(), 6);
    for record in records {
        verify_go_recipe(record);
    }
}
fn verify_go_recipe(record: GoSplineOracle) {
    let sketch = owned_spline(&FIT_POINTS, fit_kind(record.method as usize), record.closed);
    let before = sketch.clone();
    let curve = resolved(&sketch);
    let definition = definition(&curve);
    assert_eq!(definition.degree, record.degree);
    close_values(
        definition.poles.iter().flatten().copied(),
        record.poles.iter().flatten().copied(),
    );
    let flat = definition
        .knots
        .iter()
        .zip(&definition.multiplicities)
        .flat_map(|(&knot, &count)| std::iter::repeat_n(knot, count as usize));
    close_values(flat, record.knots.into_iter());
    close_values(
        curve
            .spline_construction
            .as_ref()
            .unwrap()
            .stations
            .iter()
            .copied(),
        record.stations.into_iter(),
    );
    for [parameter, x, y, dx, dy] in record.samples {
        let sample = evaluate_planar_spline(definition, parameter, 0.).unwrap();
        close_values(
            sample.position.into_iter().chain(sample.tangent),
            [x, y, dx, dy].into_iter(),
        );
    }
    assert_eq!(sketch, before);
}
#[test]
fn constructed_fit_and_control_curves_match_169_occt_positions_and_tangents() {
    let fits: Vec<_> = (0..6)
        .map(|case| resolved(&owned_spline(&FIT_POINTS, fit_kind(case % 3), case >= 3)))
        .collect();
    let controls: Vec<_> = [false, true]
        .map(|closed| {
            resolved(&owned_spline(
                &FIT_POINTS,
                SketchSplineKind::Control,
                closed,
            ))
        })
        .into();
    let fit_rows = include_str!("../planar_spline/occt-fit-construction-8.0.0.csv");
    let control_rows = include_str!("../planar_spline/occt-control-construction-8.0.0.csv");
    assert_eq!(fit_rows.lines().count() + control_rows.lines().count(), 169);
    for (rows, curves) in [(fit_rows, fits), (control_rows, controls)] {
        for row in rows.lines() {
            verify_occt_row(row, &curves);
        }
    }
}
fn verify_occt_row(row: &str, curves: &[ResolvedSketchCurve]) {
    let values: Vec<f64> = row.split(',').map(|field| field.parse().unwrap()).collect();
    let sample =
        evaluate_planar_spline(definition(&curves[values[0] as usize]), values[1], 0.).unwrap();
    close_values(
        sample.position.into_iter().chain(sample.tangent),
        values[2..6].iter().copied(),
    );
}
#[test]
fn construction_policy_is_independent_of_endpoint_and_display_bounds() {
    for closed in [false, true] {
        let sketch = owned_spline(&FIT_POINTS, fit_kind(0), closed);
        let fine = resolved(&sketch);
        let coarse = resolve(
            &sketch.read_index(),
            sketch.spline(SketchEntityId(1)).unwrap(),
            0.01,
        )
        .unwrap();
        assert_eq!(fine, coarse);
        let report = fine.spline_construction.unwrap();
        assert_eq!(report.source, SketchEntityId(1));
        assert!(report.fit_residual.unwrap() <= report.maximum_fit_residual);
        assert_eq!(report.seam_residuals.is_some(), closed);
        assert!(report.minimum_pivot > 0.);
    }
}
#[test]
fn signed_offsets_keep_parent_checks_and_cannot_alias_input_or_saved_recipe() {
    let sketch = owned_spline(&[[0., 0.], [2., 0.]], SketchSplineKind::Control, false);
    let original = resolved(&sketch);
    for distance in [-0.5, 0., 0.5] {
        let input =
            SketchOffsetSpline2D::new(SketchEntityId(2), SketchEntityId(1), distance).unwrap();
        let curve = offset(&sketch.read_index(), input, 1e-8).unwrap();
        assert_eq!(curve.entity, SketchEntityId(2));
        assert_eq!(curve.spline_construction, original.spline_construction);
        assert_eq!(curve.endpoint_residuals, None);
        let sample =
            evaluate_planar_spline(definition(&curve), 0.3, curve.definition.left_offset).unwrap();
        close_values(sample.position.into_iter(), [0.6, distance].into_iter());
    }
    let mut changed = sketch.recipe();
    changed.points[0].position[0] = 1.;
    let edited = Sketch2D::from_recipe(&changed).unwrap();
    assert_ne!(resolved(&edited).definition, original.definition);
    assert_eq!(resolved(&sketch), original);
}
#[test]
fn native_construction_failures_keep_typed_causes_and_source_identity() {
    let sketch = owned_spline(&[[0., 0.]; 3], fit_kind(0), false);
    let failure = resolve(
        &sketch.read_index(),
        sketch.spline(SketchEntityId(1)).unwrap(),
        1e-8,
    )
    .unwrap_err();
    assert_eq!(failure.entity, SketchEntityId(1));
    let super::super::SketchCurveFailureCause::Curve(cause) = failure.cause else {
        panic!("expected native curve cause")
    };
    assert_eq!(
        cause.kind(),
        crate::planar_curve::PlanarCurveFailureKind::Kernel
    );
    assert!(std::error::Error::source(cause.as_ref()).is_some());
}
#[test]
fn offset_requires_an_owned_spline_parent_and_missing_points_are_named() {
    let mut sketch = Sketch2D::default();
    sketch.add_point(SketchEntityId(1), [0., 0.]).unwrap();
    for parent in [1, 99] {
        let input =
            SketchOffsetSpline2D::new(SketchEntityId(2), SketchEntityId(parent), 1.).unwrap();
        let failure = offset(&sketch.read_index(), input, 1e-8).unwrap_err();
        assert_eq!(failure.entity, SketchEntityId(2));
    }
    let dangling = SketchSpline2D::new(
        SketchEntityId(2),
        vec![SketchEntityId(1), SketchEntityId(99)],
        false,
        SketchSplineKind::Control,
    )
    .unwrap();
    assert_eq!(
        resolve(&sketch.read_index(), &dangling, 1e-8)
            .unwrap_err()
            .entity,
        SketchEntityId(99)
    );
}
