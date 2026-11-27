// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{client::Client, wire::SaveDocumentAsArgs};

#[test]
fn native_close_without_documents_exits_without_confirmation() {
    let (mut window, mut owner) = initialized_window();
    assert!(owner.documents().workspace().documents().is_empty());
    close_without_confirmation(&mut window, &mut owner);
}

#[test]
fn native_close_with_saved_clean_document_exits_without_confirmation() {
    let directory = tempfile::tempdir().unwrap();
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    let path = directory.path().join("clean.opd");
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.to_str().unwrap().to_owned(),
        })
        .unwrap();
    assert!(path.is_file());
    assert!(owner.close_allowed().unwrap());
    close_without_confirmation(&mut window, &mut owner);
}

fn initialized_window() -> (ScriptedGuiWindow, NativeDesktopOwner) {
    let native = NativeHeadWindow::open().unwrap();
    let mut owner = native_owner(native.wake.clone()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    (window, owner)
}

fn close_without_confirmation(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window.scripted.push(ViewportEvent::CloseRequested);
    let summary = crate::run_window_with_owner(window, HeadOptions::default(), owner).unwrap();
    assert_eq!(summary.completed, 0);
    assert_eq!(window.confirmation_requests, 0);
}

#[test]
fn native_dirty_close_cancel_keeps_the_document_open() {
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    parameters::create_size_parameter(&mut window, &mut owner);
    let document = owner.documents().workspace().active_document().unwrap();
    assert!(!owner.close_allowed().unwrap());
    window.scripted.push(ViewportEvent::CloseRequested);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert_eq!(window.confirmation_requests, 1);
    assert!(application_bdd::window_geometry("Save changes").is_some());
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Cancel"),
    );
    assert!(application_bdd::window_geometry("Save changes").is_none());
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    assert!(!owner.close_allowed().unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_exit_discard_keeps_saved_file_unchanged() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("discard.opd");
    let (mut window, mut owner) = initialized_window();
    save_new_part(&mut owner, &path);
    let before = std::fs::read(&path).unwrap();
    parameters::create_size_parameter(&mut window, &mut owner);
    open_exit_prompt(&mut window, &mut owner);
    let result = click_exit_answer(&mut window, &mut owner, "No");
    assert!(
        result.attempts < 20,
        "application exit did not stop the loop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
    assert_eq!(std::fs::read(&path).unwrap(), before);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_exit_save_updates_file_before_exit() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("save.opd");
    let (mut window, mut owner) = initialized_window();
    save_new_part(&mut owner, &path);
    let before = std::fs::read(&path).unwrap();
    parameters::create_size_parameter(&mut window, &mut owner);
    open_exit_prompt(&mut window, &mut owner);
    let result = click_exit_answer(&mut window, &mut owner, "Yes");
    assert!(
        result.attempts < 20,
        "application exit did not stop the loop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
    assert_ne!(std::fs::read(&path).unwrap(), before);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_exit_cancel_first_save_as_keeps_document_open() {
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    parameters::create_size_parameter(&mut window, &mut owner);
    let document = owner.documents().workspace().active_document().unwrap();
    open_exit_prompt(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Yes"),
    );
    assert!(application_bdd::window_geometry("Save As").is_some());
    press_shell_key(
        &mut window,
        &mut owner,
        oblikovati_viewport::ViewportPhysicalKey::Escape,
    );
    assert!(application_bdd::window_geometry("Save As").is_none());
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    assert!(!owner.close_allowed().unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn save_new_part(owner: &mut NativeDesktopOwner, path: &std::path::Path) {
    owner.create_part_document().unwrap();
    let document = owner.documents().workspace().active_document().unwrap();
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.to_str().unwrap().to_owned(),
        })
        .unwrap();
}

fn open_exit_prompt(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    assert!(!owner.close_allowed().unwrap());
    window.scripted.push(ViewportEvent::CloseRequested);
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert!(application_bdd::window_geometry("Save changes").is_some());
}

fn click_exit_answer(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    label: &str,
) -> HeadSummary {
    let position = application_bdd::rendered_label_center(label);
    window.scripted = pointer_click_events(position, 1);
    crate::frames::run_frames(window, HeadOptions { frames: Some(20) }, owner).unwrap()
}

#[test]
fn native_exit_discard_from_active_sketch_keeps_saved_file() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("discard-sketch.opd");
    let (mut window, mut owner) = initialized_window();
    save_new_part(&mut owner, &path);
    let before = std::fs::read(&path).unwrap();
    begin_edited_sketch(&mut window, &mut owner);
    open_sketch_exit_prompt(&mut window, &mut owner);
    let result = click_exit_answer(&mut window, &mut owner, "No");
    assert!(
        result.attempts < 20,
        "application exit did not stop the loop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
    assert_eq!(std::fs::read(&path).unwrap(), before);
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn native_exit_save_from_active_sketch_persists_sketch() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("save-sketch.opd");
    let (mut window, mut owner) = initialized_window();
    save_new_part(&mut owner, &path);
    begin_edited_sketch(&mut window, &mut owner);
    open_sketch_exit_prompt(&mut window, &mut owner);
    let result = click_exit_answer(&mut window, &mut owner, "Yes");
    assert!(
        result.attempts < 20,
        "application exit did not stop the loop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
    let saved = std::fs::read(&path).unwrap();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&saved).unwrap();
    let sketches = package.restore_sketches().unwrap().sketches;
    assert_eq!(sketches.sketches().len(), 1);
    assert_eq!(sketches.sketches()[0].geometry().lines().len(), 1);
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn begin_edited_sketch(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    use oblikovati_model::{
        document_sketch::SketchPlacement,
        history::DocumentSketchCreation,
        sketch::{SketchPlacementRecipe, SketchPlane},
    };
    let document = owner.documents().workspace().active_document().unwrap();
    let sketch = owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "Exit sketch".into(),
                placement: SketchPlacement::Fixed(SketchPlane::xy()),
                orientation: None,
                project_origin: false,
            },
        )
        .unwrap();
    let line = SketchPlacementRecipe::line_chain(&[[0., 2.], [5., 2.]], false).unwrap();
    owner
        .commit_sketch_placement(document, sketch, &line, &[])
        .unwrap();
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert!(owner.active_sketch_edit().is_some());
}

fn open_sketch_exit_prompt(window: &mut ScriptedGuiWindow, owner: &mut NativeDesktopOwner) {
    window.scripted.push(ViewportEvent::CloseRequested);
    crate::frames::run_frames(window, HeadOptions { frames: Some(3) }, owner).unwrap();
    assert!(application_bdd::window_geometry("Save changes").is_some());
}

#[test]
fn native_exit_first_save_as_creates_file_and_exits() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("first-save.opd");
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    parameters::create_size_parameter(&mut window, &mut owner);
    open_exit_prompt(&mut window, &mut owner);
    click_shell_position(
        &mut window,
        &mut owner,
        application_bdd::rendered_label_center("Yes"),
    );
    assert!(application_bdd::window_geometry("Save As").is_some());
    type_native_text(&mut window, &mut owner, path.to_str().unwrap());
    let result = accept_first_save(&mut window, &mut owner);
    assert!(
        result.attempts < 20,
        "application exit did not stop the loop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
    assert!(path.is_file());
    oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(path).unwrap())
        .unwrap();
    owner.finish().unwrap();
    window.finish().unwrap();
}

fn accept_first_save(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
) -> HeadSummary {
    let key = oblikovati_viewport::ViewportPhysicalKey::Enter;
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
    crate::frames::run_frames(window, HeadOptions { frames: Some(20) }, owner).unwrap()
}

#[test]
fn guarded_close_exposes_dialog_and_accessible_cancel_keeps_edits() {
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    dirty_accessibility_fixture(&mut owner);
    let document = owner.documents().workspace().active_document().unwrap();
    window.scripted.push(ViewportEvent::CloseRequested);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let gui = window.native.gui.as_mut().unwrap();
    let (_, popup, _, prompt) = gui.frame_controls().unwrap();
    assert!(popup);
    let prompt = prompt.expect("dirty close exposes its prompt");
    assert_eq!(prompt.title, "Save changes");
    assert_eq!(
        prompt
            .buttons
            .iter()
            .map(|button| button.label)
            .collect::<Vec<_>>(),
        ["Yes", "No", "Cancel"]
    );
    assert!(
        prompt.buttons.iter().all(
            |button| button.bounds[2] > button.bounds[0] && button.bounds[3] > button.bounds[1]
        )
    );
    gui.answer_close_prompt(oblikovati_gui::GuiClosePromptChoice::Cancel);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    assert!(application_bdd::window_geometry("Save changes").is_none());
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    assert!(!owner.close_allowed().unwrap());
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
#[ignore = "run with the AT-SPI client in accessibility_atspi/modal_run.sh"]
fn accessible_dirty_close_live_fixture() {
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    dirty_accessibility_fixture(&mut owner);
    if let Some(path) = std::env::var_os("OBLIKOVATI_A11Y_READY") {
        std::fs::write(path, b"ready").unwrap();
    }
    crate::run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    assert!(owner.documents().workspace().documents().is_empty());
}

fn dirty_accessibility_fixture(owner: &mut NativeDesktopOwner) {
    use oblikovati_api::{
        types::{Variant, VariantValue},
        wire::SetPropertyArgs,
    };
    let document = owner.documents().workspace().active_document().unwrap();
    Client::new(owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: oblikovati_model::property::USER_DEFINED.into(),
            name: "accessible-close".into(),
            value: Variant {
                value: VariantValue::String("edited".into()),
                ..Default::default()
            },
        })
        .unwrap();
}

#[test]
#[ignore = "run with the AT-SPI client in accessibility_atspi/sketch_run.sh"]
fn accessible_sketch_exit_live_fixture() {
    let (mut window, mut owner) = initialized_window();
    owner.create_part_document().unwrap();
    begin_edited_sketch(&mut window, &mut owner);
    let state = owner.gui_command_state().unwrap();
    let document = state.document.unwrap();
    let name = &state
        .documents
        .iter()
        .find(|row| row.id == document)
        .unwrap()
        .name;
    let position = application_bdd::tab_label_center(name);
    window
        .scripted
        .push(ViewportEvent::Input(ViewportInput::Motion {
            position,
            modifiers: Default::default(),
        }));
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let close = application_bdd::tab_close_mark(name);
    click_shell_position(&mut window, &mut owner, close);
    assert!(application_bdd::window_geometry("Exit environment").is_some());
    window.observe_sketch_after_prompt = true;
    if let Some(path) = std::env::var_os("OBLIKOVATI_A11Y_READY") {
        std::fs::write(path, b"ready").unwrap();
    }
    crate::run_window_with_owner(&mut window, HeadOptions::default(), &mut owner).unwrap();
    if std::env::var("OBLIKOVATI_SKETCH_ANSWER").as_deref() != Ok("OK") {
        assert_eq!(
            window.sketch_after_prompt,
            Some(true),
            "Cancel must leave the sketch active"
        );
    }
    assert!(owner.documents().workspace().documents().is_empty());
}
