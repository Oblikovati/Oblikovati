// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    json::EmptyObject,
    wire::{AssignAppearanceArgs, BodyInfo, BodyPhysicalPropertiesArgs},
};

pub(super) fn create_from_ribbon(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    document: u64,
) {
    preview_from_ribbon(window, owner, document);
    click_shell_position(
        window,
        owner,
        application_bdd::rendered_label_center("Cancel"),
    );
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::PreviewExtrude(None)))
    );
    assert!(
        body_list(owner).is_empty(),
        "cancelled preview must not create a body"
    );
    preview_from_ribbon(window, owner, document);
    let (position, size) = application_bdd::window_geometry("Extrude").unwrap();
    let close = application_bdd::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    click_shell_position(window, owner, close);
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::PreviewExtrude(None)))
    );
    assert!(
        body_list(owner).is_empty(),
        "closing a preview must not create a body"
    );
    preview_from_ribbon(window, owner, document);
    press_shell_key(
        window,
        owner,
        oblikovati_viewport::ViewportPhysicalKey::Enter,
    );
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::Extrude { .. })),
        "expected an extrusion action"
    );
    assert!(
        !matches!(
            window.native.command_notice,
            Some(crate::DesktopCommandNotice::AuthoringFailed { .. })
        ),
        "apply failed: {:?}",
        window.native.command_notice
    );
    assert_eq!(body_list(owner).len(), 1, "expected one extruded body");
    verify(owner, document, 1.);
    let displayed = capture_framebuffer(&mut window.native);
    assert!(
        scene_tests::solid_pixels(&displayed) > 0,
        "expected visible solid pixels in the native viewport"
    );
}

fn preview_from_ribbon(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    document: u64,
) {
    let before = capture_framebuffer(&mut window.native);
    window.gui_actions.clear();
    click_shell_position(
        window,
        owner,
        application_bdd::rendered_label_center("Extrude"),
    );
    assert!(
        window
            .gui_actions
            .contains(&oblikovati_gui::GuiShellAction::OpenExtrude),
        "expected Extrude ribbon action"
    );
    window.gui_actions.clear();
    let (position, size) = application_bdd::window_geometry("Extrude").unwrap();
    let distance = application_bdd::rendered_label_in_rect(
        "10.00mm",
        position,
        [position[0] + size[0], position[1] + size[1]],
    );
    click_shell_position(window, owner, distance);
    press_shell_key(window, owner, oblikovati_viewport::ViewportPhysicalKey::End);
    for _ in "10.00 mm".chars() {
        press_shell_key(
            window,
            owner,
            oblikovati_viewport::ViewportPhysicalKey::Backspace,
        );
    }
    type_native_text(window, owner, "size");
    let validated = owner
        .gui_command_state()
        .unwrap()
        .extrude_distance_validation
        .expect("expected owner validation after typing");
    assert_eq!(validated.distance, "size");
    assert!(
        validated.error.is_none(),
        "size must resolve as a positive length"
    );
    let after = capture_framebuffer(&mut window.native);
    assert!(
        window.gui_actions.iter().any(|action| matches!(
            action,
            oblikovati_gui::GuiShellAction::PreviewExtrude(Some(_))
        )),
        "expected preview action"
    );
    assert!(
        !matches!(
            window.native.command_notice,
            Some(crate::DesktopCommandNotice::AuthoringFailed { .. })
        ),
        "preview failed: {:?}",
        window.native.command_notice
    );
    assert_ne!(before.rgba, after.rgba);
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .work_geometry
            .features()
            .is_empty()
    );
}
pub(super) fn verify(owner: &mut NativeDesktopOwner, document: u64, side: f64) {
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    let bodies = body_list(owner);
    assert_eq!(
        bodies.len(),
        1,
        "document {document}, side {side}, browser {:?}, health {}",
        owner.gui_command_state().unwrap().browser,
        feature_health(owner, document)
    );
    let body = &bodies[0];
    assert!(body.solid);
    assert_eq!((body.faces, body.edges, body.vertices), (6, 12, 8));
    let report = Client::new(owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs::default())
        .unwrap();
    assert!((report.volume_mm3 - (side * 10.).powi(3)).abs() < 1e-6);
    assert!((report.surface_area_mm2 - 6. * (side * 10.).powi(2)).abs() < 1e-6);
}
pub(super) fn body_key(owner: &mut NativeDesktopOwner) -> String {
    body_list(owner).into_iter().next().unwrap().key
}
fn body_list(owner: &mut NativeDesktopOwner) -> Vec<BodyInfo> {
    Client::new(owner.document_host())
        .body()
        .list(&EmptyObject {})
        .unwrap()
        .bodies
        .unwrap_or_default()
}

fn feature_health(owner: &NativeDesktopOwner, document: u64) -> String {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        crate::part_scene::resolution_policy::desktop_policy(),
    );
    match session.snapshot().evaluate_part_features(&mut evaluator) {
        Ok(result) => format!("{:?}", result.health()),
        Err(cause) => cause.to_string(),
    }
}

pub(super) fn face_keys(owner: &NativeDesktopOwner, document: u64, side: f64) -> Vec<Vec<u8>> {
    let workspace = owner.documents().workspace();
    let session = workspace.session(document).unwrap();
    let guid = workspace.identity(document).unwrap().internal_name;
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        crate::part_scene::resolution_policy::desktop_policy(),
    );
    let evaluation = session
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .unwrap();
    let body = evaluation
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap();
    let keys: Vec<_> = body
        .reference_keys(session.history(), guid)
        .unwrap()
        .into_iter()
        .filter(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .unwrap()
                .kind
                == 1
        })
        .collect();
    assert_eq!(keys.len(), 6);
    for key in &keys {
        let face = body
            .as_prism()
            .unwrap()
            .face_report(session.history(), guid, key)
            .unwrap();
        assert_eq!(face.loops, 1);
        assert!((face.area - side.powi(2)).abs() < 1e-8);
    }
    keys
}

pub(super) fn topology_keys(owner: &mut NativeDesktopOwner) -> std::collections::BTreeSet<String> {
    let response = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let [body] = response.bodies.unwrap().try_into().unwrap();
    let body: oblikovati_api::wire::BodyTopology = body;
    let faces = body.faces.unwrap();
    for face in &faces {
        assert_eq!(face.kind, "plane", "cube face {}", face.key);
    }
    let groups = [faces, body.edges.unwrap(), body.vertices.unwrap()];
    assert_eq!(groups.each_ref().map(Vec::len), [6, 12, 8]);
    let keys: std::collections::BTreeSet<_> = groups
        .into_iter()
        .flatten()
        .map(|entity| entity.key)
        .collect();
    assert_eq!(keys.len(), 26);
    keys
}
pub(super) fn resolve_public_face(owner: &mut NativeDesktopOwner) {
    let response = Client::new(owner.document_host())
        .model()
        .reference_keys()
        .unwrap();
    let face = response.bodies.unwrap()[0].faces.as_ref().unwrap()[0].clone();
    assert_eq!(face.kind, "plane");
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "face".into(),
            key: face.key,
            appearance_id: "default".into(),
        })
        .unwrap();
}
