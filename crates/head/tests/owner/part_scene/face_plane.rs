// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::CreateWorkPlaneArgs;

#[test]
#[ignore = "#3884: the face key and part-backed datum query are not wired yet"]
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
}
