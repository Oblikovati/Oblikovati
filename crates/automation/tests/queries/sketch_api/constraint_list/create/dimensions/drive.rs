// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Caller, wire::DriveDimensionArgs};
use oblikovati_model::history::SketchDimensionEdit;
fn owned(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    super::super::delete_dimension::populated(kind)
}
fn drive_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &DriveDimensionArgs,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .edit(request)
}
fn edit_request() -> DriveDimensionArgs {
    DriveDimensionArgs {
        expression: "40".into(),
        set_driven: true,
        driven: true,
        set_limits: true,
        min: 2.0,
        max: 3.0,
        ..Default::default()
    }
}
fn target(files: &DiscoveryWorkspace, document: u64) -> f64 {
    files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .parameters
        .model_value(2)
        .unwrap()
        .value
}
#[test]
fn canonical_edit_clamps_display_input_and_reference_queries_measure_geometry_in_both_owners() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = owned(kind);
        let original = dimensions(&files)[0].value;
        let focus = files.workspace().active_document();
        assert!(
            drive_call(&mut files, &mut source, &edit_request())
                .unwrap()
                .ok
        );
        let row = &dimensions(&files)[0];
        assert!(row.driven);
        assert_eq!(row.value, original);
        assert_eq!(target(&files, document), 3.0);
        assert_eq!(files.workspace().active_document(), focus);
        assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_DRIVE_DIMENSION));
    }
}
#[test]
fn omitted_flags_and_empty_expression_preserve_all_dimension_state() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let rows = dimensions(&files);
    let request = DriveDimensionArgs {
        driven: true,
        min: 9.0,
        max: -1.0,
        ..Default::default()
    };
    drive_call(&mut files, &mut source, &request).unwrap();
    let after = owner_revision(&files, document);
    assert_eq!((after.1, after.2), (before.1, before.2));
    assert_eq!(dimensions(&files), rows);
    assert_eq!(target(&files, document), 1.0);
}
#[test]
fn limits_convert_centimetres_but_keep_radians_at_every_working_scale() {
    for scale in [1e-6, 1.0, 1e6] {
        assert_scaled_limits(scale);
    }
}
fn assert_scaled_limits(scale: f64) {
    let mut files = super::lifecycle::populated_scaled(scale);
    let document = files.workspace().active_document().unwrap();
    let mut source = QueryGuids(1000);
    for (index, factor) in [(0, scale), (9, 1.0), (10, 1.0)] {
        verify_scaled_limit(&mut files, &mut source, document, index, factor);
    }
}
fn verify_scaled_limit(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    document: u64,
    index: i64,
    factor: f64,
) {
    let request = DriveDimensionArgs {
        dimension_index: index,
        set_limits: true,
        min: 0.25,
        max: 0.75,
        ..Default::default()
    };
    drive_call(files, source, &request).unwrap();
    let limits = selected_limits(files, document, index as usize);
    assert_eq!(limits.lower, 0.25 / factor);
    assert_eq!(limits.upper, 0.75 / factor);
}
fn selected_limits(
    files: &DiscoveryWorkspace,
    document: u64,
    index: usize,
) -> oblikovati_model::sketch::SketchDimensionLimits {
    let geometry = owned_geometry(files, document)
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let id = geometry.dimensional_constraints().nth(index).unwrap().0.id;
    geometry.dimension_limits(id).unwrap().unwrap()
}
#[test]
fn invalid_edits_cannot_publish_partial_limits_or_reference_mode() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    let original = dimensions(&files);
    refuse_invalid_drives(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(dimensions(&files), original);
    assert_eq!(target(&files, document), 1.0);
}
fn refuse_invalid_drives(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    for expression in ["missing", "30 deg", "1e309 cm"] {
        let request = DriveDimensionArgs {
            expression: expression.into(),
            ..edit_request()
        };
        expect_api_decline(drive_call(files, source, &request), "drive expression");
    }
    let request = DriveDimensionArgs {
        min: 4.0,
        max: 3.0,
        ..edit_request()
    };
    expect_api_decline(drive_call(files, source, &request), "finite ordered");
}
#[test]
fn invalid_indices_payloads_and_document_kinds_are_refused() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        let request = DriveDimensionArgs {
            dimension_index: index,
            ..edit_request()
        };
        expect_api_decline(
            drive_call(&mut files, &mut source, &request),
            "dimension index",
        );
    }
    refuse_malformed(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    refuse_wrong_owner();
}
fn refuse_wrong_owner() {
    let (mut drawing, _, mut source) = configured_files(DOCUMENT_DRAWING, Default::default());
    expect_api_decline(
        drive_call(&mut drawing, &mut source, &edit_request()),
        "part or assembly",
    );
}
fn refuse_malformed(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let host = FileHost::new(files, source, "test");
    for input in [
        b"{".as_slice(),
        b"[]",
        br#"{"setLimits":"bad"}"#,
        br#"{"setLimits":true,"max":1e309}"#,
    ] {
        assert!(
            host.call(METHOD_SKETCH_DRIVE_DIMENSION, input).is_err(),
            "{input:?}"
        );
    }
}
#[test]
fn reference_dimension_queries_ignore_failed_unused_targets_on_both_query_hosts() {
    let (mut files, document, mut source) = owned(DOCUMENT_PART);
    drive_call(&mut files, &mut source, &edit_request()).unwrap();
    fail_unused_target(&mut files, document);
    let rows = dimensions(&files);
    assert!(rows[0].driven);
    assert_eq!(rows[0].value, 1.0);
    assert_eq!(rows[0].expression, "missing");
    let mutable = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap();
    assert_eq!(mutable, rows);
    restore_failed_driving(&mut files, &mut source);
}
fn fail_unused_target(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Fail unused target", |snapshot, _| {
            snapshot.parameters.set_expression(2, "missing")
        })
        .unwrap();
}
fn restore_failed_driving(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let request = DriveDimensionArgs {
        set_driven: true,
        ..Default::default()
    };
    drive_call(files, source, &request).unwrap();
    expect_api_decline(
        Client::new(FileQueryHost::new(files))
            .sketch()
            .dimensions(&Default::default()),
        "healthy",
    );
}
#[test]
fn typed_workspace_edit_uses_the_same_document_owner() {
    let (mut files, document, _) = owned(DOCUMENT_PART);
    let edit = SketchDimensionEdit {
        reference: Some(true),
        ..Default::default()
    };
    files
        .commands(document)
        .unwrap()
        .edit_sketch_dimension(DocumentSketchId(1), SketchConstraintId(1), &edit)
        .unwrap();
    assert!(dimensions(&files)[0].driven);
    assert_eq!(target(&files, document), 1.0);
}

#[path = "drive_history.rs"]
mod history;
