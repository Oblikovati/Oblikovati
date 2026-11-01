// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{
    PlanarProfileLoop, RevolveFaceRole, RevolveProbeDefinition, probe_planar_revolve,
};
use std::f64::consts::{FRAC_PI_2, PI, TAU};

fn rectangle(angle: f64) -> RevolveProbeDefinition {
    RevolveProbeDefinition {
        outer: PlanarProfileLoop::polygon(
            vec![1, 2, 3, 4],
            vec![[1., 0.], [2., 0.], [2., 1.], [1., 1.]],
        ),
        holes: Vec::new(),
        axis: [[0., 0.], [0., 1.]],
        angle,
    }
}

#[test]
fn partial_revolution_has_caps_and_one_face_per_profile_edge() {
    let definition = rectangle(FRAC_PI_2);
    let report = probe_planar_revolve(&definition).unwrap();
    assert!((report.properties.volume - 3. * PI / 4.).abs() < 1e-10);
    assert_eq!(report.faces.len(), 6);
    assert!(
        report
            .faces
            .iter()
            .any(|face| face.role == RevolveFaceRole::Start)
    );
    assert!(
        report
            .faces
            .iter()
            .any(|face| face.role == RevolveFaceRole::End)
    );
    require_source_faces(&report, &definition.outer.edge_keys());
}

#[test]
fn full_revolution_is_declined_without_complete_source_edge_history() {
    let definition = rectangle(TAU);
    let error = probe_planar_revolve(&definition).unwrap_err();
    assert!(error.contains("complete source-edge history"));
}

#[test]
fn axis_crossing_profile_is_a_named_decline() {
    let mut definition = rectangle(FRAC_PI_2);
    definition.outer = PlanarProfileLoop::polygon(
        vec![1, 2, 3, 4],
        vec![[-1., 0.], [1., 0.], [1., 1.], [-1., 1.]],
    );
    let error = probe_planar_revolve(&definition).unwrap_err();
    assert!(error.contains("does not cross the rotation axis"));
}

fn require_source_faces(report: &oblikovati_martensite::RevolveProbeReport, keys: &[u64]) {
    for key in keys {
        assert!(
            report
                .faces
                .iter()
                .any(|face| { face.role == RevolveFaceRole::ProfileEdge(*key) })
        );
    }
}
