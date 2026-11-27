// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{
    DisplayTolerance, ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId,
    ToleranceCap,
};
use oblikovati_model::{feature::sketch_prism::SketchDirectedCurve, sketch::SketchEntityId};

fn square(size: f64) -> (SketchProfile, SketchPrismSelection) {
    let positions = [[0., 0.], [size, 0.], [size, size], [0., size]];
    let vertices = positions
        .into_iter()
        .enumerate()
        .map(|(i, point)| ProfileVertex {
            id: ProfileVertexId(i as u64 + 1),
            point,
        })
        .collect();
    let curves = (0..4)
        .map(|i| ProfileCurve::Line {
            id: ProfileCurveId(i + 11),
            start: ProfileVertexId(i + 1),
            end: ProfileVertexId((i + 1) % 4 + 1),
        })
        .collect();
    let outer = (11..15)
        .map(|id| SketchDirectedCurve {
            curve: SketchEntityId(id),
            reversed: false,
        })
        .collect();
    (
        SketchProfile {
            plane: ProfilePlane::world_xy(),
            vertices,
            curves,
        },
        SketchPrismSelection {
            outer,
            holes: vec![],
        },
    )
}
fn resolution(cap: ToleranceCap) -> Resolution {
    Resolution::new(
        1e-6,
        1e-7,
        1e-9,
        DisplayTolerance {
            chordal: 1e-3,
            angular: 0.1,
        },
        cap,
    )
}
#[test]
fn certified_cube_retains_analytic_faces_lineage_and_measured_tolerance() {
    for size in [10., 20.] {
        let (profile, selection) = square(size);
        let result = retain_certified_profile_prism(
            &profile,
            &selection,
            [0., 0., size],
            resolution(ToleranceCap::Refuse(1e-5)),
        )
        .unwrap();
        let report = result.body.report();
        assert!((report.properties.volume - size.powi(3)).abs() < 1e-8);
        assert_eq!(
            (
                report.topology.faces,
                report.topology.edges,
                report.topology.vertices
            ),
            (6, 12, 8)
        );
        assert_eq!(report.faces.len(), 6);
        for face in &report.faces {
            assert!((face.area - size.powi(2)).abs() < 1e-8, "{face:?}");
            assert_eq!(face.loops, 1);
        }
        for role in [
            PrismFaceRole::Start,
            PrismFaceRole::End,
            PrismFaceRole::ProfileEdge(11),
            PrismFaceRole::ProfileEdge(12),
            PrismFaceRole::ProfileEdge(13),
            PrismFaceRole::ProfileEdge(14),
        ] {
            assert_eq!(
                report.faces.iter().filter(|face| face.role == role).count(),
                1
            );
        }
        let achieved = result.achieved_tolerance;
        assert!(
            [achieved.vertex, achieved.edge, achieved.face]
                .into_iter()
                .all(|value| value.is_finite() && value <= 1e-5)
        );
    }
}
#[test]
fn certified_taper_preserves_roles_and_respects_the_extrusion_direction() {
    use martensite_tkmath::gp::Vector3;
    let (profile, selection) = square(3.);
    let depth = 0.8;
    let settings = resolution(ToleranceCap::Refuse(1e-5));
    let untapered =
        retain_certified_profile_prism(&profile, &selection, [0., 0., depth], settings).unwrap();
    for direction in [1., -1.] {
        for degrees in [5.0_f64, -5.] {
            let angle = degrees.to_radians();
            let z = direction * depth;
            let taper = PrismTaper::new(SketchPlane::xy(), Vector3::new(0., 0., z), angle);
            let result =
                retain_certified_tapered_prism(&profile, &selection, [0., 0., z], settings, taper)
                    .unwrap();
            let report = result.body.report();
            let slope = angle.tan();
            let expected =
                9. * depth + 6. * slope * depth.powi(2) + 4. / 3. * slope.powi(2) * depth.powi(3);
            assert!((report.properties.volume - expected).abs() < 1e-8);
            assert_eq!(report.topology, untapered.body.report().topology);
            let identities = |history: &PrismSubentityHistory| {
                history
                    .edges
                    .iter()
                    .chain(&history.vertices)
                    .map(|record| (record.role, record.source, record.relation))
                    .collect::<Vec<_>>()
            };
            assert_eq!(
                identities(&result.subentities),
                identities(&untapered.subentities)
            );
            let far_corner = result
                .subentities
                .vertices
                .iter()
                .find(|record| {
                    record.role == PrismSubentityRole::EndVertex
                        && record.source == PrismHistorySource::Vertex(1)
                })
                .unwrap()
                .point;
            assert!((far_corner[0] + depth * slope).abs() < 1e-8);
            assert!((far_corner[2] - z).abs() < 1e-8);
            let caps = [PrismFaceRole::Start, PrismFaceRole::End].map(|role| {
                report
                    .faces
                    .iter()
                    .find(|face| face.role == role)
                    .unwrap()
                    .area
            });
            assert!((caps[0] - 9.).abs() < 1e-8);
            assert!((caps[1] - (3. + 2. * depth * slope).powi(2)).abs() < 1e-8);
        }
    }
}

#[test]
fn certified_prism_declines_an_unmatched_saved_profile() {
    let (profile, mut selection) = square(10.);
    selection.outer[0].curve = SketchEntityId(99);
    let error = retain_certified_profile_prism(
        &profile,
        &selection,
        [0., 0., 10.],
        resolution(ToleranceCap::Refuse(1e-5)),
    )
    .err()
    .unwrap();
    assert_eq!(error.phase(), Phase::Profile);
    assert_eq!(error.kind(), Kind::InvalidReference);
}

#[test]
fn short_profile_curve_preserves_the_kernel_admission_decline() {
    use martensite_tkbo::profile_extrusion::{DeclineCause, OperationDecline, OperationStage};
    use std::error::Error;
    let (profile, selection) = square(1e-15);
    let error = retain_certified_profile_prism(
        &profile,
        &selection,
        [0., 0., 10.],
        resolution(ToleranceCap::Refuse(1e-5)),
    )
    .err()
    .unwrap();
    assert_eq!(error.phase(), Phase::Profile);
    assert_eq!(error.kind(), Kind::InvalidInput);
    let decline = error
        .source()
        .unwrap()
        .downcast_ref::<OperationDecline>()
        .unwrap();
    assert_eq!(decline.stage, OperationStage::Admission);
    assert!(
        matches!(decline.cause, DeclineCause::DegenerateCurve { curve: ProfileCurveId(11), size, weld } if size < weld)
    );
}

#[test]
fn kernel_decline_keeps_its_typed_cause() {
    use std::error::Error;
    let (profile, selection) = square(10.);
    let error = retain_certified_profile_prism(
        &profile,
        &selection,
        [0., 0., 0.],
        resolution(ToleranceCap::Refuse(1e-5)),
    )
    .err()
    .unwrap();
    assert_eq!(error.phase(), Phase::Prism);
    assert_eq!(error.kind(), Kind::InvalidInput);
    assert!(
        error
            .source()
            .unwrap()
            .downcast_ref::<martensite_tkbo::profile_extrusion::OperationDecline>()
            .is_some()
    );
}

#[test]
fn tolerance_requires_a_limit_and_preserves_over_budget_declines() {
    let (profile, selection) = square(10.);
    let call =
        |cap| retain_certified_profile_prism(&profile, &selection, [0., 0., 10.], resolution(cap));
    assert_eq!(
        call(ToleranceCap::NoLimit).err().unwrap().kind(),
        Kind::InvalidInput
    );
    assert_eq!(
        call(ToleranceCap::Report(1e-5)).err().unwrap().kind(),
        Kind::InvalidInput
    );
    assert_eq!(
        call(ToleranceCap::Refuse(1e-12)).err().unwrap().kind(),
        Kind::ToleranceExceeded
    );
}

#[test]
fn certified_prism_keeps_source_faces_for_an_oblique_negative_sweep() {
    let (profile, selection) = square(10.);
    let result = retain_certified_profile_prism(
        &profile,
        &selection,
        [2., 3., -10.],
        resolution(ToleranceCap::Refuse(1e-5)),
    )
    .unwrap();
    let report = result.body.report();
    assert!((report.properties.volume - 1000.).abs() < 1e-8);
    assert_eq!(report.faces.len(), 6);
    for id in 11..15 {
        assert_eq!(
            report
                .faces
                .iter()
                .filter(|face| face.role == PrismFaceRole::ProfileEdge(id))
                .count(),
            1
        );
    }
    for (actual, expected) in report.properties.centroid.into_iter().zip([6., 6.5, -5.]) {
        assert!((actual - expected).abs() < 1e-8);
    }
}

#[test]
fn certified_cube_retains_every_edge_and_vertex_parent_with_analytic_positions() {
    use PrismHistorySource::{Curve, Vertex};
    use PrismSubentityRole::{EndEdge, EndVertex, LateralEdge, StartEdge, StartVertex};
    let build = |size| {
        let (profile, selection) = square(size);
        retain_certified_profile_prism(
            &profile,
            &selection,
            [0., 0., size],
            resolution(ToleranceCap::Refuse(1e-5)),
        )
        .unwrap()
        .subentities
    };
    let first = build(10.);
    let second = build(20.);
    assert_eq!((first.edges.len(), first.vertices.len()), (12, 8));
    for role in [StartEdge, EndEdge] {
        for id in 11..15 {
            assert!(
                first
                    .edges
                    .iter()
                    .any(|entry| entry.role == role && entry.source == Curve(id))
            );
        }
    }
    for id in 1..5 {
        assert!(
            first
                .edges
                .iter()
                .any(|entry| entry.role == LateralEdge && entry.source == Vertex(id))
        );
        for role in [StartVertex, EndVertex] {
            assert!(
                first
                    .vertices
                    .iter()
                    .any(|entry| entry.role == role && entry.source == Vertex(id))
            );
        }
    }
    for (before, after) in first
        .edges
        .iter()
        .chain(&first.vertices)
        .zip(second.edges.iter().chain(&second.vertices))
    {
        assert_eq!(
            (before.role, before.source, before.relation),
            (after.role, after.source, after.relation)
        );
        for (original, doubled) in before.point.into_iter().zip(after.point) {
            assert!((doubled - original * 2.).abs() < 1e-8);
        }
    }
}

#[test]
fn certified_prism_presentation_orients_every_triangle_outward() {
    for displacement in [[0., 0., 10.], [2., 3., -10.]] {
        assert_outward_prism(displacement);
    }
}
fn assert_outward_prism(displacement: [f64; 3]) {
    let (profile, selection) = square(10.);
    let snapshot = retain_certified_profile_prism(
        &profile,
        &selection,
        displacement,
        resolution(ToleranceCap::Refuse(1e-5)),
    )
    .unwrap();
    let mesh = snapshot
        .body
        .presentation(PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        })
        .unwrap();
    let center = [
        5. + displacement[0] / 2.,
        5. + displacement[1] / 2.,
        displacement[2] / 2.,
    ];
    assert_outward_faces(mesh, center);
}
fn assert_outward_faces(mesh: PrismPresentation, center: [f64; 3]) {
    for face in mesh.faces {
        for triangle in face.triangles {
            assert!(
                outward_dot(triangle, center) > 0.,
                "face {:?}, triangle {triangle:?}, body center {center:?}: expected outward winding",
                face.role
            );
        }
    }
}
fn outward_dot([a, b, c]: [[f64; 3]; 3], center: [f64; 3]) -> f64 {
    let u: [f64; 3] = std::array::from_fn(|i| b[i] - a[i]);
    let v: [f64; 3] = std::array::from_fn(|i| c[i] - a[i]);
    let n = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    (0..3)
        .map(|i| n[i] * ((a[i] + b[i] + c[i]) / 3. - center[i]))
        .sum()
}
