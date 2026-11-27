// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::CreateWorkPlaneArgs;

#[path = "face_plane/cached_sketch.rs"]
mod cached_sketch;
#[path = "face_plane/owned_preview.rs"]
mod owned_preview;

#[test]
fn face_offset_command_binds_a_current_face_and_reports_its_geometry() {
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
        .unwrap()[0]
        .faces
        .clone()
        .unwrap();
    let face = faces
        .iter()
        .find(|face| face.kind == "plane" && (face.point.as_ref().unwrap()[2] - 2.).abs() < 1e-9)
        .unwrap();
    let result = Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![face.key.clone()]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!(result.healthy);
    let rows = Client::new(owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    let plane = &rows[result.index as usize];
    assert!(plane.healthy);
    let origin = plane.origin.as_ref().unwrap();
    let normal = plane.normal.as_ref().unwrap();
    let point = face.point.as_ref().unwrap();
    let distance: f64 = (0..3).map(|i| (origin[i] - point[i]) * normal[i]).sum();
    assert!(
        (distance - 0.5).abs() < 1e-9,
        "offset {distance}: expected 0.5 cm"
    );
    assert!((origin[2] - 2.5).abs() < 1e-9);
    set_parameter(&mut owner, "depth", "3 cm");
    assert!((plane_height(&mut owner, result.index) - 3.5).abs() < 1e-9);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert!((plane_height(&mut owner, result.index) - 2.5).abs() < 1e-9);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert!((plane_height(&mut owner, result.index) - 3.5).abs() < 1e-9);
    let path = document_fixture_path!("/memory/face-plane.opd");
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
    assert!((plane_height(&mut owner, result.index) - 3.5).abs() < 1e-9);
}

fn plane_height(owner: &mut TestOwner, index: i64) -> f64 {
    let rows = Client::new(owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    assert!(rows[index as usize].healthy);
    rows[index as usize].origin.as_ref().unwrap()[2]
}

#[test]
fn desktop_part_owner_resolves_face_again_after_a_parameter_edit() {
    use oblikovati_automation::PartUpdateAccess;
    use oblikovati_model::{
        feature::work_plane::WorkPlaneFaceReference, identity::ReferenceEnvelope,
    };
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    let mut references = published_keys(&owner, document, FeatureId(1))
        .into_iter()
        .filter(|key| ReferenceEnvelope::decode(key).unwrap().kind == 1)
        .map(|key| WorkPlaneFaceReference::new(key).unwrap());
    let reference = references
        .find(|reference| {
            let frame = owner
                .part_scenes
                .resolve_planar_face(&mut owner.files, document, reference)
                .unwrap();
            (frame.recipe().origin[2] - 2.).abs() < 1e-9 && frame.normal()[2] > 0.99
        })
        .unwrap();
    let before = owner
        .part_scenes
        .resolve_planar_face(&mut owner.files, document, &reference)
        .unwrap();
    set_parameter(&mut owner, "depth", "3 cm");
    let after = owner
        .part_scenes
        .resolve_planar_face(&mut owner.files, document, &reference)
        .unwrap();
    assert!((before.recipe().origin[2] - 2.).abs() < 1e-9);
    assert!((after.recipe().origin[2] - 3.).abs() < 1e-9);
    let plane = oblikovati_model::feature::work_plane::WorkPlaneDefinition::Offset {
        base: oblikovati_model::feature::work_plane::WorkPlaneReference::Face(reference.clone()),
        distance: DatumLength::Constant(0.5),
    };
    let plane_id = owner
        .files
        .commands(document)
        .unwrap()
        .run("Face plane", |snapshot, ids| {
            let id = ids.allocate_feature().unwrap();
            snapshot
                .work_geometry
                .append(AuthoredFeature::datum(id, "Face plane", plane))?;
            Ok::<_, oblikovati_model::feature::FeatureSequenceError>(id)
        })
        .unwrap();
    use oblikovati_model::feature::work_plane::WorkPlaneResolver;
    let datums = owner
        .part_scenes
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    assert!((datums.plane(plane_id).unwrap().recipe().origin[2] - 3.5).abs() < 1e-9);
    let rows = Client::new(owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    assert!(rows[3].healthy);
    assert!((rows[3].origin.as_ref().unwrap()[2] - 3.5).abs() < 1e-9);
    let mut headless = oblikovati_automation::PartReplayService::default();
    let replayed = headless
        .resolve_planar_face(&mut owner.files, document, &reference)
        .unwrap();
    assert!((replayed.recipe().origin[2] - 3.).abs() < 1e-9);
    set_parameter(&mut owner, "depth", "4 cm");
    let refreshed = headless
        .resolve_planar_face(&mut owner.files, document, &reference)
        .unwrap();
    assert!((refreshed.recipe().origin[2] - 4.).abs() < 1e-9);
    let datums = headless
        .datum_evaluation(&mut owner.files, document)
        .unwrap();
    assert!((datums.plane(plane_id).unwrap().recipe().origin[2] - 4.5).abs() < 1e-9);
    let unowned = Client::new(oblikovati_automation::FileQueryHost::new(owner.documents()))
        .work_planes()
        .list()
        .unwrap_err();
    assert!(
        unowned
            .to_string()
            .contains("expected an injected part replay owner")
    );
    set_parameter(&mut owner, "width", "0.000000000000001 cm");
    let rows = Client::new(owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    assert!(!rows[3].healthy);
    assert!(rows[3].origin.is_none());
    set_parameter(&mut owner, "width", "4 cm");
    let rows = Client::new(owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    assert!(rows[3].healthy);
    assert!((rows[3].origin.as_ref().unwrap()[2] - 4.5).abs() < 1e-9);
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Remove features", |snapshot, _| {
            snapshot.work_geometry.remove(plane_id)?;
            snapshot.work_geometry.remove(FeatureId(1))?;
            Ok::<_, oblikovati_model::feature::FeatureSequenceError>(())
        })
        .unwrap();
    assert!(
        headless
            .datum_evaluation(&mut owner.files, document)
            .unwrap()
            .health
            .is_empty()
    );
}

#[test]
fn face_offset_rejects_unbound_names_without_changing_the_document() {
    use oblikovati_model::identity::SemanticReferenceName;
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let unknown = SemanticReferenceName::new(guid, 99, 1, "sketch-prism", "side", &[1, 101])
        .unwrap()
        .encode()
        .unwrap();
    let wrong_kind = SemanticReferenceName::new(guid, 1, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let foreign = SemanticReferenceName::new(
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
        1,
        1,
        "sketch-prism",
        "side",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap();
    let before = owner.files.workspace().session(document).unwrap();
    let generation = before.generation();
    let features = before.snapshot().work_geometry.features().len();
    let undo = before.undo_labels().len();
    for key in [
        "xyz".to_string(),
        hex_key(&unknown),
        hex_key(&wrong_kind),
        hex_key(&foreign),
    ] {
        let error = Client::new(owner.document_host())
            .work_planes()
            .create(&CreateWorkPlaneArgs {
                kind: "plane-offset".into(),
                refs: Some(vec![key]),
                offset: "5 mm".into(),
                ..Default::default()
            });
        assert!(error.is_err());
        let session = owner.files.workspace().session(document).unwrap();
        assert_eq!(session.generation(), generation);
        assert_eq!(session.snapshot().work_geometry.features().len(), features);
        assert_eq!(session.undo_labels().len(), undo);
    }
}
fn hex_key(bytes: &[u8]) -> String {
    bytes.iter().map(|byte| format!("{byte:02x}")).collect()
}

#[test]
fn headless_face_offset_uses_its_replay_owner() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    let keys = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap();
    let key = keys[0].faces.as_ref().unwrap()[0].key.clone();
    let args = CreateWorkPlaneArgs {
        kind: "plane-offset".into(),
        refs: Some(vec![key]),
        offset: "5 mm".into(),
        ..Default::default()
    };
    let no_owner = Client::new(oblikovati_automation::FileHost::new(
        &mut owner.files,
        &mut owner.guids,
        "test",
    ))
    .work_planes()
    .create(&args);
    assert!(
        no_owner
            .unwrap_err()
            .to_string()
            .contains("expected a part replay owner")
    );
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.snapshot().work_geometry.features().len(), 1);
    let mut replay = oblikovati_automation::PartReplayService::default();
    let result = Client::new(
        oblikovati_automation::FileHost::new(&mut owner.files, &mut owner.guids, "test")
            .with_part_update(&mut replay),
    )
    .work_planes()
    .create(&args)
    .unwrap();
    assert!(result.healthy);
    let rows = Client::new(
        oblikovati_automation::FileHost::new(&mut owner.files, &mut owner.guids, "test")
            .with_part_update(&mut replay),
    )
    .work_planes()
    .list()
    .unwrap()
    .planes
    .unwrap();
    assert!(rows[result.index as usize].healthy);
    owner
        .files
        .commands(document)
        .unwrap()
        .begin("Face plane batch")
        .unwrap();
    let transient = Client::new(
        oblikovati_automation::FileHost::new(&mut owner.files, &mut owner.guids, "test")
            .with_part_update(&mut replay),
    )
    .work_planes()
    .create(&args)
    .unwrap();
    assert!(transient.healthy);
    owner.files.commands(document).unwrap().abort().unwrap();
    let rows = Client::new(
        oblikovati_automation::FileHost::new(&mut owner.files, &mut owner.guids, "test")
            .with_part_update(&mut replay),
    )
    .work_planes()
    .list()
    .unwrap()
    .planes
    .unwrap();
    assert_eq!(rows.len(), 4);
}

#[path = "face_plane/sketch_journey.rs"]
mod sketch_journey;

#[path = "face_plane/direct_sketch.rs"]
mod direct_sketch;

#[test]
fn offset_plane_rejects_non_plane_with_matching_name() {
    use crate::HeadWork;
    use oblikovati_gui::GuiOffsetPlaneBase;
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let plane = Client::new(owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![oblikovati_api::types::WORK_REF_XY_PLANE.into()]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Duplicate label", |snapshot, _| {
            snapshot.work_geometry.rename(FeatureId(1), &plane.name)
        })
        .unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len();
    assert!(
        owner
            .create_offset_plane_from_gui(
                document,
                GuiOffsetPlaneBase::WorkPlane(FeatureId(1)),
                "5 mm"
            )
            .is_err()
    );
    let after = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len();
    assert_eq!(before, after);
}

#[test]
fn offset_plane_selects_feature_id_when_plane_labels_match() {
    use crate::HeadWork;
    use oblikovati_gui::GuiOffsetPlaneBase;
    use oblikovati_model::feature::{
        work_geometry::WorkGeometryDefinition, work_plane::WorkPlaneReference,
    };
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let request = CreateWorkPlaneArgs {
        kind: "plane-offset".into(),
        refs: Some(vec![oblikovati_api::types::WORK_REF_XY_PLANE.into()]),
        offset: "5 mm".into(),
        ..Default::default()
    };
    let first = Client::new(owner.document_host())
        .work_planes()
        .create(&request)
        .unwrap();
    Client::new(owner.document_host())
        .work_planes()
        .create(&request)
        .unwrap();
    owner
        .files
        .commands(document)
        .unwrap()
        .run("Duplicate labels", |snapshot, _| {
            snapshot.work_geometry.rename(FeatureId(3), &first.name)
        })
        .unwrap();
    owner
        .create_offset_plane_from_gui(
            document,
            GuiOffsetPlaneBase::WorkPlane(FeatureId(3)),
            "5 mm",
        )
        .unwrap();
    let session = owner.files.workspace().session(document).unwrap();
    let feature = session.snapshot().work_geometry.get(FeatureId(4)).unwrap();
    assert!(matches!(
        feature.definition(),
        WorkGeometryDefinition::Offset {
            base: WorkPlaneReference::Feature(FeatureId(3)),
            ..
        }
    ));
}
