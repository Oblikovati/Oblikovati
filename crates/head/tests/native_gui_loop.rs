// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadSketchWork;
use crate::{HeadGuiUpdate, HeadWork};
use oblikovati_gui::GuiCommandState;
use oblikovati_viewport::{ViewportButton, ViewportInput};
use scene_tests::{capture_framebuffer, native_set, open_native_part, submit_native_scene};
use std::time::Duration;
#[path = "native_gui_loop/camera.rs"]
mod camera;
#[path = "native_gui_loop/create.rs"]
mod create;
#[path = "native_gui_loop/cube.rs"]
mod cube;
use camera::{drive_wheel, press_camera_home, verify_gui_camera};
#[path = "native_gui_loop/files.rs"]
mod files;
#[path = "native_gui_loop/model_state_journey.rs"]
mod model_state_journey;
#[path = "native_gui_loop/parameters.rs"]
mod parameters;
struct ScriptedGuiWindow {
    native: NativeHeadWindow,
    scripted: Vec<ViewportEvent>,
    passed: Vec<ViewportEvent>,
    gui_actions: Vec<oblikovati_gui::GuiShellAction>,
    scene_colors: Vec<[f32; 4]>,
    scene_triangles: Vec<oblikovati_viewport::ViewportTriangle>,
    presentation_delay: Duration,
    template_material_override: Option<oblikovati_gui::GuiTemplateMaterialEntry>,
    template_size_issue_override: Option<String>,
}
fn scripted_window(native: NativeHeadWindow) -> ScriptedGuiWindow {
    ScriptedGuiWindow {
        native,
        scripted: Vec::new(),
        passed: Vec::new(),
        gui_actions: Vec::new(),
        scene_colors: Vec::new(),
        scene_triangles: Vec::new(),
        presentation_delay: Duration::ZERO,
        template_material_override: None,
        template_size_issue_override: None,
    }
}
const STATUS_LINE_ROWS: usize = 22;
#[path = "native_gui_loop/scripted_window.rs"]
mod scripted_window;
fn click_shell_position(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    position: [i32; 2],
) {
    drive_pointer_clicks(window, owner, position, 1);
}
fn pointer_click_events(position: [i32; 2], count: usize) -> Vec<ViewportEvent> {
    let motion = ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers: Default::default(),
    });
    let clicks = (0..count).flat_map(|_| [true, false]).map(|pressed| {
        ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed,
            position,
            modifiers: Default::default(),
        })
    });
    std::iter::once(motion).chain(clicks).collect()
}
fn drive_pointer_clicks(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    position: [i32; 2],
    count: usize,
) {
    window.scripted = pointer_click_events(position, count);
    window.passed.clear();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert!(
        !window
            .passed
            .iter()
            .any(|event| matches!(event, ViewportEvent::Input(ViewportInput::Button { .. })))
    );
}
#[test]
fn native_new_part_button_creates_and_selects_an_unsaved_document() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path())
        .unwrap()
        .with_themes(oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Dark").unwrap());
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    create::metric_part(&mut window, &mut owner);
    assert_eq!(owner.documents().workspace().documents().len(), 1);
    assert!(owner.documents().workspace().active_document().is_some());
    assert!(!owner.close_allowed().unwrap());
    let variant = model_state_journey::prepare(&mut owner);
    parameters::create_size_parameter(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Start 2D Sketch"),
    );
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("XY Plane"),
    );
    assert!(owner.active_sketch_edit().is_some());
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Rectangle"),
    );
    assert!(owner.placement_active());
    let edit = owner.active_sketch_edit().unwrap();
    {
        let position = window
            .native
            .view
            .sketch_pixel(edit.plane, [0., 0.])
            .unwrap();
        window.scripted = [true, false]
            .map(|pressed| {
                ViewportEvent::Input(ViewportInput::Button {
                    button: ViewportButton::Left,
                    pressed,
                    position,
                    modifiers: Default::default(),
                })
            })
            .into();
        crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner)
            .unwrap();
    }
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [1000, 600],
        modifiers: Default::default(),
    })];
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for key in [
        oblikovati_viewport::ViewportPhysicalKey::Tab,
        oblikovati_viewport::ViewportPhysicalKey::Enter,
    ] {
        type_native_text(&mut window, &mut owner, "size");
        press_shell_key(&mut window, &mut owner, key);
    }
    parameters::verify_rectangle(&owner, edit, 1.);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(edit.document)
            .unwrap()
            .snapshot()
            .sketches
            .get(edit.sketch)
            .unwrap()
            .geometry()
            .lines()
            .len(),
        4
    );
    for (label, expected_lines) in [("Undo", 0), ("Redo", 4)] {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_center(label),
        );
        assert_eq!(
            owner
                .documents()
                .workspace()
                .session(edit.document)
                .unwrap()
                .snapshot()
                .sketches
                .get(edit.sketch)
                .unwrap()
                .geometry()
                .lines()
                .len(),
            expected_lines
        );
    }
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Finish Sketch"),
    );
    assert!(owner.active_sketch_edit().is_none());
    assert!(!owner.placement_active());
    verify_browser_reentry_pixels(&mut window, &mut owner, edit);
    cube::create_from_ribbon(&mut window, &mut owner, edit.document);
    model_state_journey::check_switches(&mut window, &mut owner, edit.document, variant);
    parameters::edit_size_and_history(&mut window, &mut owner, edit);
    cube::verify(&mut owner, edit.document, 2.);
    let key = cube::body_key(&mut owner);
    let face_keys = cube::face_keys(&owner, edit.document, 2.);
    let topology_keys = cube::topology_keys(&mut owner);
    let _saved_directory = files::save_rectangle_file(&mut window, &mut owner, edit);
    let reopened = owner.documents().workspace().active_document().unwrap();
    cube::verify(&mut owner, reopened, 2.);
    assert_eq!(cube::body_key(&mut owner), key);
    assert_eq!(cube::face_keys(&owner, reopened, 2.), face_keys);
    assert_eq!(cube::topology_keys(&mut owner), topology_keys);
    cube::resolve_public_face(&mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_browser_reentry_pixels(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    edit: crate::HeadSketchEdit,
) {
    move_pointer_away(window, owner);
    let before = capture_framebuffer(&mut window.native);
    assert!(
        before
            .rgba
            .as_chunks::<4>()
            .0
            .contains(&[160, 160, 160, 255]),
        "expected visible committed sketch geometry before browser activation"
    );
    // #3811: selection changes colour; isolate edit re-entry from that intended change.
    drive_pointer_clicks(
        window,
        owner,
        application_bdd::rendered_sketch_row_center(),
        1,
    );
    assert!(owner.active_sketch_edit().is_none());
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .browser
            .unwrap()
            .selected_sketches,
        [edit.sketch]
    );
    move_pointer_away(window, owner);
    let before = capture_framebuffer(&mut window.native);
    let original = window.native.view.camera_state().unwrap();
    browser::double_click_browser_row(window, owner);
    assert_eq!(owner.active_sketch_edit().unwrap(), edit);
    assert_eq!(window.native.view.camera_state().unwrap(), original);
    move_pointer_away(window, owner);
    let after = capture_framebuffer(&mut window.native);
    let width = before.extent[0] as usize;
    let (tabs, size) = application_bdd::window_geometry("##open-document-tabs").unwrap();
    let content_top = application_bdd::pointer_pixels([0., tabs[1] + size[1] + 1.])[1] as usize;
    assert!(width >= 1200 && content_top + STATUS_LINE_ROWS < before.extent[1] as usize);
    // #3798: contextual chrome changes on entry; the camera and visible model must not.
    for (row, (old, new)) in before
        .rgba
        .chunks_exact(width * 4)
        .zip(after.rgba.chunks_exact(width * 4))
        .enumerate()
        .skip(content_top)
        // The status line at the foot reports the edit, as it should.
        .take((before.extent[1] as usize).saturating_sub(content_top + STATUS_LINE_ROWS))
    {
        assert!(
            // The view controls overlap the right side of the viewport in the part workspace.
            old[500 * 4..1200 * 4] == new[500 * 4..1200 * 4],
            "re-entering the same XY camera must preserve rectangle pixels on row {row}, first difference at {:?}",
            (500..1200).find(|x| old[x * 4..x * 4 + 4] != new[x * 4..x * 4 + 4])
        );
    }
    let finish = application_bdd::rendered_label_center("Finish Sketch");
    click_shell_position(window, owner, finish);
    assert!(owner.active_sketch_edit().is_none());
}
fn move_pointer_away(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window.scripted = vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [1100, 800],
        modifiers: Default::default(),
    })];
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
}
fn type_native_text(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    expression: &str,
) {
    window.scripted = expression
        .chars()
        .map(|character| ViewportEvent::Input(ViewportInput::Text(character)))
        .collect();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
}
fn press_shell_key(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    key: oblikovati_viewport::ViewportPhysicalKey,
) {
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key,
                layout_key: key,
                command: None,
                pressed,
                modifiers: Default::default(),
            })
        })
        .into();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
}
fn publish_gui_framebuffer(
    owner: &mut NativeDesktopOwner,
    window: &mut NativeHeadWindow,
) -> oblikovati_viewport::ViewportCapture {
    scene_tests::apply_application_appearance(owner, window);
    submit_native_scene(owner, window);
    capture_framebuffer(window)
}
#[path = "native_gui_loop/history_navigation.rs"]
mod history_navigation;
#[test]
fn native_model_wheel_after_gui_click_and_gui_wheel_reaches_camera() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("wheel.opd"));
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    press_camera_home(&mut window, &mut owner);
    camera::assert_model_point_in_document(&mut window);
    let before = window.native.view.camera_state().unwrap();
    assert!(!drive_wheel(&mut window, &mut owner, [50, 50]));
    assert_eq!(window.native.view.camera_state().unwrap(), before);
    assert!(drive_wheel(&mut window, &mut owner, [600, 400]));
    assert_ne!(window.native.view.camera_state().unwrap(), before);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_gui_resize_after_attachment_requests_and_completes_a_fresh_frame() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.gui_frame(None, &GuiCommandState::default()).unwrap();
    capture_framebuffer(&mut window);
    for extent in [[800, 600], [1440, 900]] {
        verify_gui_resize_recovery(&mut window, extent);
    }
    window.view.resize([800, 600]).unwrap();
    verify_gui_resize_recovery(&mut window, [1440, 900]);
    window.finish().unwrap();
}

fn verify_gui_resize_recovery(window: &mut NativeHeadWindow, extent: [u32; 2]) {
    window.view.resize(extent).unwrap();
    assert!(window.view.capture().is_err());
    assert_eq!(window.present().unwrap(), HeadPresentation::SurfaceChanged);
    let capture = complete_resized_gui_frame(window);
    assert_gui_framebuffer_visible(&capture);
}

fn complete_resized_gui_frame(
    window: &mut NativeHeadWindow,
) -> oblikovati_viewport::ViewportCapture {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(3);
    loop {
        assert!(
            std::time::Instant::now() < deadline,
            "GUI resize: expected a completed current surface"
        );
        if let Some(capture) = current_resized_gui_capture(window) {
            return capture;
        }
    }
}

fn current_resized_gui_capture(
    window: &mut NativeHeadWindow,
) -> Option<oblikovati_viewport::ViewportCapture> {
    window.events(64).unwrap();
    window.gui_frame(None, &GuiCommandState::default()).unwrap();
    let prepared = window.drawable_extent().unwrap();
    if window.present().unwrap() == HeadPresentation::SurfaceChanged {
        assert!(window.view.capture().is_err());
        return None;
    }
    let actual = window.drawable_extent().unwrap();
    if actual != prepared {
        assert!(window.view.capture().is_err());
        return None;
    }
    let capture = window.view.capture().unwrap();
    assert_eq!(capture.extent, actual);
    Some(capture)
}

fn assert_gui_framebuffer_visible(capture: &oblikovati_viewport::ViewportCapture) {
    let visible = capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| **pixel != [0, 0, 0, 255])
        .count();
    assert!(
        visible > 1000,
        "expected the fresh GUI in the completed framebuffer"
    );
}

#[path = "native_gui_loop/browser.rs"]
mod browser;

#[allow(unsafe_code)]
#[path = "application/benchmark.rs"]
mod application_bdd;

#[path = "native_gui_loop/pattern.rs"]
mod pattern;

#[path = "native_gui_loop/document_ribbons.rs"]
mod document_ribbons;
#[path = "native_gui_loop/legacy_parameters.rs"]
mod legacy_parameters;

#[path = "native_gui_loop/sketch_preferences.rs"]
mod sketch_preferences;

#[path = "native_gui_loop/selection_preferences.rs"]
mod selection_preferences;
