// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEllipse2D;
#[path = "ellipse/acceptance.rs"]
mod acceptance;
fn ellipse_owner(owner: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = fresh(owner);
    files
        .commands(document)
        .unwrap()
        .run("Owned ellipse", install_ellipse)
        .unwrap();
    (files, document, source)
}
fn install_ellipse(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let id = ids.allocate_sketch_entity().unwrap();
    assert_eq!(id, entity(12));
    let value = SketchEllipse2D::new(id, entity(1), [3.0, 4.0], [3.0, 1.0]).unwrap();
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry
        .add_ellipse(value)
        .map_err(|cause| cause.to_string())?;
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .map_err(|cause| cause.to_string())
}

fn ellipse_request(reference: bool) -> AddDimensionArgs {
    let mut input = dimension_request("ellipseRadius", &[12]);
    input.expression = "width * 4".into();
    input.driven = reference;
    input
}
#[test]
fn ellipse_major_radius_uses_live_owned_formula_and_changes_only_its_major_variable() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_driving_ellipse(owner);
    }
}
fn assert_driving_ellipse(owner: DocumentType) {
    let (mut files, document, mut source) = ellipse_owner(owner);
    let result = create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    assert!((result.value - 4.0).abs() < 1e-8);
    assert_eq!(result.kind, "ellipseRadius");
    assert_eq!(result.parameter, "d0");
    let row = &dimensions::dimensions(&files)[0];
    assert_eq!(row.expression, "width * 4");
    assert!(!row.driven);
    assert_eq!(row.text_point, Some(vec![2.5, -7.25]));
    change_source(&mut files, document);
    assert!((dimensions::dimensions(&files)[0].value - 6.0).abs() < 1e-8);
    assert_ellipse_frame(&files, document);
    assert_eq!(result.dof, status_call(&files, 0).unwrap().dof);
}
fn assert_ellipse_frame(files: &DiscoveryWorkspace, document: u64) {
    let geometry = owned_geometry(files, document)
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let shape = geometry.ellipse(entity(12)).unwrap();
    assert_eq!(shape.radii()[1], 1.0);
    assert_eq!(shape.major_axis(), [0.6, 0.8]);
}

#[test]
fn ellipse_reference_role_measures_geometry_and_survives_undo_reopen_and_copy() {
    let (mut files, document, mut source) = ellipse_owner(DOCUMENT_PART);
    let result = create_dimension(&mut files, &mut source, &ellipse_request(true)).unwrap();
    assert_eq!(result.value, 3.0);
    let expected = dimensions::dimensions(&files);
    assert!(expected[0].driven);
    files.commands(document).unwrap().undo().unwrap();
    assert!(dimensions::dimensions(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(dimensions::dimensions(&files), expected);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(dimensions::dimensions(&files), expected);
    super::super::super::super::entities::activate_copy(
        &mut files,
        document,
        &mut source,
        Path::new("/fixtures/ellipse-dimension-copy.obk"),
    );
    assert_eq!(dimensions::dimensions(&files), expected);
}
#[test]
fn ellipse_constructor_rejects_wrong_shapes_and_radius_crossings_atomically() {
    let (mut files, document, mut source) = ellipse_owner(DOCUMENT_PART);
    for refs in [vec![], vec![9], vec![99], vec![12, 12]] {
        let before = owner_revision(&files, document);
        let mut input = ellipse_request(false);
        input.entities = Some(refs);
        assert!(create_dimension(&mut files, &mut source, &input).is_err());
        assert_eq!(owner_revision(&files, document), before);
    }
    assert_radius_crossing(&mut files, document, &mut source);
}
fn assert_radius_crossing(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    let before = owned_geometry(files, document).clone();
    let mut input = ellipse_request(false);
    input.expression = "0.5 cm".into();
    assert!(create_dimension(files, source, &input).is_err());
    assert!(before.same_recipe(owned_geometry(files, document)));
    assert!(dimensions::dimensions(files).is_empty());
    let next = create_dimension(files, source, &ellipse_request(false)).unwrap();
    assert_eq!(next.parameter, "d1");
}

#[test]
fn ellipse_dimension_edit_delete_and_recreation_use_the_normal_owner_path() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_ellipse_edit_delete(owner);
    }
}
fn assert_ellipse_edit_delete(owner: DocumentType) {
    let (mut files, _, mut source) = ellipse_owner(owner);
    create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .dimension()
        .edit(&oblikovati_api::wire::DriveDimensionArgs {
            expression: "5 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!((dimensions::dimensions(&files)[0].value - 5.0).abs() < 1e-8);
    Client::new(FileHost::new(&mut files, &mut source, "test"))
        .dimension()
        .delete(&Default::default())
        .unwrap();
    assert!(dimensions::dimensions(&files).is_empty());
    let next = create_dimension(&mut files, &mut source, &ellipse_request(false)).unwrap();
    assert_eq!(next.parameter, "d1");
    assert_eq!(next.index, 0);
}
