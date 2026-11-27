// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn ambiguous_part_pick_reports_the_failure_without_losing_the_prior_selection() {
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, true);
    owner.take_scene().unwrap();
    let feature = oblikovati_model::feature::FeatureId(1);
    owner.select_browser_feature(document, feature).unwrap();
    assert!(
        owner
            .click_model_with_window(document, [[1., 1., 10.], [0., 0., -1.]], Some(0.1))
            .unwrap()
    );
    assert!(
        matches!(owner.take_command_notice(), Some(crate::DesktopCommandNotice::SelectionFailed { cause }) if cause.contains("unambiguous"))
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .selected_browser_feature(),
        Some(feature)
    );
}

#[test]
fn exact_solid_hit_obeys_selection_presets_and_active_measure() {
    use oblikovati_session::selection::{SelectionKind, SelectionPriority};
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let ray = [[1., 1., 10.], [0., 0., -1.]];
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert!(owner.model_face.is_some());
    owner
        .ambient_selection
        .apply_priority(SelectionPriority::Part);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(session.selected_browser_body().is_some());
    assert!(owner.model_face.is_none());
    owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::StartMeasure)
        .unwrap();
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert!(
        owner.model_face.is_some(),
        "active measure must select the face"
    );
    owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::FinishMeasure)
        .unwrap();
    owner
        .ambient_selection
        .apply_priority(SelectionPriority::Face);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert!(owner.model_face.is_some());
    owner.ambient_selection.enable_all();
    owner
        .ambient_selection
        .set_enabled(SelectionKind::Face, false);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .selected_browser_body()
            .is_some()
    );
    owner.ambient_selection.disable_all();
    assert!(
        !owner
            .click_model_with_window(document, ray, Some(0.1))
            .unwrap()
    );
}

#[test]
fn viewport_ray_from_inside_a_solid_selects_the_forward_face() {
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let picked = owner
        .part_scenes
        .face_on_line(&owner.files, document, [[1., 1., 1.], [0., 0., 1.]])
        .unwrap()
        .unwrap();
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let expected = oblikovati_model::identity::SemanticReferenceName::new(
        guid,
        1,
        1,
        "sketch-prism",
        "end",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap();
    assert_eq!(picked.1, expected);
}

#[test]
fn viewport_toggle_preserves_other_selected_bodies_and_can_remove_the_pick() {
    use oblikovati_session::selection::{SelectionClickMode::Replace, SelectionPriority};
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    owner
        .ambient_selection
        .apply_priority(SelectionPriority::Part);
    let body = owner
        .gui_browser_snapshot(document)
        .unwrap()
        .bodies
        .remove(0);
    owner
        .files
        .commands(document)
        .unwrap()
        .apply_browser_body_click(
            "other-body",
            oblikovati_model::feature::FeatureId(99),
            Replace,
        );
    let ray = [[1., 1., 10.], [0., 0., -1.]];
    owner
        .click_model_with_modifiers(document, ray, Some(0.1), true)
        .unwrap();
    let selection = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .selected_browser_bodies();
    assert_eq!(
        selection
            .iter()
            .map(|item| item.key.as_str())
            .collect::<Vec<_>>(),
        ["other-body", body.key.as_str()]
    );
    owner
        .click_model_with_modifiers(document, ray, Some(0.1), true)
        .unwrap();
    let selection = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .selected_browser_bodies();
    assert_eq!(
        selection
            .iter()
            .map(|item| item.key.as_str())
            .collect::<Vec<_>>(),
        ["other-body"]
    );
}

#[test]
fn face_toggle_preserves_body_selection_and_toggles_the_face() {
    use oblikovati_session::selection::SelectionPriority;
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let ray = [[1., 1., 10.], [0., 0., -1.]];
    owner
        .ambient_selection
        .apply_priority(SelectionPriority::Part);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    let key = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .selected_browser_body()
        .unwrap()
        .0
        .to_owned();
    owner
        .ambient_selection
        .apply_priority(SelectionPriority::Face);
    owner
        .click_model_with_modifiers(document, ray, Some(0.1), true)
        .unwrap();
    assert!(owner.take_command_notice().is_none());
    assert_eq!(owner.current_face_appearance_picks(document).len(), 1);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .selected_browser_body()
            .unwrap()
            .0,
        key
    );
    owner
        .click_model_with_modifiers(document, ray, Some(0.1), true)
        .unwrap();
    assert!(owner.current_face_appearance_picks(document).is_empty());
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .selected_browser_body()
            .unwrap()
            .0,
        key
    );
}

fn plane_request(visible: bool) -> oblikovati_api::wire::CreateWorkPlaneArgs {
    oblikovati_api::wire::CreateWorkPlaneArgs {
        kind: "plane-offset".into(),
        refs: Some(vec![oblikovati_api::types::WORK_REF_XY_PLANE.into()]),
        offset: "3 cm".into(),
        visible: Some(Box::new(visible)),
        ..Default::default()
    }
}

fn add_plane(owner: &mut TestOwner, visible: bool) {
    oblikovati_api::client::Client::new(owner.document_host())
        .work_planes()
        .create(&plane_request(visible))
        .unwrap();
    owner.take_scene().unwrap();
}

fn chosen_feature(
    owner: &TestOwner,
    document: u64,
) -> Option<oblikovati_model::feature::FeatureId> {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .selected_browser_feature()
}

#[test]
fn near_visible_plane_and_face_follow_the_user_priority() {
    use oblikovati_session::selection::SelectionKind;
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    add_plane(&mut owner, true);
    let ray = [[1., 1., 10.], [0., 0., -1.]];
    owner
        .click_model_with_window(document, ray, Some(1.1))
        .unwrap();
    assert!(owner.model_face.is_some());
    let plane_rank = owner.ambient_selection.rank(SelectionKind::WorkPlane);
    owner.ambient_selection.move_kind(plane_rank, 0);
    owner
        .click_model_with_window(document, ray, Some(1.1))
        .unwrap();
    assert!(owner.model_face.is_none());
    assert_eq!(
        chosen_feature(&owner, document),
        Some(oblikovati_model::feature::FeatureId(2))
    );
}

#[test]
fn visible_plane_hits_respect_selection_filter() {
    use oblikovati_session::selection::SelectionKind;
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    add_plane(&mut owner, true);
    let plane = owner
        .gui_browser_snapshot(document)
        .unwrap()
        .features
        .into_iter()
        .find(|feature| feature.name == "Work Plane1")
        .unwrap()
        .id;
    let ray = [[4., 4., 10.], [0., 0., -1.]];
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert_eq!(chosen_feature(&owner, document), Some(plane));
    assert!(owner.gui_command_state().unwrap().look_at_available);
    assert!(
        crate::HeadWork::look_at_selection_plane(&owner)
            .unwrap()
            .is_some()
    );
    owner
        .ambient_selection
        .set_enabled(SelectionKind::WorkPlane, false);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert_eq!(chosen_feature(&owner, document), None);
    assert!(!owner.gui_command_state().unwrap().look_at_available);
}

#[test]
fn overlapping_planes_are_ambiguous_and_preserve_selection() {
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    add_plane(&mut owner, true);
    add_plane(&mut owner, true);
    let plane = owner
        .gui_browser_snapshot(document)
        .unwrap()
        .features
        .into_iter()
        .find(|feature| feature.name == "Work Plane1")
        .unwrap()
        .id;
    owner.select_browser_feature(document, plane).unwrap();
    owner
        .click_model_with_window(document, [[4., 4., 10.], [0., 0., -1.]], Some(0.1))
        .unwrap();
    assert!(
        matches!(owner.take_command_notice(), Some(crate::DesktopCommandNotice::SelectionFailed { cause }) if cause.contains("unambiguous"))
    );
    assert_eq!(chosen_feature(&owner, document), Some(plane));
}

#[test]
fn hidden_planes_are_not_pick_candidates() {
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    add_plane(&mut owner, false);
    owner
        .click_model_with_window(document, [[4., 4., 10.], [0., 0., -1.]], Some(0.1))
        .unwrap();
    assert_eq!(chosen_feature(&owner, document), None);
}
