// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchSelectionTransform;
fn rotation(entities: Vec<u64>, center: Vec<f64>, angle: &str) -> TransformSketchArgs {
    TransformSketchArgs {
        op: "rotate".into(),
        entities: Some(entities),
        center: Some(center),
        angle: angle.into(),
        ..Default::default()
    }
}
fn assert_quarter_turn(before: &[SketchEntityInfo], after: &[SketchEntityInfo], center: [f64; 2]) {
    assert_eq!(before.len(), after.len());
    for (old, new) in before.iter().zip(after) {
        assert_rotated_row(old, new, center);
    }
}
fn assert_rotated_row(old: &SketchEntityInfo, new: &SketchEntityInfo, center: [f64; 2]) {
    let (old_points, new_points) = (old.points.as_ref().unwrap(), new.points.as_ref().unwrap());
    assert_eq!(old_points.len(), new_points.len());
    for (xy, found) in old_points.iter().zip(new_points) {
        let xy = xy.as_ref().unwrap();
        let want = [center[0] + center[1] - xy[1], center[1] + xy[0] - center[0]];
        near_pair(found.as_ref().unwrap(), want);
    }
    let mut expected = old.clone();
    expected.points = new.points.clone();
    assert_eq!(expected, *new);
}
fn near_pair(found: &[f64], expected: [f64; 2]) {
    assert_eq!(found.len(), 2);
    for (actual, want) in found.iter().zip(expected) {
        assert!((actual - want).abs() < 1e-12, "{found:?} != {expected:?}");
    }
}
#[test]
fn canonical_rotation_converts_centers_and_preserves_keys_with_shared_points() {
    for scale in [0.1, 1.0, 2.0] {
        let (mut files, document, mut source) = configured_scaled(scale);
        install_mixed(&mut files, document);
        let before = rows(&files);
        let input = rotation(all_ids(&files), vec![1.0, 2.0], "45 deg * 2");
        assert!(
            move_call(&mut files, &mut source, &input)
                .unwrap()
                .created
                .is_none()
        );
        assert_quarter_turn(&before, &rows(&files), [1.0, 2.0]);
        verify_keys(&files, &before);
        assert_eq!(
            owner_revision(&files, document).1.last().unwrap(),
            "Rotate sketch entities"
        );
    }
}
#[test]
fn rotate_undo_redo_group_abort_and_saved_reload_restore_exact_recipes() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let before = rows(&files);
    let input = rotation(all_ids(&files), vec![1.0, 2.0], "90 deg");
    begin_move_group(&mut files, document);
    move_call(&mut files, &mut source, &input).unwrap();
    files.commands(document).unwrap().abort().unwrap();
    assert_eq!(rows(&files), before);
    move_call(&mut files, &mut source, &input).unwrap();
    let turned = rows(&files);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(rows(&files), before);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(rows(&files), turned);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), turned);
    assert!(!files.workspace().needs_save(document).unwrap());
}
#[test]
fn invalid_centers_preserve_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    for center in [vec![], vec![1.0], vec![1.0, 2.0, 3.0]] {
        let input = rotation(vec![1], center, "90 deg");
        assert_move_refused(&mut files, document, &mut source, &input, "two finite");
    }
}
#[test]
fn invalid_angles_and_quantity_categories_preserve_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    for angle in ["", "1 cm", "unknown", "1 rad / 0"] {
        let input = rotation(vec![1], vec![0.0; 2], angle);
        assert_move_refused(&mut files, document, &mut source, &input, "angle");
    }
}
#[test]
fn invalid_rotation_selections_preserve_the_owner() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    for (selection, message) in [(vec![1, 999], "999"), (vec![], "at least one")] {
        let input = rotation(selection, vec![0.0; 2], "90 deg");
        assert_move_refused(&mut files, document, &mut source, &input, message);
    }
}
#[test]
fn rejected_rotation_reply_discards_coordinates_and_evidence() {
    let (mut files, document, _) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let original = owned_geometry(&files, document).clone();
    let before = owner_revision(&files, document);
    reject_rotation_reply(&mut files, document);
    assert!(original.same_evaluated_state(owned_geometry(&files, document)));
    assert_eq!(owner_revision(&files, document), before);
}
#[test]
fn grounded_circle_centers_remain_controlled_in_part_and_assembly() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let sketch = create_owner(&mut files, document);
        let circle = create_circle(&mut files, document, sketch);
        let before = rows(&files);
        move_call(
            &mut files,
            &mut source,
            &rotation(vec![circle.entity.0], vec![3.0, 4.0], "90 deg"),
        )
        .unwrap();
        assert_eq!(rows(&files), before);
    }
}
#[test]
fn veto_and_read_only_host_refuse_rotation_without_saved_state_changes() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = rows(&files);
    let input = rotation(all_ids(&files), vec![0.0; 2], "90 deg");
    expect_api_decline(move_call(&mut files, &mut source, &input), "locked");
    assert_eq!(rows(&files), before);
    assert!(!files.workspace().needs_save(document).unwrap());
    assert_query_move_refused(&files, &input);
}
#[test]
fn named_angle_is_resolved_once_and_bare_angle_uses_the_document_unit() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    install_mixed(&mut files, document);
    files
        .commands(document)
        .unwrap()
        .run("Angle source", author_rotation_angle)
        .unwrap();
    let before = rows(&files);
    let ids = all_ids(&files);
    let input = rotation(ids.clone(), vec![0.0; 2], "turn_angle * 2");
    move_call(&mut files, &mut source, &input).unwrap();
    assert_quarter_turn(&before, &rows(&files), [0.0; 2]);
    files.commands(document).unwrap().undo().unwrap();
    move_call(&mut files, &mut source, &rotation(ids, vec![0.0; 2], "90")).unwrap();
    assert_quarter_turn(&before, &rows(&files), [0.0; 2]);
}
fn author_rotation_angle(
    snapshot: &mut DocumentSnapshot,
    reservations: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<(), String> {
    let id = reservations.allocate().map_err(|cause| cause.to_string())?;
    snapshot
        .parameters
        .insert_expression(
            id,
            "turn_angle",
            oblikovati_api::types::USER_PARAM,
            "45 deg",
        )
        .map_err(|cause| cause.to_string())
}

fn reject_rotation_reply(files: &mut DiscoveryWorkspace, document: u64) {
    let sketch = owned_geometry(files, document).sketches()[0].id();
    let transform = SketchSelectionTransform::Rotate {
        center: [0.0; 2],
        radians: 1.0,
    };
    let result = files
        .commands(document)
        .unwrap()
        .transform_sketch_entities_projected(sketch, &[SketchEntityId(3)], transform, |result| {
            assert_eq!(result.unwrap(), [SketchEntityId(1), SketchEntityId(2)]);
            Err::<(), _>("reply refused")
        });
    assert!(result.is_err());
}
