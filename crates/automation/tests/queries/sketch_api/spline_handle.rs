// SPDX-License-Identifier: GPL-2.0-only
use super::spline_requests::fit_request;
use super::*;

fn set_handle(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: &SetSplineHandleArgs,
) -> Result<SplineHandleInfo, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .set_spline_handle(request)
}

#[test]
fn typed_fit_spline_handles_activate_edit_and_deactivate() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    let spline = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .add_entity(&fit_request("smooth", false))
        .unwrap();
    let mut request = SetSplineHandleArgs {
        spline: spline.entity_id,
        fit_point_index: 2,
        active: true,
        ..Default::default()
    };
    let natural = set_handle(&mut files, &mut source, &request).unwrap();
    assert!(natural.handle_id > 0);
    assert_eq!(natural.tangent.as_ref().map(Vec::len), Some(2));
    assert_eq!(natural.weight, 1.0);
    request.tangent = Some(vec![3.0, 4.0]);
    request.weight = 2.0;
    let edited = set_handle(&mut files, &mut source, &request).unwrap();
    assert_eq!(edited.handle_id, natural.handle_id);
    assert_eq!(edited.tangent, Some(vec![0.6, 0.8]));
    assert!((edited.weight - 2.0).abs() < 1e-12);
    request.active = false;
    let inactive = set_handle(&mut files, &mut source, &request).unwrap();
    assert_eq!(inactive, SplineHandleInfo::default());
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_SET_SPLINE_HANDLE));
}

#[test]
fn malformed_spline_handle_request_does_not_create_history() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .create(&Default::default())
        .unwrap();
    let spline = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .add_entity(&fit_request("chord", false))
        .unwrap();
    let before = owner_revision(&files, document);
    let request = SetSplineHandleArgs {
        spline: spline.entity_id,
        fit_point_index: 1,
        active: true,
        tangent: Some(vec![1.0]),
        ..Default::default()
    };
    expect_api_decline(
        set_handle(&mut files, &mut source, &request),
        "expected [x, y]",
    );
    assert_eq!(owner_revision(&files, document), before);
}
