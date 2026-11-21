// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn ellipse(
    center: SketchPointInput,
    major_axis: [f64; 2],
    radii: [f64; 2],
) -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Ellipse {
        center,
        major_axis,
        radii,
        construction: true,
    }
}
#[test]
fn ellipse_shares_the_grounded_origin_without_reserving_another_point() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, true);
    let request = ellipse(
        SketchPointInput::Owned(SketchEntityId(1)),
        [3.0, 4.0],
        [3.0, 1.0],
    );
    let report = session.create_sketch_primitive(sketch, &request).unwrap();
    assert_eq!(report.points, [SketchEntityId(1)]);
    assert_eq!(report.created, [SketchEntityId(2)]);
    let value = geometry(&session, sketch).ellipse(report.entity).unwrap();
    assert_eq!(value.center(), SketchEntityId(1));
    assert_eq!(value.major_axis(), [0.6, 0.8]);
    assert_eq!(value.radii(), [3.0, 1.0]);
    assert_numeric_verification(&session, sketch, 3);
}
#[test]
fn invalid_ellipse_frames_and_centers_do_not_reserve_identities() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    refuse_invalid_frames(&mut session, sketch);
    assert_refused(
        &mut session,
        sketch,
        &ellipse(
            SketchPointInput::Owned(SketchEntityId(99)),
            [1.0, 0.0],
            [3.0, 1.0],
        ),
    );
    assert_next_point(&mut session, sketch, 1);
}
#[test]
fn rejected_ellipse_reply_discards_geometry_and_retires_both_identities() {
    let mut session = session(None);
    let sketch = empty_sketch(&mut session, false);
    let request = ellipse(SketchPointInput::New([0.0; 2]), [1.0, 0.0], [1.0; 2]);
    let result = session.create_sketch_primitive_projected(sketch, &request, |result| {
        assert_eq!(result.unwrap().entity, SketchEntityId(2));
        Err::<(), _>("ellipse reply refused")
    });
    assert!(
        result
            .unwrap_err()
            .to_string()
            .contains("ellipse reply refused")
    );
    assert!(geometry(&session, sketch).ellipses().is_empty());
    let report = session.create_sketch_primitive(sketch, &request).unwrap();
    assert_eq!(report.entity, SketchEntityId(4));
    assert_eq!(report.points, [SketchEntityId(3)]);
    assert_numeric_verification(&session, sketch, 5);
}

fn refuse_invalid_frames(
    session: &mut DocumentSession<RecordingObserver>,
    sketch: DocumentSketchId,
) {
    for axis in [[0.0; 2], [f64::NAN, 1.0], [1.0, f64::INFINITY]] {
        assert_refused(
            session,
            sketch,
            &ellipse(SketchPointInput::New([0.0; 2]), axis, [3.0, 1.0]),
        );
    }
    for radii in [[0.0, 1.0], [1.0, 2.0], [3.0, f64::NAN]] {
        assert_refused(
            session,
            sketch,
            &ellipse(SketchPointInput::New([0.0; 2]), [1.0, 0.0], radii),
        );
    }
}
