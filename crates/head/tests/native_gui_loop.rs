// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadGuiUpdate, HeadWork};
use oblikovati_gui::GuiCommandState;
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
fn click_shell(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner, x: i32) {
    window.scripted = [true, false]
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed,
                position: [x, 45],
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
    click_shell(&mut window, &mut owner, 75);
    assert_model_without_gui(&mut window.native, &before);
    click_shell(&mut window, &mut owner, 120);
    assert_model_without_gui(&mut window.native, &edited);
    click_shell(&mut window, &mut owner, 30);
    verify_gui_save(&path, &mut window.native);
    verify_gui_camera(&mut window, &mut owner);
    owner.finish().unwrap();
    window.finish().unwrap();
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
    click_shell(window, owner, 165);
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
    click_shell(&mut window, &mut owner, 165);
    let before = window.native.view.camera_state().unwrap();
    assert!(!drive_wheel(&mut window, &mut owner, [50, 50]));
    assert_eq!(window.native.view.camera_state().unwrap(), before);
    assert!(drive_wheel(&mut window, &mut owner, [600, 400]));
    assert_ne!(window.native.view.camera_state().unwrap(), before);
    owner.finish().unwrap();
    window.finish().unwrap();
}
