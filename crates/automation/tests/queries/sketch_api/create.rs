// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    application_options::{SketchApplicationOptions, SketchPreferenceAccess},
    document_sketch::SketchPlacement,
    feature::{AuthoredFeature, FeatureId, work_geometry::WorkGeometryDefinition},
    sketch::SketchPlane,
};
use oblikovati_persistence::application_options::SketchPreferenceFile;
use std::boxed::Box as StdBox;
#[path = "create/history.rs"]
mod creation_history;
#[path = "create/options.rs"]
mod options;
fn create_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &CreateSketchArgs,
) -> Result<CreateSketchResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .create(request)
}
fn create_with_options(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    preferences: &mut dyn SketchPreferenceAccess,
) -> CreateSketchResult {
    Client::new(FileHost::new(files, source, "test").with_sketch_preferences(preferences))
        .sketch()
        .create(&Default::default())
        .unwrap()
}
fn snapshot(files: &DiscoveryWorkspace, document: u64) -> &DocumentSnapshot {
    files.workspace().session(document).unwrap().snapshot()
}
fn assert_created(files: &DiscoveryWorkspace, document: u64, index: i64, projected: bool) {
    let snapshot = snapshot(files, document);
    let placed = &snapshot.sketches.sketches()[usize::try_from(index).unwrap()];
    assert_eq!(placed.point_projections().len(), usize::from(projected));
    if projected {
        let view = snapshot.require_resolved_sketch(placed.id()).unwrap();
        assert_eq!(view.sketch.geometry().points()[0].position, [0.0; 2]);
        assert_eq!(view.verification.structure.degrees_of_freedom, 0);
    }
}
fn persisted_preferences(refuse_write: bool) -> SketchPreferenceFile<DiscoveryFiles> {
    let files = DiscoveryFiles {
        refuse_write,
        ..Default::default()
    };
    files.packages.borrow_mut().insert(Path::new("/preferences/options.yaml").into(), b"sketch: {headsUpDisplay: false, suppressFormatOverrides: true}\ngeneral: {startupAction: 9}\n".to_vec());
    SketchPreferenceFile::load(Path::new("/preferences/options.yaml"), files).unwrap()
}
#[test]
fn typed_creation_normalizes_all_origin_planes_and_creates_exact_origin_anchors() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    for (index, label, normalized, frame) in [
        (0, "", "XY", SketchPlane::xy()),
        (1, " xz ", "XZ", SketchPlane::xz()),
        (2, "yz", "YZ", SketchPlane::yz()),
    ] {
        let request = origin_request(label);
        let reply = create_call(&mut files, &mut source, &request).unwrap();
        assert_eq!(reply.sketch_index, index);
        assert_eq!(reply.plane, normalized);
        assert_created(&files, document, index, true);
        assert_current_frame(&files, document, index, frame);
    }
    assert_eq!(owner_revision(&files, document).1, ["Create sketch"; 3]);
}

#[test]
fn work_plane_indices_take_precedence_and_keep_the_catalogue_origin_order() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    for (index, frame, label) in [
        (0, SketchPlane::xy(), "XY Plane"),
        (1, SketchPlane::xz(), "XZ Plane"),
        (2, SketchPlane::yz(), "YZ Plane"),
    ] {
        let mut request = indexed_request(index);
        request.plane = "invalid but unused".into();
        let reply = create_call(&mut files, &mut source, &request).unwrap();
        assert_eq!(reply.plane, label);
        assert_placement(
            &files,
            document,
            reply.sketch_index,
            SketchPlacement::Fixed(frame),
        );
    }
}

fn install_owned_plane(files: &mut DiscoveryWorkspace, document: u64) -> FeatureId {
    files
        .commands(document)
        .unwrap()
        .run("Plane", |snapshot, ids| {
            let id = ids.allocate_feature().unwrap();
            let definition = WorkGeometryDefinition::Fixed(SketchPlane::xz());
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(id, "Owned host", definition))
                .unwrap();
            snapshot
                .work_geometry
                .set_work_plane_construction(id, true)
                .unwrap();
            Ok::<_, String>(id)
        })
        .unwrap()
}
#[test]
fn a_construction_host_is_addressable_by_its_stable_catalogue_index() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let host = install_owned_plane(&mut files, document);
    let reply = create_call(&mut files, &mut source, &indexed_request(3)).unwrap();
    assert_eq!(reply.plane, "Owned host");
    assert_placement(&files, document, 0, SketchPlacement::WorkPlane(host));
    assert_current_frame(&files, document, 0, SketchPlane::xz());
    assert_created(&files, document, reply.sketch_index, true);
}

fn orientation_request(axis: &str, origin: Option<Vec<f64>>) -> CreateSketchArgs {
    CreateSketchArgs {
        orientation: Some(StdBox::new(SketchOrientation {
            axis: axis.into(),
            axis_is_x: false,
            reverse: true,
            origin,
        })),
        ..Default::default()
    }
}
#[test]
fn origin_axis_orientation_and_custom_origin_precede_associative_projection() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let input = orientation_request(WORK_REF_X_AXIS, Some(vec![7.0, 11.0, 13.0]));
    let reply = create_call(&mut files, &mut source, &input).unwrap();
    let placed = &snapshot(&files, document).sketches.sketches()
        [usize::try_from(reply.sketch_index).unwrap()];
    assert_eq!(placed.geometry().points()[0].position, [-11.0, 7.0]);
    assert!(
        snapshot(&files, document)
            .require_resolved_sketch(placed.id())
            .is_ok()
    );
}
#[test]
fn empty_optional_origin_keeps_the_current_base_origin() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let input = orientation_request(WORK_REF_X_AXIS, Some(vec![]));
    let reply = create_call(&mut files, &mut source, &input).unwrap();
    assert_created(&files, document, reply.sketch_index, true);
}
fn assert_creation_declines(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    document: u64,
    input: CreateSketchArgs,
    offending: &str,
) {
    let before = owner_revision(files, document);
    let authored = snapshot(files, document).sketches.clone();
    expect_api_decline(create_call(files, source, &input), offending);
    assert!(authored.same_evaluated_state(&snapshot(files, document).sketches));
    let after = owner_revision(files, document);
    assert_eq!(before.0, after.0);
    assert_eq!(before.1, after.1);
}
#[test]
fn malformed_plane_and_orientation_inputs_do_not_publish_a_sketch() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    for label in ["bad", "XY/extra"] {
        assert_creation_declines(
            &mut files,
            &mut source,
            document,
            origin_request(label),
            label,
        );
    }
}

#[test]
fn suppressed_host_is_a_current_creation_decline_without_a_saved_frame() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let host = install_owned_plane(&mut files, document);
    files
        .commands(document)
        .unwrap()
        .run("Suppress host", |snapshot, _| {
            snapshot.work_geometry.set_suppressed(host, true)
        })
        .unwrap();
    assert_creation_declines(
        &mut files,
        &mut source,
        document,
        indexed_request(3),
        "placement",
    );
}

#[test]
fn assembly_creation_omits_the_part_origin_and_uses_the_shared_sketch_commands() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    let reply = create_call(&mut files, &mut source, &Default::default()).unwrap();
    assert_created(&files, document, reply.sketch_index, false);
    assert_eq!(status_call(&files, 0).unwrap().dof, 0);
    assert!(solve_call(&mut files, &mut source, 0).unwrap().converged);
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .sketch()
            .delete(&SketchArgs::default())
            .unwrap()
            .ok
    );
    assert!(snapshot(&files, document).sketches.sketches().is_empty());
}
#[test]
fn unsupported_document_kinds_and_an_absent_owner_refuse_creation() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    expect_api_decline(
        create_call(&mut files, &mut source, &Default::default()),
        "active part",
    );
    for kind in [DOCUMENT_DRAWING, DOCUMENT_PRESENTATION] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let before = owner_revision(&files, document);
        expect_api_decline(
            create_call(&mut files, &mut source, &Default::default()),
            "kind",
        );
        assert_eq!(before, owner_revision(&files, document));
    }
}

fn origin_request(label: &str) -> CreateSketchArgs {
    CreateSketchArgs {
        plane: label.into(),
        ..Default::default()
    }
}
fn indexed_request(index: i64) -> CreateSketchArgs {
    CreateSketchArgs {
        work_plane_index: Some(StdBox::new(index)),
        ..Default::default()
    }
}
fn assert_current_frame(
    files: &DiscoveryWorkspace,
    document: u64,
    index: i64,
    expected: SketchPlane,
) {
    let snapshot = snapshot(files, document);
    let sketch = snapshot.sketches.sketches()[usize::try_from(index).unwrap()].id();
    assert_eq!(snapshot.resolve_sketch_plane(sketch).unwrap(), expected);
}
fn assert_placement(
    files: &DiscoveryWorkspace,
    document: u64,
    index: i64,
    expected: SketchPlacement,
) {
    let snapshot = snapshot(files, document);
    let sketch = &snapshot.sketches.sketches()[usize::try_from(index).unwrap()];
    assert_eq!(sketch.placement(), expected);
}
#[test]
fn unavailable_and_degenerate_axes_and_malformed_origins_decline_creation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    for axis in ["", "edge/v1/7", "datum/v1/999", WORK_REF_Z_AXIS] {
        let offending = if axis == WORK_REF_Z_AXIS {
            "orientation"
        } else {
            axis
        };
        assert_creation_declines(
            &mut files,
            &mut source,
            document,
            orientation_request(axis, None),
            offending,
        );
    }
    let input = orientation_request(WORK_REF_X_AXIS, Some(vec![1.0, 2.0]));
    assert_creation_declines(&mut files, &mut source, document, input, "[1.0, 2.0]");
}

#[test]
fn invalid_work_plane_indices_decline_before_mutation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    for index in [-1, 3, i64::MAX] {
        assert_creation_declines(
            &mut files,
            &mut source,
            document,
            indexed_request(index),
            &index.to_string(),
        );
    }
}
fn install_axis(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> String {
    Client::new(FileHost::new(files, source, "test"))
        .work_axes()
        .create(&CreateWorkAxisArgs {
            kind: WORK_AXIS_LINE.0.into_owned(),
            origin: Some(vec![0.0; 3]),
            direction: Some(vec![0.0, 1.0, 0.0]),
            ..Default::default()
        })
        .unwrap()
        .ref_
}
#[test]
fn current_owned_axis_references_select_the_requested_sketch_axis() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let axis = install_axis(&mut files, &mut source);
    let mut input = orientation_request(&axis, Some(vec![7.0, 11.0, 13.0]));
    let orientation = input.orientation.as_mut().unwrap();
    orientation.axis_is_x = true;
    orientation.reverse = false;
    let reply = create_call(&mut files, &mut source, &input).unwrap();
    let placed = &snapshot(&files, document).sketches.sketches()
        [usize::try_from(reply.sketch_index).unwrap()];
    assert_eq!(placed.geometry().points()[0].position, [-11.0, 7.0]);
    assert!(
        snapshot(&files, document)
            .require_resolved_sketch(placed.id())
            .is_ok()
    );
}
#[test]
fn wrong_kind_axis_and_nonfinite_api_origin_decline_before_creation() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let plane = install_owned_plane(&mut files, document);
    let reference = format!("datum/v1/{}", plane.0);
    assert_creation_declines(
        &mut files,
        &mut source,
        document,
        orientation_request(&reference, None),
        "axis definition",
    );
    let input = orientation_request(WORK_REF_X_AXIS, Some(vec![f64::NAN; 3]));
    expect_api_decline(create_call(&mut files, &mut source, &input), "finite");
    assert!(snapshot(&files, document).sketches.sketches().is_empty());
}
