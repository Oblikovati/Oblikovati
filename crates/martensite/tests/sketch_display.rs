// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::planar_curve::sketch::{
    SketchDisplayPrimitive, SketchSamplingOptionsV1, sample_sketch_display,
};
use oblikovati_model::sketch::{
    Sketch2D, SketchEllipse2D, SketchEntityId as Id, SketchSpline2D, SketchSplineKind,
};
fn options() -> SketchSamplingOptionsV1 {
    SketchSamplingOptionsV1 {
        deflection: 0.01,
        maximum_curve_points: 4096,
    }
}
fn fixture() -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for (id, point) in [(100, [0., 0.]), (101, [3., 0.]), (102, [0., 3.])] {
        sketch.add_defining_point(Id(id), point).unwrap();
    }
    sketch.add_line(Id(9), [Id(100), Id(101)]).unwrap();
    sketch.add_circle(Id(8), Id(100), 3.).unwrap();
    sketch
        .add_arc(Id(7), Id(100), [Id(101), Id(102)], true)
        .unwrap();
    sketch
        .add_ellipse(SketchEllipse2D::new(Id(6), Id(100), [1., 0.], [4., 2.]).unwrap())
        .unwrap();
    sketch.add_point(Id(5), [8., 9.]).unwrap();
    sketch.set_line_centerline(Id(9), true).unwrap();
    sketch.set_curve_construction(&[Id(8)], true).unwrap();
    sketch
}
#[test]
fn native_display_preserves_order_roles_and_standalone_points() {
    let sketch = fixture();
    let before = sketch.clone();
    let report = sample_sketch_display(&sketch, options()).unwrap();
    assert!(report.declines.is_empty(), "{:?}", report.declines);
    assert_eq!(
        report.entities.iter().map(|e| e.entity).collect::<Vec<_>>(),
        sketch.entity_ids()
    );
    assert!(report.entities[0].construction && report.entities[0].centerline);
    assert!(report.entities[1].construction && !report.entities[1].centerline);
    assert_eq!(
        report.entities[4].primitive,
        SketchDisplayPrimitive::Point([8., 9.])
    );
    assert!(sketch.same_recipe(&before));
    assert_eq!(report, sample_sketch_display(&sketch, options()).unwrap());
    for entity in &report.entities[..4] {
        assert_analytic_samples(entity.entity, &entity.primitive);
    }
}
fn assert_analytic_samples(id: Id, primitive: &SketchDisplayPrimitive) {
    let SketchDisplayPrimitive::Curve(points) = primitive else {
        panic!("expected curve")
    };
    assert!(points.len() >= 2);
    match id.0 {
        9 => assert_eq!(points, &[[0., 0.], [3., 0.]]),
        8 | 7 => {
            for point in points {
                assert!((point[0].hypot(point[1]) - 3.).abs() < 1e-12);
            }
        }
        6 => {
            for point in points {
                assert!((point[0] * point[0] / 16. + point[1] * point[1] / 4. - 1.).abs() < 1e-12);
            }
        }
        _ => panic!("unexpected id {id:?}"),
    }
}
#[test]
fn unsupported_and_degenerate_entities_are_named_without_hiding_valid_geometry() {
    let mut sketch = fixture();
    let spline = SketchSpline2D::new(
        Id(2),
        vec![Id(100), Id(101)],
        false,
        SketchSplineKind::Control,
    )
    .unwrap();
    sketch.add_spline(spline).unwrap();
    sketch.add_line(Id(1), [Id(100); 2]).unwrap();
    let report = sample_sketch_display(&sketch, options()).unwrap();
    assert_eq!(report.entities.len(), 5);
    assert_eq!(
        report.declines.iter().map(|d| d.entity).collect::<Vec<_>>(),
        [Id(2), Id(1)]
    );
    assert!(report.declines[0].cause.contains("issue #9"));
    assert!(!report.declines[1].cause.is_empty());
}
#[test]
fn invalid_policy_is_refused_even_for_an_empty_sketch() {
    let mut policy = options();
    policy.deflection = f64::NAN;
    assert!(sample_sketch_display(&Sketch2D::default(), policy).is_err());
}
#[test]
fn reduced_deflection_adds_samples_without_changing_the_model() {
    let sketch = fixture();
    let mut fine = options();
    fine.deflection *= 0.1;
    let report = sample_sketch_display(&sketch, fine).unwrap();
    let coarse = sample_sketch_display(&sketch, options()).unwrap();
    for index in 1..4 {
        let SketchDisplayPrimitive::Curve(a) = &report.entities[index].primitive else {
            panic!()
        };
        let SketchDisplayPrimitive::Curve(b) = &coarse.entities[index].primitive else {
            panic!()
        };
        assert!(a.len() > b.len());
    }
}

#[test]
fn excessive_conic_sampling_is_declined_before_native_allocation() {
    let mut policy = options();
    policy.deflection = f64::MIN_POSITIVE;
    let report = sample_sketch_display(&fixture(), policy).unwrap();
    assert_eq!(
        report
            .declines
            .iter()
            .map(|decline| decline.entity)
            .collect::<Vec<_>>(),
        [Id(8), Id(7), Id(6)]
    );
    assert_eq!(report.entities.len(), 2);
}

#[test]
fn circular_center_marker_keeps_the_defining_identity_and_position() {
    let report = sample_sketch_display(&fixture(), options()).unwrap();
    assert_eq!(report.center_markers.len(), 1);
    assert_eq!(report.center_markers[0].id, Id(100));
    assert_eq!(report.center_markers[0].position, [0., 0.]);
}
