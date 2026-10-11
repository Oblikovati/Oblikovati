// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn input() -> AddSketchEntityArgs {
    let mut input = literal_input("ellipse", &[[1.0, 2.0]]);
    input.axis = Some(vec![3.0, 4.0]);
    input.major_radius = "30 mm".into();
    input.minor_radius = "width".into();
    input.construction = true;
    input
}
#[test]
fn ellipse_creation_returns_owned_center_and_curve_in_both_document_kinds() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        create_empty_sketch(&mut files, &mut source);
        let result = add_call(&mut files, &mut source, &input()).unwrap();
        assert_eq!(result.kind, "ellipse");
        assert_eq!(result.point_i_ds.as_ref().unwrap().len(), 1);
        let inspected = row(&files, result.entity_id);
        assert_eq!(inspected.points, expected_points(&[[1.0, 2.0]]));
        assert_eq!(inspected.radius, 0.0);
        assert!(inspected.construction);
        verify_keys(&files, &rows(&files));
        assert_frame(&files, document, result.entity_id, 1.0);
    }
}
fn assert_frame(files: &DiscoveryWorkspace, document: u64, id: u64, scale: f64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let geometry = snapshot.sketches.sketches()[0].geometry();
    let ellipse = geometry.ellipse(SketchEntityId(id)).unwrap();
    assert_eq!(ellipse.major_axis(), [0.6, 0.8]);
    assert_eq!(ellipse.radii(), [3.0 / scale, 1.0 / scale]);
    assert_eq!(
        geometry.point(ellipse.center()).unwrap().position,
        [1.0 / scale, 2.0 / scale]
    );
}
#[test]
fn ellipse_inputs_convert_lengths_but_keep_axis_direction_unitless() {
    for scale in [0.1, 2.0, 1e6] {
        let (mut files, document, mut source) = configured_scaled(scale);
        create_empty_sketch(&mut files, &mut source);
        let mut request = input();
        request.variant = "ignored by Go ellipse constructor".into();
        let result = add_call(&mut files, &mut source, &request).unwrap();
        assert_frame(&files, document, result.entity_id, scale);
    }
}
#[test]
fn bad_ellipse_frames_and_expressions_refuse_before_identity_reservation() {
    let (mut files, document, mut source) = fresh();
    let before = owner_revision(&files, document);
    for axis in [None, Some(vec![]), Some(vec![1.0]), Some(vec![0.0, 0.0])] {
        let mut request = input();
        request.axis = axis;
        assert!(add_call(&mut files, &mut source, &request).is_err());
    }
    refuse_bad_radii(&mut files, &mut source);
    assert_eq!(owner_revision(&files, document), before);
    let result = add_call(&mut files, &mut source, &input()).unwrap();
    assert_creation(&result, "ellipse", 3, &[2]);
}
fn refuse_bad_radii(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    for (major, minor) in [
        ("0 cm", "1 cm"),
        ("1 cm", "2 cm"),
        ("90 deg", "1 cm"),
        ("missing", "1 cm"),
    ] {
        let mut request = input();
        request.major_radius = major.into();
        request.minor_radius = minor.into();
        assert!(add_call(files, source, &request).is_err());
    }
}
#[test]
fn ellipse_creation_survives_history_reopen_copy_and_accepts_a_dimension() {
    let (mut files, document, mut source) = fresh();
    let result = add_call(&mut files, &mut source, &input()).unwrap();
    let expected = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert!(rows(&files).len() < expected.len());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), expected);
    check_saved_copy(&mut files, document, &mut source, &expected);
    check_major_dimension(&mut files, &mut source, result.entity_id);
}
fn check_saved_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    expected: &[SketchEntityInfo],
) {
    let document = reopen_with_observer(files, document, source, Default::default());
    assert_eq!(rows(files), expected);
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/ellipse-entity-copy.obk"),
    );
    assert_copy_namespaces(expected, &rows(files));
    verify_keys(files, &rows(files));
}
fn check_major_dimension(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, id: u64) {
    let dimension = Client::new(FileHost::new(files, source, "test"))
        .dimension()
        .add(&AddDimensionArgs {
            kind: "ellipseRadius".into(),
            entities: Some(vec![id]),
            expression: "4 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!((dimension.value - 4.0).abs() < 1e-8);
}

#[test]
fn ellipse_expression_inputs_seed_geometry_from_current_parameters_without_live_bindings() {
    let (mut files, document, mut source) = fresh();
    let mut request = input();
    request.points = Some(vec![None]);
    request.point_exprs = Some(vec![Some(vec!["width".into(), "width * 2".into()])]);
    let before = owned_parameters(&files, document).clone();
    let created = add_call(&mut files, &mut source, &request).unwrap();
    assert_frame(&files, document, created.entity_id, 1.0);
    assert!(owned_parameters(&files, document).same_recipe(&before));
    set_width(&mut files, &mut source, "2 cm");
    assert_frame(&files, document, created.entity_id, 1.0);
    let second = add_call(&mut files, &mut source, &request).unwrap();
    check_reseeded_ellipse(&files, document, second.entity_id);
}
fn check_reseeded_ellipse(files: &DiscoveryWorkspace, document: u64, id: u64) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    assert_eq!(
        geometry.ellipse(SketchEntityId(id)).unwrap().radii(),
        [3.0, 2.0]
    );
    assert_eq!(row(files, id).points, expected_points(&[[2.0, 4.0]]));
}
