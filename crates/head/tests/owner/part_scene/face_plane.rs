// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::CreateWorkPlaneArgs;

#[test]
#[ignore = "#3884: face-key creation is not wired yet"]
fn face_offset_command_binds_a_current_face_and_reports_its_geometry() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    open_prism(&mut owner, false);
    let faces = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()[0]
        .faces
        .clone()
        .unwrap();
    let face = faces.iter().find(|face| face.kind == "plane").unwrap();
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
