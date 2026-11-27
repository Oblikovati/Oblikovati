// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchEllipse2D;
fn ground_owner(owner: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = fresh(owner);
    files
        .commands(document)
        .unwrap()
        .run("Ellipse ground operands", install_ground_ellipse)
        .unwrap();
    (files, document, source)
}
fn install_ground_ellipse(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let id = ids
        .allocate_sketch_entity()
        .map_err(|cause| cause.to_string())?;
    assert_eq!(id, entity(12));
    let mut geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry
        .add_ellipse(SketchEllipse2D::new(id, entity(1), [3.0, 4.0], [3.0, 1.0]).unwrap())
        .unwrap();
    snapshot
        .sketches
        .replace_geometry(DocumentSketchId(1), geometry)
        .map_err(|cause| cause.to_string())
}
#[test]
fn public_ground_freezes_defining_points_in_both_document_owners() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        for (curve, points) in [
            (2, vec![2]),
            (7, vec![1, 2]),
            (9, vec![1]),
            (11, vec![1, 2, 3]),
            (12, vec![1]),
        ] {
            assert_ground_creation(owner, curve, &points);
        }
    }
}
fn assert_ground_creation(owner: DocumentType, curve: u64, points: &[u64]) {
    let (mut files, document, mut source) = ground_owner(owner);
    let before = status_call(&files, 0).unwrap().dof;
    let result = add(&mut files, &mut source, &request("ground", &[curve])).unwrap();
    assert_eq!(result.kind, "ground");
    let reduction = if curve == 11 { 5 } else { points.len() * 2 };
    assert_eq!(result.dof, before - reduction as i64);
    assert_eq!(result.index, 0);
    let row = &list(&files)[0];
    assert_eq!(row.entities.as_deref(), Some(points));
    assert_ground_recipe(&files, document, points);
}
fn assert_ground_recipe(files: &DiscoveryWorkspace, document: u64, expected: &[u64]) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let PlanarConstraint::Ground { points } = &geometry.constraints()[0].relation else {
        panic!("expected frozen ground anchors");
    };
    assert_eq!(
        points.iter().map(|point| point.id.0).collect::<Vec<_>>(),
        expected
    );
    for point in points {
        assert_eq!(point.position, geometry.point(point.id).unwrap().position);
    }
    assert_eq!(geometry.ellipse(entity(12)).unwrap().radii(), [3.0, 1.0]);
    assert_eq!(
        geometry.ellipse(entity(12)).unwrap().major_axis(),
        [0.6, 0.8]
    );
}
#[test]
fn ground_history_reopen_copy_and_delete_keep_frozen_points_and_retired_ids() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_ground_history(owner);
    }
}
fn assert_ground_history(owner: DocumentType) {
    let (mut files, document, mut source) = ground_owner(owner);
    add(&mut files, &mut source, &request("ground", &[12])).unwrap();
    let expected = list(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert!(list(&files).is_empty());
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(list(&files), expected);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_ground_recipe(&files, document, &[1]);
    copy_ground(&mut files, document, &mut source);
    assert_eq!(list(&files), expected);
    delete_ground(&mut files, &mut source);
}
fn copy_ground(files: &mut DiscoveryWorkspace, document: u64, source: &mut QueryGuids) {
    super::super::super::entities::activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/ground-copy.obk"),
    );
    let copy = files.workspace().active_document().unwrap();
    assert_ground_recipe(files, copy, &[1]);
}
fn delete_ground(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    Client::new(FileHost::new(files, source, "test"))
        .constrain()
        .delete(&DeleteConstraintArgs {
            sketch_index: 0,
            constraint_index: 0,
        })
        .unwrap();
    assert!(list(files).is_empty());
    add(files, source, &request("ground", &[12])).unwrap();
    let document = files.workspace().active_document().unwrap();
    assert_eq!(
        owned_geometry(files, document).sketches()[0]
            .geometry()
            .constraints()[0]
            .id,
        SketchConstraintId(2)
    );
}
#[test]
fn ground_bad_operands_refuse_before_reserving_a_constraint() {
    let (mut files, document, mut source) = ground_owner(DOCUMENT_PART);
    let before = owner_revision(&files, document);
    for refs in [vec![], vec![1, 2], vec![99]] {
        assert!(add(&mut files, &mut source, &request("ground", &refs)).is_err());
        assert_eq!(owner_revision(&files, document), before);
    }
    add(&mut files, &mut source, &request("ground", &[12])).unwrap();
    assert_eq!(
        owned_geometry(&files, document).sketches()[0]
            .geometry()
            .constraints()[0]
            .id,
        SketchConstraintId(1)
    );
}
#[test]
fn ground_aborted_group_retires_identity_without_adopting_anchors() {
    let (mut files, document, mut source) = ground_owner(DOCUMENT_PART);
    let before = owned_geometry(&files, document).clone();
    files
        .commands(document)
        .unwrap()
        .begin("Ground group")
        .unwrap();
    add(&mut files, &mut source, &request("ground", &[7])).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert!(list(&files).is_empty());
    assert!(before.same_evaluated_state(owned_geometry(&files, document)));
    add(&mut files, &mut source, &request("ground", &[12])).unwrap();
    assert_ground_identity(&files, document, 2);
}
fn assert_ground_identity(files: &DiscoveryWorkspace, document: u64, expected: u64) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let constraint = &geometry.constraints()[0];
    assert_eq!(constraint.id, SketchConstraintId(expected));
}
#[test]
fn ground_veto_cannot_publish_a_partial_anchor_set() {
    let (mut files, document, mut source) = ground_owner(DOCUMENT_ASSEMBLY);
    let document = reopen_with_observer(
        &mut files,
        document,
        &mut source,
        QueryObserver {
            veto: true,
            ..Default::default()
        },
    );
    let before = owned_geometry(&files, document).clone();
    expect_api_decline(
        add(&mut files, &mut source, &request("ground", &[11])),
        "locked",
    );
    assert!(list(&files).is_empty());
    assert!(before.same_evaluated_state(owned_geometry(&files, document)));
}
