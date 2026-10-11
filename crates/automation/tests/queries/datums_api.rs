// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_api::{configured_files, owner_revision, reopen_with_observer};
use super::*;
use oblikovati_automation::{DATUM_COMMAND_METHODS, DATUM_QUERY_METHODS, FileHost};
use oblikovati_model::feature::{
    FeatureId,
    work_geometry::{DatumLength, WorkGeometryDefinition},
};

fn create_point(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    at: [f64; 3],
    construction: bool,
) -> CreateWorkPointResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_points()
        .create(&CreateWorkPointArgs {
            at: Some(at.to_vec()),
            construction,
            ..Default::default()
        })
        .unwrap()
}
fn create_line(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> CreateWorkAxisResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_axes()
        .create(&CreateWorkAxisArgs {
            kind: WORK_AXIS_LINE.0.into_owned(),
            origin: Some(vec![2.0, 3.0, 4.0]),
            direction: Some(vec![0.0, 0.0, 7.0]),
            ..Default::default()
        })
        .unwrap()
}
fn create_two_points(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    first: &str,
    second: &str,
    construction: bool,
) -> CreateWorkAxisResult {
    Client::new(FileHost::new(files, source, "test"))
        .work_axes()
        .create(&CreateWorkAxisArgs {
            kind: WORK_AXIS_TWO_POINTS.0.into_owned(),
            refs: Some(vec![first.into(), second.into()]),
            construction,
            ..Default::default()
        })
        .unwrap()
}
fn point_rows(files: &DiscoveryWorkspace, include: bool) -> Vec<WorkPointInfo> {
    if !include {
        return Client::new(FileQueryHost::new(files))
            .work_points()
            .list()
            .unwrap()
            .points
            .unwrap();
    }
    typed_point_options(
        files,
        ListWorkPointsArgs {
            include_construction: true,
            ..Default::default()
        },
    )
    .unwrap()
    .points
    .unwrap()
}
fn typed_point_options(
    files: &DiscoveryWorkspace,
    request: ListWorkPointsArgs,
) -> Result<ListWorkPointsResult, oblikovati_api::Error> {
    Client::new(FileQueryHost::new(files))
        .work_points()
        .list_with_options(&request)
}
fn axis_rows(files: &DiscoveryWorkspace, include: bool) -> Vec<WorkAxisInfo> {
    if !include {
        return Client::new(FileQueryHost::new(files))
            .work_axes()
            .list()
            .unwrap()
            .axes
            .unwrap();
    }
    typed_axis_options(
        files,
        ListWorkAxesArgs {
            include_construction: true,
            ..Default::default()
        },
    )
    .unwrap()
    .axes
    .unwrap()
}
fn typed_axis_options(
    files: &DiscoveryWorkspace,
    request: ListWorkAxesArgs,
) -> Result<ListWorkAxesResult, oblikovati_api::Error> {
    Client::new(FileQueryHost::new(files))
        .work_axes()
        .list_with_options(&request)
}

#[test]
fn typed_lists_include_the_origin_frame_without_recording_edits() {
    let (files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    let axes = axis_rows(&files, false);
    assert_eq!(
        axes.iter()
            .map(|axis| axis.ref_.as_str())
            .collect::<Vec<_>>(),
        [WORK_REF_X_AXIS, WORK_REF_Y_AXIS, WORK_REF_Z_AXIS]
    );
    assert!(axes.iter().all(|axis| axis.is_origin && axis.healthy));
    let points = point_rows(&files, false);
    assert_eq!(points[0].ref_, WORK_REF_CENTER);
    assert_eq!(points[0].position, Some(vec![0.0; 3]));
    assert!(!points[0].visible);
    assert_eq!(owner_revision(&files, document), before);
    assert!(DATUM_COMMAND_METHODS.contains(&METHOD_WORK_POINTS_CREATE));
    assert!(DATUM_QUERY_METHODS.contains(&METHOD_WORK_AXES_LIST));
}
#[test]
fn typed_creations_share_permanent_keys_and_preserve_hidden_collection_indices() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let point = create_point(&mut files, &mut source, [0.0, 0.0, 4.0], false);
    let line = create_line(&mut files, &mut source);
    assert_eq!((point.index, line.index), (1, 3));
    let keys = (point.ref_.as_str(), line.ref_.as_str());
    assert_eq!(keys, ("datum/v1/1", "datum/v1/2"));
    let hidden = create_point(&mut files, &mut source, [1.0, 2.0, 3.0], true);
    let axis = create_two_points(&mut files, &mut source, WORK_REF_CENTER, &point.ref_, true);
    assert!(axis.healthy);
    assert_eq!(axis.index, 4);
    assert_eq!(point_rows(&files, false).len(), 2);
    assert_eq!(point_rows(&files, true)[2].ref_, hidden.ref_);
    assert_eq!(axis_rows(&files, false).len(), 4);
    assert_eq!(
        axis_rows(&files, true)[4].direction,
        Some(vec![0.0, 0.0, 1.0])
    );
}
fn change_point(files: &mut DiscoveryWorkspace, document: u64, id: u64, position: [f64; 3]) {
    let definition = WorkGeometryDefinition::Point(position.map(DatumLength::Constant));
    files
        .commands(document)
        .unwrap()
        .run("Move point", |snapshot, _| {
            snapshot
                .work_geometry
                .replace(FeatureId(id), definition, definition.reads())
        })
        .unwrap();
}
#[test]
fn well_formed_degenerate_axes_persist_and_never_list_stale_geometry() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let first = create_point(&mut files, &mut source, [0.0, 0.0, 4.0], false);
    let second = create_point(&mut files, &mut source, [0.0, 0.0, 4.0], false);
    let axis = create_two_points(&mut files, &mut source, &first.ref_, &second.ref_, false);
    assert!(!axis.healthy);
    assert!(axis.reason.contains("nonzero"));
    assert_eq!(axis_rows(&files, false)[3].direction, None);
    change_point(&mut files, document, 2, [0.0, 0.0, 8.0]);
    assert!(axis_rows(&files, false)[3].healthy);
    change_point(&mut files, document, 2, [0.0, 0.0, 4.0]);
    let failed = &axis_rows(&files, false)[3];
    assert!(!failed.healthy);
    assert_eq!((&failed.origin, &failed.direction), (&None, &None));
}
fn assert_datum_decline<T>(reply: Result<T, oblikovati_api::Error>, expected: &str) {
    let error = reply.err().expect("expected datum command decline");
    assert!(error.to_string().contains(expected), "{error}");
}
#[test]
fn malformed_and_unavailable_constructor_requests_do_not_mutate_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    reject_point_requests(&mut files, &mut source);
    reject_axis_requests(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(point_rows(&files, false).len(), 1);
    assert_eq!(axis_rows(&files, false).len(), 3);
}
fn reject_point_requests(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    assert_datum_decline(
        client.work_points().create(&CreateWorkPointArgs {
            at: Some(vec![1.0, 2.0]),
            ..Default::default()
        }),
        "exactly three finite",
    );
    assert_datum_decline(
        client.work_points().create(&CreateWorkPointArgs {
            kind: WORK_POINT_PLANE_AXIS_INTERSECTION.0.into_owned(),
            ..Default::default()
        }),
        "expected supported",
    );
}
fn reject_axis_requests(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    for request in invalid_axis_requests() {
        assert!(client.work_axes().create(&request).is_err());
    }
}
fn invalid_axis_requests() -> [CreateWorkAxisArgs; 4] {
    let line = CreateWorkAxisArgs {
        kind: WORK_AXIS_LINE.0.into_owned(),
        origin: Some(vec![0.0; 3]),
        direction: Some(vec![0.0; 3]),
        ..Default::default()
    };
    let empty = CreateWorkAxisArgs {
        kind: WORK_AXIS_TWO_POINTS.0.into_owned(),
        ..Default::default()
    };
    let wrong = CreateWorkAxisArgs {
        refs: Some(vec![WORK_REF_CENTER.into(), WORK_REF_X_AXIS.into()]),
        ..empty.clone()
    };
    [line, empty, wrong, unsupported_axis_request()]
}
#[test]
fn local_part_scope_is_checked_before_datum_queries_or_commands() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_ASSEMBLY, Default::default());
    assert_datum_decline(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .work_points()
            .create(&CreateWorkPointArgs::default()),
        "expected",
    );
    assert_datum_decline(
        Client::new(FileQueryHost::new(&files)).work_axes().list(),
        "expected",
    );
    assert_occurrence_declined();
}
fn assert_occurrence_declined() {
    let (files, _, _) = configured_files(DOCUMENT_PART, Default::default());
    assert_datum_decline(
        typed_point_options(
            &files,
            ListWorkPointsArgs {
                occurrence: Some(vec!["placement".into()]),
                ..Default::default()
            },
        ),
        "local part scope",
    );
}
#[path = "datums_api/history.rs"]
mod history;

fn unsupported_axis_request() -> CreateWorkAxisArgs {
    CreateWorkAxisArgs {
        kind: WORK_AXIS_PLANE_INTERSECTION.0.into_owned(),
        ..Default::default()
    }
}

#[path = "datums_api/on_point.rs"]
mod on_point;

#[path = "datums_api/display.rs"]
mod display;
#[path = "datums_api/flip.rs"]
mod flip;
#[path = "datums_api/offset.rs"]
mod offset;
#[path = "datums_api/planes.rs"]
mod planes;

#[path = "datums_api/parallel.rs"]
mod parallel;
