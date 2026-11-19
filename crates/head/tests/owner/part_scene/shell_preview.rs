// SPDX-License-Identifier: GPL-2.0-only
//! The desktop preview must report an invalid Shell rather than display it (#4106, Martensite #554).
use super::*;
use oblikovati_api::wire::featureargs::Shell;

#[test]
fn rejected_shell_preview_keeps_the_document_and_scene_unchanged() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    let prepared = shell_preview_input(&owner, document);
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let error = owner
        .preview_selected_blend(document, |candidate, ids| prepared.create(candidate, ids))
        .unwrap_err();
    assert!(
        error.contains("tolerance"),
        "unexpected Shell refusal: {error}"
    );
    assert!(owner.extrusion_preview.is_none());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}

fn shell_preview_input(owner: &TestOwner, document: u64) -> oblikovati_automation::PreparedShell {
    let workspace = owner.files.workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let body = owner
        .part_scenes
        .current(session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap();
    let body_key = body
        .body_reference_key(session.history(), guid)
        .unwrap()
        .to_vec();
    let faces = body.face_reference_keys(session.history(), guid).unwrap();
    let input = Shell {
        thickness: "1 mm".into(),
        ..Default::default()
    };
    oblikovati_automation::prepare_shell(
        session.snapshot(),
        body_key,
        vec![faces[0].clone()],
        &input,
    )
    .unwrap()
}

#[test]
fn shell_desktop_request_validates_selection_and_reports_kernel_refusal() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap().unwrap();
    owner.ambient_selection.disable_all();
    owner
        .ambient_selection
        .set_enabled(oblikovati_session::selection::SelectionKind::Face, true);
    owner
        .click_model_with_window(document, [[2., 1.5, 10.], [0., 0., -1.]], Some(0.01))
        .unwrap();
    let face = owner.chosen_model_face(document).unwrap();
    let generation = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    let request = oblikovati_gui::GuiShellFeatureRequest {
        document,
        generation,
        faces: vec![face.key.iter().map(|byte| format!("{byte:02x}")).collect()],
        thickness: "1 mm".into(),
    };
    let mut stale = request.clone();
    stale.generation += 1;
    assert!(crate::HeadWork::preview_shell_from_gui(&mut owner, Some(&stale)).is_err());
    let error = crate::HeadWork::preview_shell_from_gui(&mut owner, Some(&request)).unwrap_err();
    assert!(
        error.contains("tolerance"),
        "unexpected Shell refusal: {error}"
    );
    assert!(owner.extrusion_preview.is_none());
    assert!(!owner.files.workspace().needs_save(document).unwrap());
}
