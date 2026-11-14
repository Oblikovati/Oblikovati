// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn sketch_on_face_plane_tracks_the_part_and_history() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let bodies = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap();
    let face = bodies[0]
        .faces
        .as_ref()
        .unwrap()
        .iter()
        .find(|face| (face.point.as_ref().unwrap()[2] - 2.).abs() < 1e-9)
        .unwrap();
    let plane = Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![face.key.clone()]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let sketch = Client::new(owner.document_host())
        .sketch()
        .create(&CreateSketchArgs {
            work_plane_index: Some(Box::new(plane.index)),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(sketch.sketch_index, 1);
    let info = Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs { sketch_index: 1 })
        .unwrap();
    assert!(info.healthy);
    assert_eq!(info.dof, 0);
    let scene = owner.take_sketch_scene().unwrap().unwrap();
    assert!(
        scene.plane_declines.is_empty(),
        "{:?}",
        scene.plane_declines
    );
    assert_eq!(scene.sketches.len(), 2);
    let placed = &scene.sketches[1];
    assert!((placed.plane.recipe().origin[2] - 2.5).abs() < 1e-9);
    let sketch_id = placed.sketch;
    owner.begin_sketch_edit(document, sketch_id).unwrap();
    owner.finish_sketch_edit().unwrap();
    set_parameter(&mut owner, "depth", "3 cm");
    let updated = owner.take_sketch_scene().unwrap().unwrap();
    assert!(
        updated.plane_declines.is_empty(),
        "{:?}",
        updated.plane_declines
    );
    let plane = &updated
        .sketches
        .iter()
        .find(|item| item.sketch == sketch_id)
        .unwrap()
        .plane;
    assert!((plane.recipe().origin[2] - 3.5).abs() < 1e-9);
    use oblikovati_automation::PartUpdateAccess;
    let datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let health = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .document_sketch_health_with_datums(sketch_id, &datums)
        .unwrap();
    assert!(
        matches!(
            health,
            oblikovati_model::document_sketch::PlacedSketchHealth::Current(
                oblikovati_model::document_sketch::SketchEvaluation::Numerical(_)
            )
        ),
        "{health:?}"
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_scene_height(&mut owner, sketch_id, 2.5);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_scene_height(&mut owner, sketch_id, 3.5);
    let path = "/memory/face-sketch.opd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_scene_height(&mut owner, sketch_id, 3.5);
    let info = Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs { sketch_index: 1 })
        .unwrap();
    assert!(info.healthy);
    set_parameter(&mut owner, "depth", "0 cm");
    let failed = owner.take_sketch_scene().unwrap().unwrap();
    assert!(failed.plane_declines.iter().any(|(id, _)| *id == sketch_id));
    set_parameter(&mut owner, "depth", "3 cm");
    assert_scene_height(&mut owner, sketch_id, 3.5);
}

fn assert_scene_height(
    owner: &mut TestOwner,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    expected: f64,
) {
    let scene = owner.take_sketch_scene().unwrap().unwrap();
    assert!(
        scene.plane_declines.is_empty(),
        "{:?}",
        scene.plane_declines
    );
    let plane = &scene
        .sketches
        .iter()
        .find(|item| item.sketch == sketch)
        .unwrap()
        .plane;
    assert!((plane.recipe().origin[2] - expected).abs() < 1e-9);
}

#[test]
fn part_update_repairs_face_sketch_inputs_inside_an_open_group() {
    use oblikovati_automation::PartUpdateAccess;
    use oblikovati_model::{
        document_sketch::{PlacedSketchHealth, SketchEvaluation, SketchPlacement},
        history::DocumentSketchCreation,
    };
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let faces = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap();
    let top = faces[0]
        .faces
        .as_ref()
        .unwrap()
        .iter()
        .find(|face| (face.point.as_ref().unwrap()[2] - 2.).abs() < 1e-9)
        .unwrap();
    let plane = Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![top.key.clone()]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let plane_id = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[1]
        .id();
    let request = DocumentSketchCreation {
        name: "Face sketch".into(),
        placement: SketchPlacement::WorkPlane(plane_id),
        orientation: None,
        project_origin: true,
    };
    let sketch = owner
        .files
        .commands(document)
        .unwrap()
        .run("Create face sketch", |snapshot, ids| {
            snapshot.create_document_sketch_with_datums(&request, ids, &datums)
        })
        .unwrap();
    owner.begin_sketch_edit(document, sketch).unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    let watch = owner.sketch_edit.as_ref().unwrap().1.clone();
    assert_eq!(plane.index, 3);
    let undo_count = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Edit size")
        .unwrap();
    set_parameter(&mut owner, "depth", "3 cm");
    Client::new(owner.document_host())
        .documents()
        .update(&UpdateDocumentArgs::default())
        .unwrap();
    let datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let snapshot = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let health = snapshot
        .document_sketch_health_with_datums(sketch, &datums)
        .unwrap();
    assert!(
        matches!(
            health,
            PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
        ),
        "{health:?}"
    );
    owner.files.commands(document).unwrap().end().unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .undo_labels()
            .len(),
        undo_count + 1
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_face_sketch_current(&mut owner, document, sketch, 2.5);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_face_sketch_current(&mut owner, document, sketch, 3.5);
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Abort size")
        .unwrap();
    set_parameter(&mut owner, "depth", "4 cm");
    assert_face_sketch_current(&mut owner, document, sketch, 4.5);
    owner.files.commands(document).unwrap().abort().unwrap();
    assert_face_sketch_current(&mut owner, document, sketch, 3.5);
    assert!(watch.is_live());
    set_parameter(&mut owner, "depth", "0 cm");
    assert!(!watch.is_live());
    assert!(owner.active_sketch_edit().is_none());
    let failed_datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let snapshot = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &failed_datums)
            .is_err()
    );
    set_parameter(&mut owner, "depth", "3 cm");
    assert!(!watch.is_live());
    assert_face_sketch_current(&mut owner, document, sketch, 3.5);
    set_parameter(&mut owner, "depth", "5 cm");
    let mut headless = oblikovati_automation::PartReplayService::default();
    let datums = headless
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let snapshot = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(
        (snapshot
            .resolve_sketch_plane_with_datums(sketch, &datums)
            .unwrap()
            .recipe()
            .origin[2]
            - 5.5)
            .abs()
            < 1e-9
    );
    assert!(matches!(
        snapshot
            .document_sketch_health_with_datums(sketch, &datums)
            .unwrap(),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
    ));
}
fn assert_face_sketch_current(
    owner: &mut TestOwner,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    expected: f64,
) {
    use oblikovati_automation::PartUpdateAccess;
    use oblikovati_model::document_sketch::{PlacedSketchHealth, SketchEvaluation};
    let datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    let snapshot = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let plane = snapshot
        .resolve_sketch_plane_with_datums(sketch, &datums)
        .unwrap();
    assert!((plane.recipe().origin[2] - expected).abs() < 1e-9);
    let health = snapshot
        .document_sketch_health_with_datums(sketch, &datums)
        .unwrap();
    assert!(
        matches!(
            health,
            PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
        ),
        "{health:?}"
    );
}

#[test]
fn desktop_creates_an_editable_sketch_on_a_face_plane() {
    use super::super::super::slot_input::deliver;
    use crate::{SketchPlacementTool, placement_test_inputs::*};
    use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let bodies = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap();
    let key = bodies[0]
        .faces
        .as_ref()
        .unwrap()
        .iter()
        .find(|face| (face.point.as_ref().unwrap()[2] - 2.).abs() < 1e-9)
        .unwrap()
        .key
        .clone();
    Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![key]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    let plane = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()[1]
        .id();
    let request = DocumentSketchCreation {
        name: String::new(),
        placement: SketchPlacement::WorkPlane(plane),
        orientation: None,
        project_origin: true,
    };
    let sketch = owner
        .create_automatic_sketch_for_edit(document, &request)
        .unwrap();
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketch);
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.dimensions_needed, Some(0));
    let row = state
        .browser
        .unwrap()
        .sketches
        .into_iter()
        .find(|row| row.id == sketch)
        .unwrap();
    assert!(row.inspection.unwrap().healthy);
    let scene = owner.take_sketch_scene().unwrap().unwrap();
    assert!(scene.plane_declines.is_empty());
    assert!(
        (scene
            .sketches
            .iter()
            .find(|item| item.sketch == sketch)
            .unwrap()
            .plane
            .recipe()
            .origin[2]
            - 2.5)
            .abs()
            < 1e-9
    );
    owner
        .start_sketch_placement(document, sketch, SketchPlacementTool::Rectangle)
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            motion([10, 5]),
            button([10, 5], true),
        ],
    );
    let notice = owner.take_command_notice();
    assert!(
        !matches!(
            notice,
            Some(crate::DesktopCommandNotice::PlacementFailed { .. })
        ),
        "rectangle placement failed: {notice:?}"
    );
    let snapshot = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .lines()
            .len(),
        4
    );
    let state = owner.gui_command_state().unwrap();
    assert!(state.dimensions_needed.is_some_and(|count| count > 0));
    assert!(
        state
            .browser
            .unwrap()
            .sketches
            .into_iter()
            .find(|row| row.id == sketch)
            .unwrap()
            .inspection
            .unwrap()
            .healthy
    );
    owner.finish_sketch_edit().unwrap();
    set_parameter(&mut owner, "depth", "3 cm");
    assert_scene_height(&mut owner, sketch, 3.5);
    assert_face_sketch_current(&mut owner, document, sketch, 3.5);
}
