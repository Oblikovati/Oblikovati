// SPDX-License-Identifier: GPL-2.0-only
//! Named fillet replay through the real desktop scene owner (#4045).
use super::*;
use oblikovati_model::{
    feature::{fillet::FilletDefinition, part::PartFeatureDefinition},
    identity::{DocumentGuid, ReferenceEnvelope, SemanticReferenceName},
};

fn guid() -> DocumentGuid {
    "11111111-2222-3333-4444-555555555555".parse().unwrap()
}

#[test]
fn typed_api_creates_a_named_edge_fillet_and_rejects_unsupported_options() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    let edge_text: String = edge.iter().map(|byte| format!("{byte:02x}")).collect();
    let request = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: Some(vec![edge_text]),
        radius: "1 mm".into(),
        ..Default::default()
    };
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let bad_cross_section = oblikovati_api::wire::featureargs::Fillet {
        cross_section: "g2".into(),
        ..request.clone()
    };
    let bad_reference = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: Some(vec!["not-hex".into()]),
        ..request.clone()
    };
    let missing_reference = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: None,
        ..request.clone()
    };
    let bad_radius = oblikovati_api::wire::featureargs::Fillet {
        radius: "0 mm".into(),
        ..request.clone()
    };
    let foreign = SemanticReferenceName::new(
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
        1,
        2,
        "sketch-prism",
        "start-edge-curve",
        &[1, 101],
    )
    .unwrap()
    .encode()
    .unwrap();
    let foreign = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: Some(vec![
            foreign.iter().map(|byte| format!("{byte:02x}")).collect(),
        ]),
        ..request.clone()
    };
    for invalid in [
        bad_cross_section,
        bad_reference,
        missing_reference,
        bad_radius,
        foreign,
    ] {
        assert!(
            Client::new(owner.document_host())
                .features()
                .add_feature(&invalid)
                .is_err()
        );
    }
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        before
    );
    let result = Client::new(owner.document_host())
        .features()
        .add_feature(&request)
        .unwrap();
    assert_eq!(result.0.as_ref().unwrap()["kind"], "fillet");
    let mut replay = oblikovati_automation::PartReplayService::with_policy(
        crate::part_scene::resolution_policy::desktop_policy(),
    )
    .unwrap();
    assert!(
        oblikovati_automation::PartUpdateAccess::update_part(
            &mut replay,
            &mut owner.files,
            document
        )
        .unwrap()
        .is_empty()
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .features()
            .len(),
        1
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .features()
            .len(),
        2
    );
    let next_edge =
        SemanticReferenceName::new(guid, 1, 2, "sketch-prism", "start-edge-curve", &[1, 103])
            .unwrap()
            .encode()
            .unwrap();
    let next_request = oblikovati_api::wire::featureargs::Fillet {
        edge_refs: Some(vec![
            next_edge.iter().map(|byte| format!("{byte:02x}")).collect(),
        ]),
        ..request
    };
    let next = Client::new(owner.document_host())
        .features()
        .add_feature(&next_request)
        .unwrap();
    assert_eq!(next.0.as_ref().unwrap()["kind"], "fillet");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

pub(super) fn fillet_package() -> Vec<u8> {
    let mut snapshot = fixture::selected_prism_document();
    let body = SemanticReferenceName::new(guid(), 1, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    let edge =
        SemanticReferenceName::new(guid(), 1, 2, "sketch-prism", "start-edge-curve", &[1, 101])
            .unwrap()
            .encode()
            .unwrap();
    let definition = FilletDefinition::new(body, vec![edge], DatumLength::Constant(0.1)).unwrap();
    let reads = definition.reads();
    snapshot
        .work_geometry
        .append(AuthoredFeature::new(
            FeatureId(2),
            "Fillet",
            PartFeatureDefinition::Fillet(Box::new(definition)),
            reads,
        ))
        .unwrap();
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\nidentity: {internalName: 11111111-2222-3333-4444-555555555555}\n").unwrap()
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default()).unwrap()
        .with_sketches(&snapshot.sketches, 1, 104).unwrap()
        .with_work_geometry(&snapshot.work_geometry, 2).unwrap().encode().unwrap()
}

#[test]
fn selected_edge_fillet_preview_cancel_and_commit_are_isolated() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let path = std::env::temp_dir().join("head-fillet-authoring.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), fillet_package());
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    owner.take_scene().unwrap().unwrap();
    owner.ambient_selection.disable_all();
    owner
        .ambient_selection
        .set_enabled(oblikovati_session::selection::SelectionKind::Edge, true);
    owner
        .click_model_with_window(document, [[2., -1., 3.], [0., 1., -1.]], Some(0.01))
        .unwrap();
    assert_eq!(owner.chosen_model_edges(document).len(), 1);
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let mut request = oblikovati_gui::GuiFilletRequest {
        document,
        generation,
        edges: owner
            .chosen_model_edges(document)
            .iter()
            .map(|edge| edge.key.clone())
            .collect(),
        radius: "0.1 mm".into(),
    };
    let mut stale = request.clone();
    stale.generation += 1;
    assert!(crate::HeadWork::preview_fillet_from_gui(&mut owner, Some(&stale)).is_err());
    assert!(crate::HeadWork::commit_fillet_from_gui(&mut owner, &stale).is_err());
    stale.generation = generation;
    stale.edges[0].push(0);
    assert!(crate::HeadWork::commit_fillet_from_gui(&mut owner, &stale).is_err());
    let selected_outline = owner.current_selected_edge_outlines();
    assert!(!selected_outline.is_empty());
    assert!(crate::HeadWork::preview_fillet_from_gui(&mut owner, Some(&request)).unwrap());
    assert_eq!(owner.current_selected_edge_outlines(), selected_outline);
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(owner.extrusion_preview.is_some());
    assert!(crate::HeadWork::preview_fillet_from_gui(&mut owner, None).unwrap());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    request.radius = "0 mm".into();
    assert!(crate::HeadWork::preview_fillet_from_gui(&mut owner, Some(&request)).is_err());
    assert!(crate::HeadWork::commit_fillet_from_gui(&mut owner, &request).is_err());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    request.radius = "0.1 mm".into();
    assert!(crate::HeadWork::commit_fillet_from_gui(&mut owner, &request).unwrap());
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation()
            > generation
    );
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
}

#[test]
fn desktop_owner_displays_a_fillet_and_picks_its_named_face() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let path = std::env::temp_dir().join("head-fillet-scene.opd");
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), fillet_package());
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let update = owner.take_scene().unwrap().unwrap();
    assert_eq!(
        update.status.visible_bodies, 1,
        "health: {:?}",
        update.status.health
    );
    assert!(update.status.presentation_declines.is_empty());
    assert!(!update.scene.triangles.is_empty());
    let session = owner.files.workspace().session(document).unwrap();
    let body = owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(FeatureId(2))
        .unwrap();
    let expected_body = SemanticReferenceName::new(guid(), 1, 4, "sketch-prism", "body", &[1])
        .unwrap()
        .encode()
        .unwrap();
    assert_eq!(
        body.body_reference_key(session.history(), guid()).unwrap(),
        expected_body
    );
    let (feature, key) = owner
        .part_scenes
        .face_on_line(&owner.files, document, [[2., 1.5, 10.], [0., 0., -1.]])
        .unwrap()
        .unwrap();
    let (edge_feature, edge_key) = owner
        .part_scenes
        .edge_on_line(&owner.files, document, [[2., -1., 3.], [0., 1., -1.]], 0.01)
        .unwrap()
        .unwrap();
    assert_eq!(edge_feature, FeatureId(2));
    assert_eq!(ReferenceEnvelope::decode(&edge_key).unwrap().kind, 2);
    use oblikovati_session::selection::SelectionKind;
    owner.ambient_selection.disable_all();
    owner
        .ambient_selection
        .set_enabled(SelectionKind::Edge, true);
    owner
        .click_model_with_window(document, [[2., -1., 3.], [0., 1., -1.]], Some(0.01))
        .unwrap();
    assert_eq!(owner.chosen_model_edges(document).len(), 1);
    assert_eq!(owner.chosen_model_edges(document)[0].key, edge_key);
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
    assert!(!owner.current_browser_outline().is_empty());
    let groups = owner.current_selected_edge_outlines();
    assert_eq!(groups.len(), 1);
    assert_eq!(groups[0], owner.current_browser_outline());
    let key_before_fault = owner.model_edges[0].key.clone();
    owner.model_edges[0].key = vec![0];
    assert!(owner.current_selected_edge_outlines().is_empty());
    assert!(matches!(
        owner.take_command_notice(),
        Some(crate::DesktopCommandNotice::SelectionFailed { .. })
    ));
    owner.model_edges[0].key = key_before_fault;
    owner
        .click_model_with_modifiers(document, [[2., -1., 3.], [0., 1., -1.]], Some(0.01), true)
        .unwrap();
    assert!(owner.chosen_model_edges(document).is_empty());
    assert!(owner.current_selected_edge_outlines().is_empty());
    owner
        .click_model_with_modifiers(document, [[2., -1., 3.], [0., 1., -1.]], Some(0.01), true)
        .unwrap();
    assert_eq!(owner.chosen_model_edges(document)[0].key, edge_key);
    assert_eq!(owner.current_selected_edge_outlines(), groups);
    owner
        .click_model_with_window(document, [[20., 20., 10.], [0., 0., -1.]], Some(0.01))
        .unwrap();
    assert!(owner.chosen_model_edges(document).is_empty());
    assert!(owner.current_browser_outline().is_empty());
    owner.ambient_selection.enable_all();
    assert_eq!(feature, FeatureId(2));
    assert_eq!(ReferenceEnvelope::decode(&key).unwrap().kind, 1);
    use oblikovati_automation::{PartTopologyAccess, TopologyEntityKind};
    owner
        .part_scenes
        .verify(&mut owner.files, document, &key, TopologyEntityKind::Face)
        .unwrap();
    owner
        .ambient_selection
        .move_kind(owner.ambient_selection.rank(SelectionKind::Face), 0);
    owner
        .click_model_with_window(document, [[2., 1.5, 10.], [0., 0., -1.]], Some(0.1))
        .unwrap();
    let selected = owner.chosen_model_face(document).unwrap();
    assert_eq!(selected.feature, feature);
    assert_eq!(selected.key, key);
    let blend_ray = [[2., -1., -1.], [0., 1., 1.]];
    let (feature, blend) = owner
        .part_scenes
        .face_on_line(&owner.files, document, blend_ray)
        .unwrap()
        .unwrap();
    assert_eq!(feature, FeatureId(2));
    assert_eq!(ReferenceEnvelope::decode(&blend).unwrap().context, 2);
    owner
        .click_model_with_window(document, blend_ray, Some(0.1))
        .unwrap();
    assert_eq!(owner.chosen_model_face(document).unwrap().key, blend);
    set_parameter(&mut owner, "width", "6 cm");
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_eq!(
        owner
            .part_scenes
            .face_on_line(&owner.files, document, blend_ray)
            .unwrap()
            .unwrap()
            .1,
        blend
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_eq!(
        owner
            .part_scenes
            .face_on_line(&owner.files, document, blend_ray)
            .unwrap()
            .unwrap()
            .1,
        blend
    );
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        1
    );
    assert_eq!(
        owner
            .part_scenes
            .face_on_line(&owner.files, document, blend_ray)
            .unwrap()
            .unwrap()
            .1,
        blend
    );
}
