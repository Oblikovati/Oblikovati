// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn typed_distance_edit_accepts_literal_extrusions_without_editing_sketch_parameters() {
    use oblikovati_api::wire::{EditFeatureArgs, FeatureRefArgs, ScalarEdit};
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.distance = "10 mm".into();
    let response = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    let id = response.0.unwrap()["id"].as_u64().unwrap();
    let distance = {
        let session = owner.documents().workspace().session(document).unwrap();
        let snapshot = session.snapshot();
        let parameter = snapshot
            .parameters
            .find("d0")
            .expect("owned distance parameter");
        let detail = snapshot.parameters.get(parameter).unwrap();
        assert_eq!(detail.source, "10 mm");
        assert!(detail.built_in);
        let feature = snapshot
            .work_geometry
            .get(oblikovati_model::feature::FeatureId(id))
            .unwrap();
        assert_eq!(feature.owned_scalars()[0].role, "distance");
        assert_eq!(feature.owned_scalars()[0].parameter, parameter);
        parameter
    };
    let request = EditFeatureArgs {
        id,
        scalars: Some(vec![ScalarEdit {
            index: 0,
            value: "20 mm".into(),
        }]),
        ..Default::default()
    };
    Client::new(owner.document_host())
        .features()
        .edit(&request)
        .unwrap();
    let detail = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap();
    assert_eq!(detail.feature.scalars.unwrap()[0].value, 20.);
    let session = owner.documents().workspace().session(document).unwrap();
    let parameters = &session.snapshot().parameters;
    assert_eq!(parameters.find("d0"), Some(distance));
    assert_eq!(parameters.get(distance).unwrap().source, "20 mm");
    assert_eq!(
        parameters
            .model_value(parameters.find("width").unwrap())
            .unwrap()
            .value,
        1.
    );
    Client::new(owner.document_host())
        .features()
        .delete(&FeatureRefArgs { id })
        .unwrap();
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(snapshot.parameters.find("d0").is_none());
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(snapshot.parameters.find("d0"), Some(distance));
}

#[test]
fn distance_edit_preview_uses_private_candidate_and_cancel_preserves_history() {
    use oblikovati_api::wire::FeatureRefArgs;
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.distance = "10 mm".into();
    let response = Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    let id = response.0.unwrap()["id"].as_u64().unwrap();
    let original = owner.take_scene().unwrap().unwrap();
    owner
        .select_browser_feature(document, oblikovati_model::feature::FeatureId(id))
        .unwrap();
    assert!(!owner.current_browser_outline().is_empty());
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    assert!(
        owner
            .preview_edited_extrusion(document, id, "20 mm", None)
            .unwrap()
    );
    assert_eq!(
        owner.extrusion_preview.as_ref().unwrap().3,
        crate::owner::owner_extrusion_preview::PreviewKind::Replacement
    );
    assert!(owner.current_browser_outline().is_empty());
    let preview = owner.take_scene().unwrap().unwrap();
    assert_ne!(
        triangle_positions(&original.scene),
        triangle_positions(&preview.scene)
    );
    assert!(
        preview
            .scene
            .triangles
            .iter()
            .all(|triangle| triangle.color == [1., 0.65, 0.1, 1.])
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    let value = Client::new(owner.document_host())
        .features()
        .get(&FeatureRefArgs { id })
        .unwrap()
        .feature
        .scalars
        .unwrap()[0]
        .value;
    assert_eq!(value, 10.);
    assert!(owner.clear_extrusion_preview());
    assert!(!owner.current_browser_outline().is_empty());
    let restored = owner.take_scene().unwrap().unwrap();
    assert_eq!(
        triangle_positions(&original.scene),
        triangle_positions(&restored.scene)
    );
    assert!(
        owner
            .preview_edited_extrusion(document, id, "0 mm", None)
            .is_err()
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
}

fn triangle_positions(scene: &oblikovati_viewport::ViewportScene) -> Vec<[[f32; 3]; 3]> {
    scene
        .triangles
        .iter()
        .map(|triangle| triangle.positions)
        .collect()
}

#[test]
fn stale_extrusion_edit_is_rejected_without_changing_the_feature() {
    let (owner, _) = owner();
    let mut owner = owner.with_extrusion_resolution(resolution()).unwrap();
    let document = open_sketch(&mut owner);
    let mut input = request();
    input.distance = "10 mm".into();
    Client::new(owner.document_host())
        .features()
        .add_feature(&input)
        .unwrap();
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let request = oblikovati_gui::GuiExtrudeEditRequest {
        generation: Some(generation + 1),
        document,
        feature: oblikovati_model::feature::FeatureId(1),
        distance: "20 mm".into(),
        second_distance: None,
    };
    assert!(
        owner
            .commit_extrusion_edit(&request)
            .unwrap_err()
            .contains("generation")
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
}
