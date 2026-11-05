// SPDX-License-Identifier: GPL-2.0-only
use super::recompute::saved_driven_sketch;
use super::*;
use oblikovati_model::sketch::{SketchEntityId, SketchPointInput, SketchPrimitiveCreation};
fn centerline_request() -> SketchPrimitiveCreation {
    SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::Owned(SketchEntityId(1)),
            SketchPointInput::Owned(SketchEntityId(2)),
        ],
        construction: false,
        centerline: true,
    }
}
fn create_centerline(host: &mut Host, document: u64, sketch: DocumentSketchId) -> SketchEntityId {
    host.commands(document)
        .unwrap()
        .create_sketch_primitive(sketch, &centerline_request())
        .unwrap()
        .entity
}
fn assert_centerline_owner(
    host: &Host,
    document: u64,
    sketch: DocumentSketchId,
    line: SketchEntityId,
) {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let view = snapshot.require_resolved_sketch(sketch).unwrap();
    let geometry = view.sketch.geometry();
    assert!(geometry.is_centerline(line).unwrap());
    assert!(geometry.is_construction(line).unwrap());
    assert_eq!(geometry.profile_curve_ids(), [SketchEntityId(4)]);
    assert_eq!(geometry.centerline_axis().unwrap().unwrap().id, line);
    assert_eq!(view.verification.structure.degrees_of_freedom, 0);
}
#[test]
fn typed_centerline_creation_keeps_shared_points_and_current_solver_evidence() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    let line = create_centerline(&mut host, document, sketch);
    assert_eq!(line, SketchEntityId(5));
    assert_centerline_owner(&host, document, sketch, line);
    host.commands(document).unwrap().undo().unwrap();
    assert!(host.workspace().needs_save(document).unwrap());
    host.commands(document).unwrap().redo().unwrap();
    assert_centerline_owner(&host, document, sketch, line);
}
#[test]
fn centerline_creation_and_group_abort_leave_original_profile_candidates() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    host.commands(document)
        .unwrap()
        .begin("Axis group")
        .unwrap();
    let line = create_centerline(&mut host, document, sketch);
    assert_centerline_owner(&host, document, sketch, line);
    host.commands(document).unwrap().abort().unwrap();
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot.sketches.get(sketch).unwrap().geometry();
    assert_eq!(geometry.centerline_axis().unwrap(), None);
    assert_eq!(geometry.profile_curve_ids(), [SketchEntityId(4)]);
    assert!(host.workspace().needs_save(document).unwrap());
}
fn save_centerline_copy(host: &mut Host, document: u64, source: &mut SequenceGuids) -> u64 {
    host.save_copy(document, &path("axis-copy.obk"), None, "alpha", source)
        .unwrap();
    open_file(host, "axis-copy.obk", source, OpenActivation::Background)
}
#[test]
fn native_save_reopen_and_copy_preserve_axis_metadata_and_current_geometry() {
    let (mut host, mut source, document, sketch) = saved_driven_sketch();
    let line = create_centerline(&mut host, document, sketch);
    host.save(document, "alpha", &mut source).unwrap();
    let copied = save_centerline_copy(&mut host, document, &mut source);
    assert_centerline_owner(&host, copied, sketch, line);
    host.close_discarding_changes(document).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_centerline_owner(&host, reopened, sketch, line);
    assert!(!host.workspace().needs_save(reopened).unwrap());
}
#[test]
fn rejected_centerline_reply_publishes_no_axis_and_retains_its_curve_reservation() {
    let (mut host, _, document, sketch) = saved_driven_sketch();
    let before = owned_geometry(&host, document, sketch);
    let result = host
        .commands(document)
        .unwrap()
        .create_sketch_primitive_projected(sketch, &centerline_request(), |report| {
            assert_eq!(report.unwrap().entity, SketchEntityId(5));
            Err::<(), _>("axis reply refused")
        });
    assert!(result.is_err());
    let after = owned_geometry(&host, document, sketch);
    assert!(before.same_recipe(&after));
    assert!(host.workspace().needs_save(document).unwrap());
    assert_eq!(
        create_centerline(&mut host, document, sketch),
        SketchEntityId(6)
    );
}
fn owned_geometry(host: &Host, document: u64, sketch: DocumentSketchId) -> Sketch2D {
    let snapshot = host.workspace().session(document).unwrap().snapshot();
    snapshot.sketches.get(sketch).unwrap().geometry().clone()
}
