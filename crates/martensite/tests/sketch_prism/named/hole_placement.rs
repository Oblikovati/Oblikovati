// SPDX-License-Identifier: GPL-2.0-only
//! Face-based Hole placement retains the current named geometry (#4093).
use super::rebuild::certified_resolution;
use super::rebuild::holed_document;
use super::*;
use oblikovati_martensite::NamedSolid;

fn solid(history: &DocumentEditHistory) -> NamedSolid {
    NamedSolid::Prism(Box::new(named(history)))
}

#[test]
fn face_centroid_and_point_keep_the_exact_location_and_inward_direction() {
    let history = owner(selected_prism_document());
    let body = solid(&history);
    let axis = body
        .hole_axis(
            &history,
            document(),
            &key("end", &[1]),
            None,
            certified_resolution(),
        )
        .unwrap();
    assert_centroid(axis[0], [2., 1.5, 2.]);
    assert_eq!(axis[1], [0., 0., -1.]);
    let axis = body
        .hole_axis(
            &history,
            document(),
            &key("start", &[1]),
            Some([1., 1., 0.]),
            certified_resolution(),
        )
        .unwrap();
    assert_eq!(axis, [[1., 1., 0.], [0., 0., 1.]]);
}

#[test]
fn hole_position_rejects_outside_off_plane_boundary_and_wrong_names() {
    let history = owner(selected_prism_document());
    let body = solid(&history);
    for point in [[5., 1., 2.], [2., 1., 3.], [0., 1., 2.], [f64::NAN, 1., 2.]] {
        assert!(
            body.hole_axis(
                &history,
                document(),
                &key("end", &[1]),
                Some(point),
                certified_resolution()
            )
            .is_err(),
            "unexpectedly accepted {point:?}"
        );
    }
    assert!(
        body.hole_axis(
            &history,
            document(),
            &key("body", &[1]),
            None,
            certified_resolution()
        )
        .is_err()
    );
    let mut resolution = certified_resolution();
    resolution.sew = 0.;
    assert!(
        body.hole_axis(&history, document(), &key("end", &[1]), None, resolution)
            .is_err()
    );
}

#[test]
fn hole_placement_refuses_a_stale_named_body_after_an_edit() {
    let mut history = owner(selected_prism_document());
    let body = solid(&history);
    let token = history.begin("Depth").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.parameters.set_expression(3, "3 cm")
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert!(
        body.hole_axis(
            &history,
            document(),
            &key("end", &[1]),
            None,
            certified_resolution()
        )
        .is_err()
    );
    let axis = solid(&history)
        .hole_axis(
            &history,
            document(),
            &key("end", &[1]),
            None,
            certified_resolution(),
        )
        .unwrap();
    assert_centroid(axis[0], [2., 1.5, 3.]);
}

#[test]
fn face_centroid_inside_an_existing_opening_is_refused() {
    let history = owner(holed_document(0));
    let body = solid(&history);
    let face = key("start", &[1]);
    assert!(
        body.hole_axis(&history, document(), &face, None, certified_resolution())
            .is_err()
    );
    let axis = body
        .hole_axis(
            &history,
            document(),
            &face,
            Some([0.5, 0.5, 0.]),
            certified_resolution(),
        )
        .unwrap();
    assert_eq!(axis[0], [0.5, 0.5, 0.]);
}

#[test]
fn negative_oblique_extrusion_retains_the_outward_face_orientation() {
    let history = owner(holed_document(0));
    let body = solid(&history);
    let face = key("end", &[1]);
    let axis = body
        .hole_axis(
            &history,
            document(),
            &face,
            Some([1.5, 2.5, -3.]),
            certified_resolution(),
        )
        .unwrap();
    assert_eq!(axis, [[1.5, 2.5, -3.], [0., 0., 1.]]);
}
