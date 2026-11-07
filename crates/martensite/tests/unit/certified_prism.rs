// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbo::profile_extrusion::{
    DisplayTolerance, ProfileCurve, ProfileCurveId, ProfilePlane, ProfileVertex, ProfileVertexId,
    ToleranceCap,
};
use oblikovati_model::{feature::sketch_prism::SketchDirectedLine, sketch::SketchEntityId};

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
        .map(|id| SketchDirectedLine {
            line: SketchEntityId(id),
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
fn certified_prism_declines_an_unmatched_saved_profile() {
    let (profile, mut selection) = square(10.);
    selection.outer[0].line = SketchEntityId(99);
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
