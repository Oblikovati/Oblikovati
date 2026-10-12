// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::run_window_with_owner;
use oblikovati_api::{client::Client, wire::*};
use oblikovati_viewport::{ViewportInput, ViewportKey};
struct SaveOnceWindow {
    native: NativeHeadWindow,
    saved_title: Option<String>,
}
impl HeadWindow for SaveOnceWindow {
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut inputs = self.native.events(limit)?;
        if self.saved_title.is_some() {
            return Ok(inputs);
        }
        inputs.push(ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Save,
            pressed: true,
        }));
        Ok(inputs)
    }
    fn submit_scene(&mut self, scene: crate::HeadSceneUpdate) -> Result<(), String> {
        self.native.submit_scene(scene)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.native.show_command_notice(notice)?;
        self.saved_title = Some(self.native.view.native_title()?);
        Ok(())
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
#[test]
fn native_frame_saves_an_edited_prism_and_displays_the_destination() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("saved prism.opd");
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_script_owner(native.wake.clone()).unwrap();
    let document = super::scene_tests::open_native_part(owner.document_owner_mut(), &path);
    edit_native_width(owner.document_owner_mut());
    let mut window = SaveOnceWindow {
        native,
        saved_title: None,
    };
    let summary =
        run_window_with_owner(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(summary.completed > 0);
    verify_owner_saved(
        &owner,
        document,
        &path,
        window.saved_title.as_deref().unwrap(),
    );
}
fn edit_native_width(owner: &mut NativeDesktopOwner) {
    Client::new(owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
}
fn verify_saved_width(path: &std::path::Path) {
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(path).unwrap())
            .unwrap();
    let restored = package.restore_document_parameters().unwrap();
    let width = restored.parameters.find("width").unwrap();
    assert_eq!(restored.parameters.get(width).unwrap().source, "6 cm");
}
#[test]
fn native_failure_feedback_survives_scene_publication_without_closing_the_view() {
    let mut window = NativeHeadWindow::open().unwrap();
    window
        .show_command_notice(crate::DesktopCommandNotice::SavePathRequired { document: 7 })
        .unwrap();
    let notice = window.view.native_title().unwrap();
    assert!(notice.contains("save path is required"));
    let empty = empty_save_scene();
    window.submit_scene(empty).unwrap();
    assert_eq!(window.view.native_title().unwrap(), notice);
    window.present().unwrap();
    window.finish().unwrap();
}

fn empty_save_scene() -> crate::HeadSceneUpdate {
    crate::HeadSceneUpdate {
        scene: ViewportScene {
            background: [0.; 3],
            triangles: Vec::new(),
        },
        status: crate::HeadSceneStatus {
            source: None,
            visible_bodies: 0,
            health: Vec::new(),
            presentation_declines: Vec::new(),
        },
    }
}

fn verify_owner_saved(
    owner: &NativeScriptDesktop,
    document: u64,
    path: &std::path::Path,
    title: &str,
) {
    let workspace = owner.document_owner().documents().workspace();
    assert!(!workspace.needs_save(document).unwrap());
    assert!(title.contains(path.to_str().unwrap()));
    verify_saved_width(path);
}
