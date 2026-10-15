// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadGuiUpdate, HeadWork};
use oblikovati_gui::{GuiCommandState, GuiShellAction};
use oblikovati_viewport::{ViewportButton, ViewportInput};
use scene_tests::{capture_framebuffer, native_set, open_native_part, submit_native_scene};
struct ScriptedGuiWindow {
    native: NativeHeadWindow,
    scripted: Vec<ViewportEvent>,
    passed: Vec<ViewportEvent>,
}
fn scripted_window(native: NativeHeadWindow) -> ScriptedGuiWindow {
    ScriptedGuiWindow {
        native,
        scripted: Vec::new(),
        passed: Vec::new(),
    }
}
impl HeadWindow for ScriptedGuiWindow {
    fn gui_parameters_open(&self) -> bool {
        self.native.gui_parameters_open()
    }
    fn set_gui_parameter_snapshot(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.native.set_gui_parameter_snapshot(snapshot)
    }
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        self.native.show_gui_parameter_issue(issue)
    }
    fn acknowledge_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.native.acknowledge_gui_parameter(action, result)
    }

    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        self.native.gui_frame(event, commands)
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        events.append(&mut self.scripted);
        Ok(events)
    }
    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.passed.push(*event);
        self.native.process_view_input(event)
    }
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(update)
    }
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.native.show_command_notice(notice)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
fn click_shell(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    action: GuiShellAction,
) {
    let (tab, control) = match action {
        GuiShellAction::Save => ([25, 38], [100, 60]),
        GuiShellAction::Undo => ([25, 38], [145, 60]),
        GuiShellAction::Redo => ([25, 38], [190, 60]),
        GuiShellAction::Home => ([200, 38], [40, 60]),
        found => panic!("native shell click {found:?}: expected a history, save, or view action"),
    };
    click_shell_point(window, owner, tab);
    click_shell_point(window, owner, control);
}

fn click_shell_point(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    position: [i32; 2],
) {
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
fn native_gui_buttons_save_and_restore_model_pixels_without_camera_leakage() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("gui commands.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    let before = publish_gui_framebuffer(&mut owner, &mut window.native);
    native_set(&mut owner, "width", "6 cm");
    let edited = publish_gui_framebuffer(&mut owner, &mut window.native);
    assert_ne!(before.rgba, edited.rgba);
    click_shell(&mut window, &mut owner, GuiShellAction::Undo);
    assert_model_without_gui(&mut window.native, &before);
    click_shell(&mut window, &mut owner, GuiShellAction::Redo);
    assert_model_without_gui(&mut window.native, &edited);
    click_shell(&mut window, &mut owner, GuiShellAction::Save);
    verify_gui_save(&path, &mut window.native);
    verify_gui_camera(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn physical_keyboard_navigation_activates_native_save_without_camera_input() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("keyboard commands.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &path);
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    native_set(&mut owner, "width", "6 cm");
    click_shell(&mut window, &mut owner, GuiShellAction::Save);
    verify_keyboard_save_navigation(&mut window, &mut owner, &path);
    owner.finish().unwrap();
    window.finish().unwrap();
}
fn verify_keyboard_save_navigation(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    path: &std::path::Path,
) {
    let camera = window.native.view.camera_state().unwrap();
    for _ in 0..3 {
        press_shell_key(
            window,
            owner,
            oblikovati_viewport::ViewportPhysicalKey::LeftArrow,
        );
    }
    press_shell_key(
        window,
        owner,
        oblikovati_viewport::ViewportPhysicalKey::Space,
    );
    verify_gui_save(path, &mut window.native);
    assert_eq!(window.native.view.camera_state().unwrap(), camera);
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
    submit_native_scene(owner, window);
    capture_framebuffer(window)
}
fn assert_model_without_gui(
    window: &mut NativeHeadWindow,
    expected: &oblikovati_viewport::ViewportCapture,
) {
    let composed = window.view.capture().unwrap();
    assert_ne!(composed.rgba, expected.rgba);
    assert!(window.command_notice.is_some());
    window.clear_gui_frame().unwrap();
    assert!(window.view.capture().is_err());
    assert_eq!(capture_framebuffer(window).rgba, expected.rgba);
}
fn verify_gui_save(path: &std::path::Path, window: &mut NativeHeadWindow) {
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(path).unwrap())
            .unwrap();
    let restored = package.restore_document_parameters().unwrap();
    let width = restored.parameters.find("width").unwrap();
    assert_eq!(restored.parameters.get(width).unwrap().source, "6 cm");
    assert!(window.view.native_title().unwrap().contains("Saved"));
}
fn verify_gui_camera(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window
        .native
        .view
        .navigate(oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 })
        .unwrap();
    let initial = window.native.view.camera_state().unwrap();
    assert!(!drive_wheel(window, owner, [50, 50]));
    assert_eq!(window.native.view.camera_state().unwrap(), initial);
    assert!(drive_wheel(window, owner, [600, 400]));
    assert_ne!(window.native.view.camera_state().unwrap(), initial);
    click_shell(window, owner, GuiShellAction::Home);
    assert_eq!(window.native.view.camera_state().unwrap(), initial);
}
fn drive_wheel(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    position: [i32; 2],
) -> bool {
    let wheel = ViewportEvent::Input(ViewportInput::Wheel {
        notches: 2,
        position,
    });
    window.scripted = vec![wheel];
    window.passed.clear();
    crate::frames::run_frames(window, HeadOptions { frames: Some(2) }, owner).unwrap();
    window.passed.contains(&wheel)
}

#[test]
fn native_model_wheel_after_gui_click_and_gui_wheel_reaches_camera() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("wheel.opd"));
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    click_shell(&mut window, &mut owner, GuiShellAction::Home);
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

#[test]
fn native_parameter_panel_opens_with_owner_metadata_and_displays_local_results() {
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(&mut owner, &directory.path().join("native parameters.opd"));
    let mut window = scripted_window(native);
    let closed = publish_gui_framebuffer(&mut owner, &mut window.native);
    assert!(!window.gui_parameters_open());
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed,
                position: [60, 69],
                modifiers: Default::default(),
            })
        })
        .into();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(window.gui_parameters_open());
    let open = capture_framebuffer(&mut window.native);
    assert_ne!(closed.rgba, open.rgba);
    let snapshot = owner.gui_parameter_snapshot().unwrap().unwrap();
    let action = oblikovati_gui::GuiParameterAction::Toggle {
        document: snapshot.document,
        parameter: snapshot.rows[0].parameter,
        field: oblikovati_gui::GuiParameterToggleField::Key,
        value: true,
    };
    window
        .acknowledge_gui_parameter(
            &action,
            Err("Parameter edit failed: expected an editable row".into()),
        )
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let failed = capture_framebuffer(&mut window.native);
    assert_ne!(open.rgba, failed.rgba);
    window.acknowledge_gui_parameter(&action, Ok(())).unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let recovered = capture_framebuffer(&mut window.native);
    assert_ne!(failed.rgba, recovered.rgba);
    owner.finish().unwrap();
    window.finish().unwrap();
}
#[test]
fn native_parameter_view_failure_is_visible_and_recovers_without_closing_the_window() {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    let directory = tempfile::tempdir().unwrap();
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    open_native_part(
        &mut owner,
        &directory.path().join("parameter view recovery.opd"),
    );
    let mut window = scripted_window(native);
    publish_gui_framebuffer(&mut owner, &mut window.native);
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed,
                position: [60, 69],
                modifiers: Default::default(),
            })
        })
        .into();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let healthy = capture_framebuffer(&mut window.native);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "material".into(),
            expression: "bad\0text".into(),
            value_type: "text".into(),
            ..Default::default()
        })
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let failed = capture_framebuffer(&mut window.native);
    assert_ne!(healthy.rgba, failed.rgba);
    assert!(window.gui_parameters_open());
    let view = owner.gui_parameter_snapshot().unwrap().unwrap();
    let action = oblikovati_gui::GuiParameterAction::Text(oblikovati_gui::GuiParameterTextEdit {
        target: oblikovati_gui::GuiParameterTextTarget {
            document: view.document,
            parameter: view.rows.last().unwrap().parameter,
            field: oblikovati_gui::GuiParameterTextField::Equation,
        },
        request: 1,
        value: "steel".into(),
    });
    owner.apply_gui_parameter(&action).unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(1) }, &mut owner).unwrap();
    let recovered = capture_framebuffer(&mut window.native);
    assert_ne!(failed.rgba, recovered.rgba);
    assert!(window.gui_parameters_open());
    owner.finish().unwrap();
    window.finish().unwrap();
}
