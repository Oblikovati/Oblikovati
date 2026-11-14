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
fn competing_exact_face_and_body_picks_follow_the_user_priority() {
    use oblikovati_session::selection::SelectionKind;
    let (mut owner, _) = owner();
    let document = super::super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let ray = [[1., 1., 10.], [0., 0., -1.]];
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    assert!(session.selected_browser_body().is_some());
    let face = owner.ambient_selection.rank(SelectionKind::Face);
    owner.ambient_selection.move_kind(face, 0);
    owner
        .click_model_with_window(document, ray, Some(0.1))
        .unwrap();
    assert!(owner.model_face.is_some());
    owner.ambient_selection.enable_all();
    owner
        .ambient_selection
        .move_kind(owner.ambient_selection.rank(SelectionKind::Body), 0);
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
    assert_eq!(picked.1, oblikovati_martensite::PrismFaceRole::End);
}
