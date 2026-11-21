// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the picked-edge Fillet panel through native input (#4045).
use super::*;
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::ViewportPhysicalKey;

#[test]
fn native_fillet_can_pick_an_edge_after_the_tool_starts() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    let document = open_native_part(&mut owner, &directory.path().join("fillet-pick.opd"));
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 0);
    let before = feature_count(&owner, document);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Fillet"),
    );
    assert!(application_bdd::window_geometry("Fillet").is_some());
    assert!(!owner.gui_command_state().unwrap().extrude_preview_active);
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Enter);
    assert_eq!(feature_count(&owner, document), before);
    let pixel = window.native.view.model_pixel([2., 0., 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
    let selected = pixel_at(&capture_framebuffer(&mut window.native), pixel);
    assert!(
        selected[1] > selected[0] && selected[1] > selected[2],
        "selected edge pixel {selected:?}"
    );
    capture_when_requested(&mut window, "OBLIKOVATI_FILLET_PICK_CAPTURE");
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [1000, 600],
        modifiers: Default::default(),
    })];
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let selected = pixel_at(&capture_framebuffer(&mut window.native), pixel);
    assert!(
        selected[2] > selected[0] && selected[2] > selected[1],
        "retained edge pixel {selected:?}"
    );
    assert_eq!(feature_count(&owner, document), before);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("OK"),
    );
    assert_eq!(feature_count(&owner, document), before + 1);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_edge_fillet_preview_cancel_commit_and_history() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    let path = directory.path().join("fillet.opd");
    let document = open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let pixel = window.native.view.model_pixel([2., 0., 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().selected_edge_count,
        1,
        "pixel {pixel:?}; notice {:?}",
        window.native.command_notice
    );
    verify_edge_hover_and_persistent_selection(&mut window, &mut owner, pixel);
    verify_context_dismissal_clears_edge(&mut window, &mut owner, pixel);
    capture_when_requested(&mut window, "OBLIKOVATI_FILLET_BEFORE_CAPTURE");
    let face_pixel = window.native.view.model_pixel([2., 1.5, 2.]).unwrap();
    let original = capture_framebuffer(&mut window.native);
    let before = feature_count(&owner, document);
    open_fillet_menu(&mut window, &mut owner);
    assert!(
        window
            .gui_actions
            .iter()
            .any(|action| matches!(action, GuiShellAction::PreviewFillet(Some(_))))
    );
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
    let preview = capture_framebuffer(&mut window.native);
    assert_ne!(
        pixel_at(&original, face_pixel),
        pixel_at(&preview, face_pixel),
        "preview must be visible at the model face"
    );
    assert_ne!(
        pixel_at(&preview, face_pixel),
        pixel_at(&preview, [400, 200]),
        "the preview must not be hidden by its panel"
    );
    capture_when_requested(&mut window, "OBLIKOVATI_FILLET_TEST_CAPTURE");
    assert_eq!(feature_count(&owner, document), before);
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Escape);
    assert!(!owner.gui_command_state().unwrap().extrude_preview_active);
    assert_eq!(feature_count(&owner, document), before);
    open_fillet_toolbar(&mut window, &mut owner);
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Enter);
    assert_eq!(
        feature_count(&owner, document),
        before + 1,
        "notice {:?}",
        window.native.command_notice
    );
    for (label, expected) in [("Undo", before), ("Redo", before + 1)] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
        assert_eq!(feature_count(&owner, document), expected);
    }
    reopen_fillet(&mut window, &mut owner, &path, before + 1);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_fillet_radius_edit_previews_cancels_and_commits() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    let path = directory.path().join("fillet-edit.opd");
    let document = open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let pixel = window.native.view.model_pixel([2., 0., 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    open_fillet_toolbar(&mut window, &mut owner);
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Enter);
    assert_eq!(fillet_radius(&owner, document), 0.2);
    open_fillet_edit(&mut window, &mut owner);
    change_fillet_radius(&mut window, &mut owner, "1 mm");
    assert!(owner.gui_command_state().unwrap().extrude_preview_active);
    assert_eq!(window.native.command_notice, None);
    capture_when_requested(&mut window, "OBLIKOVATI_FILLET_EDIT_CAPTURE");
    assert_eq!(fillet_radius(&owner, document), 0.2);
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Escape);
    assert!(!owner.gui_command_state().unwrap().extrude_preview_active);
    assert_eq!(fillet_radius(&owner, document), 0.2);
    open_fillet_edit(&mut window, &mut owner);
    change_fillet_radius(&mut window, &mut owner, "1 mm");
    press_shell_key(&mut window, &mut owner, ViewportPhysicalKey::Enter);
    assert_eq!(fillet_radius(&owner, document), 0.1);
    for (label, expected) in [("Undo", 0.2), ("Redo", 0.1)] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
        assert_eq!(fillet_radius(&owner, document), expected);
    }
    reopen_fillet(&mut window, &mut owner, &path, 2);
    let reopened = owner.documents().workspace().active_document().unwrap();
    assert_eq!(fillet_radius(&owner, reopened), 0.1);
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn open_fillet_edit(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    click_shell_position(
        window,
        owner,
        application_bdd::rendered_label_center("Fillet1"),
    );
    let state = owner.gui_command_state().unwrap();
    assert_eq!(
        state.selected_edge_count, 0,
        "browser choice must clear edge selection"
    );
    assert!(
        !state.model_face_chosen,
        "browser choice must clear face selection"
    );
    assert_eq!(
        state.browser.unwrap().selected_feature,
        Some(oblikovati_model::feature::FeatureId(2))
    );
    assert!(
        !owner.selected_browser_outline().is_empty(),
        "selected Fillet has no outline"
    );
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [1000, 700],
        modifiers: Default::default(),
    })];
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let (position, size) = application_bdd::window_geometry("##selection-toolbar").unwrap();
    let end = [position[0] + size[0], position[1] + size[1]];
    let edit = application_bdd::rendered_label_in_rect("EditFeature", position, end);
    click_shell_position(window, owner, edit);
    assert!(
        application_bdd::window_geometry("Fillet").is_some(),
        "edit toolbar did not open panel"
    );
}

fn change_fillet_radius(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    value: &str,
) {
    let (position, size) = application_bdd::window_geometry("Fillet").unwrap();
    let end = [position[0] + size[0], position[1] + size[1]];
    let label = application_bdd::rendered_label_in_rect("Radius", position, end);
    let field_x = application_bdd::pointer_pixels([position[0] + 80., position[1]])[0];
    click_shell_position(window, owner, [field_x, label[1]]);
    for _ in 0..12 {
        press_shell_key(window, owner, ViewportPhysicalKey::Backspace);
    }
    type_native_text(window, owner, value);
}

fn fillet_radius(owner: &NativeDesktopOwner, document: u64) -> f64 {
    let session = owner.documents().workspace().session(document).unwrap();
    let feature = session
        .snapshot()
        .work_geometry
        .features()
        .iter()
        .find(|feature| {
            matches!(
                feature.definition(),
                oblikovati_model::feature::part::PartFeatureDefinition::Fillet(_)
            )
        })
        .unwrap();
    let oblikovati_model::feature::part::PartFeatureDefinition::Fillet(definition) =
        feature.definition()
    else {
        panic!("expected Fillet")
    };
    definition
        .resolved_radius(&session.snapshot().parameters)
        .unwrap()
}
fn verify_context_dismissal_clears_edge(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edge: [i32; 2],
) {
    open_part_context_menu(window, owner);
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
    window.scripted = pointer_click_events([500, 750], 1);
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert_eq!(
        owner.gui_command_state().unwrap().selected_edge_count,
        0,
        "empty viewport click must clear the edge"
    );
    window.scripted = pointer_click_events(edge, 1);
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
}
fn verify_edge_hover_and_persistent_selection(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    pixel: [i32; 2],
) {
    let hovered = capture_framebuffer(&mut window.native);
    let green = pixel_at(&hovered, pixel);
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [1000, 600],
        modifiers: Default::default(),
    })];
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    let selected = capture_framebuffer(&mut window.native);
    let blue = pixel_at(&selected, pixel);
    assert!(
        green[1] > green[0] && green[1] > green[2],
        "hover pixel {green:?}"
    );
    assert!(
        blue[2] > blue[0] && blue[2] > blue[1],
        "selected pixel {blue:?}"
    );
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
}
fn open_fillet_menu(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    open_part_context_menu(window, owner);
    assert_eq!(owner.gui_command_state().unwrap().selected_edge_count, 1);
    assert!(!owner.gui_command_state().unwrap().look_at_available);
    application_bdd::rendered_label_in_rect("LookAt", [900., 500.], [1440., 900.]);
    application_bdd::rendered_label_in_rect("HomeView", [900., 500.], [1440., 900.]);
    let button = application_bdd::rendered_label_in_rect("Fillet", [1100., 500.], [1440., 900.]);
    window.gui_actions.clear();
    click_shell_position(window, owner, button);
}
fn open_part_context_menu(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let position = [1000, 600];
    let motion = ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers: Default::default(),
    });
    let clicks = [true, false].map(|pressed| {
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Right,
            pressed,
            position,
            modifiers: Default::default(),
        })
    });
    for event in std::iter::once(motion).chain(clicks) {
        window.scripted = vec![event];
        crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    }
}

#[test]
fn native_look_at_orients_to_a_selected_planar_face() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner_at(native.wake.clone(), directory.path()).unwrap();
    open_native_part(&mut owner, &directory.path().join("look-at.opd"));
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let filter = owner.gui_command_state().unwrap().selection_filter.unwrap();
    let rank = filter
        .entries
        .iter()
        .position(|entry| entry.kind == oblikovati_gui::GuiSelectionKind::Face)
        .unwrap();
    owner.move_selection_kind_from_gui(rank, 0).unwrap();
    let pixel = window.native.view.model_pixel([2., 1.5, 2.]).unwrap();
    window.scripted = pointer_click_events(pixel, 1);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(owner.gui_command_state().unwrap().look_at_available);
    let plane = owner.look_at_selection_plane().unwrap().unwrap();
    let camera = window.native.view.camera_state().unwrap().isometric();
    window.native.set_view_camera(camera).unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    open_part_context_menu(&mut window, &mut owner);
    let button = application_bdd::rendered_label_in_rect("LookAt", [900., 500.], [1440., 900.]);
    click_shell_position(&mut window, &mut owner, button);
    let facing = window.native.view.camera_state().unwrap();
    assert_ne!(camera.eye, facing.eye);
    assert_eq!(facing.target, plane.recipe().origin);
    let delta: [f64; 3] = std::array::from_fn(|axis| facing.eye[axis] - facing.target[axis]);
    let normal = plane.normal();
    let cross = [
        delta[1] * normal[2] - delta[2] * normal[1],
        delta[2] * normal[0] - delta[0] * normal[2],
        delta[0] * normal[1] - delta[1] * normal[0],
    ];
    assert!(cross.iter().all(|value| value.abs() < 1e-8));
    assert!(window.previous_view().unwrap());
    assert_eq!(window.native.view.camera_state().unwrap().eye, camera.eye);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn pixel_at(capture: &oblikovati_viewport::ViewportCapture, pixel: [i32; 2]) -> [u8; 4] {
    let [x, y] = pixel.map(|coordinate| usize::try_from(coordinate).unwrap());
    let offset = (y * capture.extent[0] as usize + x) * 4;
    capture.rgba[offset..offset + 4].try_into().unwrap()
}
fn reopen_fillet(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    path: &std::path::Path,
    expected: usize,
) {
    click_shell_position(
        window,
        owner,
        application_bdd::rendered_label_center("Save"),
    );
    for label in ["Close", "Open..."] {
        click_shell_position(
            window,
            owner,
            application_bdd::rendered_label_center("File"),
        );
        click_shell_position(window, owner, application_bdd::rendered_label_center(label));
    }
    type_native_text(window, owner, path.to_str().unwrap());
    press_shell_key(window, owner, ViewportPhysicalKey::Enter);
    let reopened = owner.documents().workspace().active_document().unwrap();
    assert_eq!(feature_count(owner, reopened), expected);
    assert_eq!(window.native.scene_status().unwrap().visible_bodies, 1);
}
fn capture_when_requested(window: &mut ScriptedGuiWindow, variable: &str) {
    let Ok(path) = std::env::var(variable) else {
        return;
    };
    use crate::WindowCaptureFiles;
    let capture = capture_framebuffer(&mut window.native);
    crate::NativeWindowCaptureFiles
        .write_capture(std::path::Path::new(&path), &capture)
        .unwrap();
}
fn feature_count(owner: &NativeDesktopOwner, document: u64) -> usize {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len()
}
fn open_fillet_toolbar(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    let (position, size) =
        application_bdd::window_geometry("##selection-toolbar").expect("selected edge toolbar");
    let end = [position[0] + size[0], position[1] + size[1]];
    let button = application_bdd::rendered_label_in_rect("Fillet", position, end);
    window.gui_actions.clear();
    click_shell_position(window, owner, button);
}
