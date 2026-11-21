// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn sketch_preferences_change_the_owned_application_options() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for label in ["Tools", "Application Options"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    window.gui_actions.clear();
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Show grid in sketch"),
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    assert!(
        window
            .gui_actions
            .iter()
            .all(|action| !matches!(action, oblikovati_gui::GuiShellAction::SetSketchOptions(_)))
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Apply"),
    );
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, oblikovati_gui::GuiShellAction::SetSketchOptions(_)))
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn escaping_sketch_preferences_discards_the_pending_edit() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for label in ["Tools", "Application Options", "Show grid in sketch"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    window.scripted = [true, false]
        .into_iter()
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                layout_key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                command: Some(oblikovati_viewport::ViewportKey::Cancel),
                modifiers: Default::default(),
                pressed,
            })
        })
        .collect();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for label in ["Application Options", "Show grid in sketch", "Apply"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .sketch_options
            .unwrap()
            .grid_visible
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn part_sketch_view_preference_is_staged_reverted_and_applied() {
    let preferences = tempfile::tempdir().unwrap();
    std::fs::write(
        preferences.path().join("options.yaml"),
        "part: {chamferFlatCorners: false, tangentChainSelect: true}\n",
    )
    .unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for label in ["Tools", "Application Options"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
    }
    let current = |owner: &mut NativeDesktopOwner| {
        owner
            .gui_command_state()
            .unwrap()
            .part_options
            .unwrap()
            .look_at_sketch_plane
    };
    assert!(current(&mut owner));
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .part_options
            .unwrap()
            .chamfer_flat_corners
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Look at sketch plane"),
    );
    assert!(current(&mut owner));
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Revert Part"),
    );
    assert!(current(&mut owner));
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Look at sketch plane"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Apply Part"),
    );
    assert!(!current(&mut owner));
    let reloaded =
        crate::native::preferences::load_native_application_options(preferences.path()).unwrap();
    assert!(
        !oblikovati_model::application_options::PartPreferenceAccess::part_options(&reloaded)
            .look_at_sketch_plane
    );
    assert!(
        !oblikovati_model::application_options::PartPreferenceAccess::part_options(&reloaded)
            .chamfer_flat_corners
    );
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn startup_plane_control_creates_the_requested_new_part_sketch() {
    use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
    for (label, plane) in [
        ("x-y", Some(WorkPlaneOrigin::XY)),
        ("y-z", Some(WorkPlaneOrigin::YZ)),
        ("x-z", Some(WorkPlaneOrigin::XZ)),
        ("No new sketch", None),
    ] {
        check_startup_plane_choice(label, plane);
    }
}
fn check_startup_plane_choice(
    label: &str,
    plane: Option<oblikovati_model::feature::work_plane::WorkPlaneOrigin>,
) {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for text in ["Tools", "Application Options"] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(text),
        );
    }
    if plane.is_some() {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center("No new sketch"),
        );
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center("Apply Part"),
        );
    }
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .part_options
            .unwrap()
            .startup_sketch_plane,
        plane
    );
    window.scripted = [true, false]
        .into_iter()
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                layout_key: oblikovati_viewport::ViewportPhysicalKey::Escape,
                command: Some(oblikovati_viewport::ViewportKey::Cancel),
                modifiers: Default::default(),
                pressed,
            })
        })
        .collect();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Get Started"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("New Part"),
    );
    let document = owner.documents().workspace().active_document().unwrap();
    let session = owner.documents().workspace().session(document).unwrap();
    let sketches = session.snapshot().sketches.sketches();
    assert_eq!(sketches.len(), usize::from(plane.is_some()));
    if let Some(plane) = plane {
        assert_eq!(
            sketches[0].placement(),
            oblikovati_model::document_sketch::SketchPlacement::OriginPlane(plane)
        );
        assert_eq!(owner.active_sketch_edit().unwrap().sketch, sketches[0].id());
        let camera = window.native.view.camera_state().unwrap();
        let (eye, up) = match plane {
            oblikovati_model::feature::work_plane::WorkPlaneOrigin::XY => {
                ([0., 0., 20.], [0., 1., 0.])
            }
            oblikovati_model::feature::work_plane::WorkPlaneOrigin::YZ => {
                ([20., 0., 0.], [0., 1., 0.])
            }
            oblikovati_model::feature::work_plane::WorkPlaneOrigin::XZ => {
                ([0., 20., 0.], [0., 0., 1.])
            }
        };
        assert_eq!((camera.eye, camera.target, camera.up), (eye, [0.; 3], up));
    } else {
        assert!(owner.active_sketch_edit().is_none());
    }
    owner.finish().unwrap();
    window.finish().unwrap();
}
