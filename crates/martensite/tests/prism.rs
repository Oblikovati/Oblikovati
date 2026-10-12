// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{
    PrismFaceRole, PrismProbeDefinition, PrismProbeLoop, probe_planar_prism,
};
fn rectangle() -> PrismProbeDefinition {
    PrismProbeDefinition {
        outer: PrismProbeLoop {
            edge_keys: vec![1, 2, 3, 4],
            vertices: vec![[0., 0.], [4., 0.], [4., 3.], [0., 3.]],
        },
        holes: Vec::new(),
        displacement: [0., 0., 2.],
    }
}
#[test]
fn closed_profile_prism_has_analytic_caps_sides_and_complete_source_roles() {
    let definition = rectangle();
    let report = probe_planar_prism(&definition).unwrap();
    assert!((report.properties.volume - 24.).abs() < 1e-11);
    assert_eq!(report.faces.len(), 6);
    verify_caps(&report, 12., 1);
    for key in &definition.outer.edge_keys {
        assert!(
            report
                .faces
                .iter()
                .any(|face| face.role == PrismFaceRole::ProfileEdge(*key))
        );
    }
}
fn verify_caps(report: &oblikovati_martensite::PrismProbeReport, area: f64, loops: usize) {
    let start = report
        .faces
        .iter()
        .find(|face| face.role == PrismFaceRole::Start)
        .unwrap();
    let end = report
        .faces
        .iter()
        .find(|face| face.role == PrismFaceRole::End)
        .unwrap();
    assert!((start.area - area).abs() < 1e-11 && (end.area - area).abs() < 1e-11);
    assert_eq!((start.loops, end.loops), (loops, loops));
}
fn with_hole() -> PrismProbeDefinition {
    let mut definition = rectangle();
    definition.holes.push(PrismProbeLoop {
        edge_keys: vec![11, 12, 13, 14],
        vertices: vec![[1., 1.], [1., 2.], [3., 2.], [3., 1.]],
    });
    definition
}
#[test]
fn inner_loop_prism_uses_the_same_path_for_positive_negative_and_skew_displacements() {
    for vector in [[0., 0., 2.], [0., 0., -2.], [1., 2., 3.]] {
        let mut definition = with_hole();
        definition.displacement = vector;
        let report = probe_planar_prism(&definition).unwrap();
        assert!((report.properties.volume - 10. * vector[2].abs()).abs() < 1e-10);
        assert_eq!(report.faces.len(), 10);
        for face in &report.faces {
            assert_eq!(
                face.loops,
                if matches!(face.role, PrismFaceRole::Start | PrismFaceRole::End) {
                    2
                } else {
                    1
                }
            );
        }
    }
}
#[test]
fn malformed_profile_and_direction_inputs_are_declined_before_construction() {
    for vector in [
        [0.; 3],
        [1., 0., 0.],
        [f64::NAN, 0., 1.],
        [0., 0., f64::INFINITY],
    ] {
        let mut definition = rectangle();
        definition.displacement = vector;
        assert!(
            probe_planar_prism(&definition)
                .unwrap_err()
                .contains("displacement")
        );
    }
}
#[test]
fn malformed_profile_keys_and_loop_shapes_are_declined_before_construction() {
    for keys in [vec![1, 2], vec![0, 2, 3, 4], vec![1, 2, 1, 4]] {
        let mut definition = rectangle();
        definition.outer.edge_keys = keys;
        assert!(
            probe_planar_prism(&definition)
                .unwrap_err()
                .contains("expected")
        );
    }
    let mut definition = with_hole();
    definition.holes[0].edge_keys[0] = 1;
    assert!(
        probe_planar_prism(&definition)
            .unwrap_err()
            .contains("unique")
    );
}
#[test]
fn collapsed_and_nonfinite_segments_are_refused_with_the_source_identity() {
    let mut definition = rectangle();
    definition.outer.vertices[1] = definition.outer.vertices[0];
    assert!(
        probe_planar_prism(&definition)
            .unwrap_err()
            .contains("edge 1")
    );
    definition.outer.vertices[1] = [1e-10, 0.];
    assert!(
        probe_planar_prism(&definition)
            .unwrap_err()
            .contains("collapsed")
    );
    definition.outer.vertices[1] = [f64::INFINITY, 0.];
    let error = probe_planar_prism(&definition).unwrap_err();
    assert!(error.contains("finite XY"));
}
#[path = "prism/oracle.rs"]
mod oracle;
#[test]
fn self_crossing_and_wrongly_oriented_inner_loops_are_kernel_declines() {
    let mut crossing = rectangle();
    crossing.outer.vertices = vec![[0., 0.], [4., 3.], [0., 3.], [4., 0.]];
    assert!(probe_planar_prism(&crossing).is_err());
    let mut wrong_hole = with_hole();
    wrong_hole.holes[0].vertices.reverse();
    assert!(probe_planar_prism(&wrong_hole).is_err());
}
#[test]
fn separate_prism_evaluations_leave_the_owned_profile_unchanged() {
    let definition = with_hole();
    let before = definition.clone();
    let first = probe_planar_prism(&definition).unwrap();
    let mut other = rectangle();
    other.displacement = [1., 2., 4.];
    probe_planar_prism(&other).unwrap();
    assert_eq!(definition, before);
    assert_eq!(probe_planar_prism(&definition).unwrap(), first);
}
#[path = "prism/mesh.rs"]
mod mesh;
#[path = "prism/placement.rs"]
mod placement;
