// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
#[cfg(not(target_os = "linux"))]
use oblikovati_viewport::ViewportEvent;

#[path = "fixtures.rs"]
mod fixtures;
pub(super) use fixtures::save_staged_part_on_exit;

#[given("the MCAD Software has an open part document with unsaved changes")]
fn dirty_part(world: &mut ApplicationWorld) {
    let document =
        if super::scenario_capture::is_main_window_frame(world.scenario_feature_path.as_deref()) {
            world
                .closed_document
                .unwrap_or_else(|| super::main_window_frame::open_dirty_frame_part(world))
        } else {
            let document = world.add_part();
            world.start_sketch();
            document
        };
    world.closed_document = Some(document);
    assert!(world.state().document_storage[&document].unsaved_edits);
}

#[given("the MCAD Software has an open saved document with no unsaved changes")]
fn clean_saved_part(world: &mut ApplicationWorld) {
    let document = world.add_part();
    world.save_part_to_file(document, "application-close-clean.opd");
    assert!(world.app().owner.close_allowed().unwrap());
}

#[given("the MCAD Software is open with no document open")]
fn no_document(world: &mut ApplicationWorld) {
    assert!(world.state().documents.is_empty());
    assert!(world.app().owner.close_allowed().unwrap());
}

#[given(
    "Home is the only open document tab with no model, drawing, or presentation document open in the MCAD Software"
)]
fn home_only(world: &mut ApplicationWorld) {
    super::home_recent_documents::open_home(world);
    world.expect_tab("Home");
    no_document(world);
}

#[when("the user selects the application window's Close control")]
pub(super) fn request_application_close(world: &mut ApplicationWorld) {
    world.application_close_requested = true;
    let can_close = world.app().owner.close_allowed().unwrap();
    if can_close {
        // A successful clean exit destroys the drawable before the final capture hook.
        world.save_capture("application-closing-ready");
    }
    #[cfg(target_os = "linux")]
    if world.app().window.native.view.uses_wayland() {
        return close_from_wayland_frame(world, can_close);
    }
    #[cfg(target_os = "linux")]
    {
        super::x11_input::focus_window();
        let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
        let close = [width - 24, 17];
        super::x11_input::move_pointer(close);
        world.pump(Vec::new());
        if can_close {
            super::x11_input::click(close);
        } else {
            super::x11_input::left_button(close, true);
            world.pump(Vec::new());
            super::x11_input::left_button(close, false);
        }
    }
    #[cfg(not(target_os = "linux"))]
    if !can_close {
        return world.pump(vec![ViewportEvent::CloseRequested]);
    } else {
        world
            .app()
            .window
            .scripted
            .push(ViewportEvent::CloseRequested);
    }
    if can_close {
        return finish_clean_application(world);
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
fn close_from_wayland_frame(world: &mut ApplicationWorld, can_close: bool) {
    let close = super::main_window_frame_native::frame_control_center(2);
    let events = super::pointer_click_events(close, 1);
    if can_close {
        world.app().window.scripted = events;
        finish_clean_application(world);
    } else {
        world.pump(events);
        world.pump(Vec::new());
    }
}

fn finish_clean_application(world: &mut ApplicationWorld) {
    let mut app = world.app.take().expect("expected an open application");
    crate::run_window_with_owner(
        &mut app.window,
        crate::HeadOptions::default(),
        &mut app.owner,
    )
    .expect("clean application: expected the native close to finish");
    assert_eq!(app.window.confirmation_requests, 0);
}

#[then("the MCAD Software closes without a save-changes prompt")]
fn application_closed_without_prompt(world: &mut ApplicationWorld) {
    assert!(world.app.is_none());
}

pub(super) fn cancel_application_close(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    if world.app().window.native.view.uses_wayland() {
        world.click_text("Cancel");
    } else {
        let position = super::rendered_label_center("Cancel");
        super::x11_input::click(position);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.click_text("Cancel");
    world.application_close_requested = false;
}

#[then("a save-changes prompt identifies the part document")]
#[then("a save-changes prompt identifies the assembly document")]
fn prompt_identifies_document(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited document");
    let name = world.document_name(document);
    world.expect_text(&format!("Save changes to {name}"));
    world.save_capture("application-closing-dirty-prompt");
}

#[then("the save-changes prompt closes")]
fn prompt_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("before closing");
}

#[then("the part document remains open")]
#[then("the assembly document remains open")]
fn document_remains_open(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited document");
    assert!(world.document_is_open(document));
}

#[then("the prompt shows the \"Always show this message\" setting")]
fn prompt_shows_frequency(world: &mut ApplicationWorld) {
    world.expect_text("before closing");
    world.expect_text("Always show this message");
}

#[then("the MCAD Software remains open")]
fn application_remains_open(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    assert!(world.app.is_some());
}

#[cfg(target_os = "linux")]
#[test]
fn native_close_cancels_after_discarding_only_the_confirmed_document() {
    let mut world = ApplicationWorld::new();
    let (first, first_path) = dirty_saved_part(&mut world, "first-close.opd");
    let first_bytes = std::fs::read(&first_path).unwrap();
    let (second, second_path) = dirty_saved_part(&mut world, "second-close.opd");
    let second_bytes = std::fs::read(&second_path).unwrap();
    let first_name = world.document_name(first);
    let second_name = world.document_name(second);
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    world.expect_text(&format!("Save changes to {second_name}"));
    world.save_capture("application-close-two-dirty-first");
    click_close_answer(&mut world, "No");
    world.expect_text(&format!("Save changes to {first_name}"));
    world.save_capture("application-close-two-dirty-second");
    assert!(!world.document_is_open(second));
    assert!(world.document_is_open(first));
    click_close_answer(&mut world, "Cancel");
    world.expect_no_text("before closing");
    assert!(world.document_is_open(first));
    assert!(world.state().document_storage[&first].unsaved_edits);
    assert_eq!(std::fs::read(first_path).unwrap(), first_bytes);
    assert_eq!(std::fs::read(second_path).unwrap(), second_bytes);
    world.close();
}

#[cfg(target_os = "linux")]
fn dirty_saved_part(world: &mut ApplicationWorld, name: &str) -> (u64, std::path::PathBuf) {
    let document = world.add_part();
    let path = world.save_part_to_file(document, name);
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            value_type: "numeric".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    assert!(world.state().document_storage[&document].unsaved_edits);
    (document, path)
}

#[cfg(target_os = "linux")]
fn click_close_answer(world: &mut ApplicationWorld, answer: &str) {
    let position = super::rendered_label_center(answer);
    super::x11_input::click(position);
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
#[test]
fn native_exit_save_failure_keeps_all_modified_documents_open() {
    let mut world = ApplicationWorld::new();
    let (first, first_path) = dirty_saved_part(&mut world, "first-failure.opd");
    let first_bytes = std::fs::read(&first_path).unwrap();
    let (second, second_path) = dirty_saved_part(&mut world, "second-failure.opd");
    let second_bytes = std::fs::read(&second_path).unwrap();
    let backup = second_path.with_extension("original.opd");
    std::fs::rename(&second_path, &backup).unwrap();
    std::fs::create_dir(&second_path).unwrap();
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    click_close_answer(&mut world, "Yes");
    world.pump(Vec::new());
    assert!(world.document_is_open(first));
    assert!(world.document_is_open(second));
    assert!(world.state().document_storage[&first].unsaved_edits);
    assert!(world.state().document_storage[&second].unsaved_edits);
    assert_eq!(std::fs::read(&first_path).unwrap(), first_bytes);
    assert_eq!(std::fs::read(&backup).unwrap(), second_bytes);
    assert!(second_path.is_dir());
    world.expect_status("Save document");
    world.save_capture("application-close-save-failure");
    std::fs::remove_dir(&second_path).unwrap();
    std::fs::rename(&backup, &second_path).unwrap();
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    click_close_answer(&mut world, "Yes");
    assert!(!world.document_is_open(second));
    assert!(world.document_is_open(first));
    assert_ne!(std::fs::read(&second_path).unwrap(), second_bytes);
    click_close_answer(&mut world, "Cancel");
    assert!(world.document_is_open(first));
    assert!(world.state().document_storage[&first].unsaved_edits);
    assert_eq!(std::fs::read(first_path).unwrap(), first_bytes);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_shared_discard_closes_both_dirty_documents() {
    shared_answer_closes_both("No");
}

#[cfg(target_os = "linux")]
#[test]
fn native_shared_save_closes_both_dirty_documents() {
    shared_answer_closes_both("Yes");
}

#[cfg(target_os = "linux")]
fn shared_answer_closes_both(answer: &str) {
    use crate::HeadWindow;
    use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
    let mut world = ApplicationWorld::new();
    let (first, first_path) = dirty_saved_part(&mut world, "shared-first.opd");
    let first_bytes = std::fs::read(&first_path).unwrap();
    let (_, second_path) = dirty_saved_part(&mut world, "shared-second.opd");
    let second_bytes = std::fs::read(&second_path).unwrap();
    world
        .app()
        .owner
        .set_prompt_frequency_from_gui(
            ApplicationPrompt::SaveChanges,
            PromptFrequency::OncePerOperation,
        )
        .unwrap();
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    world.expect_text("before closing");
    world.expect_text("This answer also applies to:");
    let first_name = world.document_name(first);
    world.expect_text(&first_name);
    world.save_capture(&format!("shared-close-{answer}-4056"));
    super::x11_input::click(super::rendered_label_center(answer));
    let mut app = world.app.take().unwrap();
    let result = crate::frames::run_frames(
        &mut app.window,
        crate::HeadOptions { frames: Some(30) },
        &mut app.owner,
    )
    .unwrap();
    assert!(result.attempts < 30, "exit did not finish after one answer");
    assert!(app.owner.documents().workspace().documents().is_empty());
    let changed = answer == "Yes";
    assert_eq!(std::fs::read(first_path).unwrap() != first_bytes, changed);
    assert_eq!(std::fs::read(second_path).unwrap() != second_bytes, changed);
    app.owner.finish().unwrap();
    app.window.finish().unwrap();
}

#[cfg(target_os = "linux")]
#[test]
fn native_shared_save_stops_when_a_later_document_cannot_be_saved() {
    use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
    let mut world = ApplicationWorld::new();
    let (first, first_path) = dirty_saved_part(&mut world, "shared-failure-first.opd");
    let first_bytes = std::fs::read(&first_path).unwrap();
    let (second, second_path) = dirty_saved_part(&mut world, "shared-failure-second.opd");
    let second_bytes = std::fs::read(&second_path).unwrap();
    let backup = first_path.with_extension("original.opd");
    std::fs::rename(&first_path, &backup).unwrap();
    std::fs::create_dir(&first_path).unwrap();
    world
        .app()
        .owner
        .set_prompt_frequency_from_gui(
            ApplicationPrompt::SaveChanges,
            PromptFrequency::OncePerOperation,
        )
        .unwrap();
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    click_close_answer(&mut world, "Yes");
    world.pump(Vec::new());
    assert!(world.document_is_open(first));
    assert!(!world.document_is_open(second));
    assert!(world.state().document_storage[&first].unsaved_edits);
    assert_eq!(std::fs::read(&backup).unwrap(), first_bytes);
    assert_ne!(std::fs::read(second_path).unwrap(), second_bytes);
    world.expect_status("Save document");
    std::fs::remove_dir(&first_path).unwrap();
    std::fs::rename(backup, first_path).unwrap();
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn native_shared_first_save_cancel_keeps_both_documents_open() {
    use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
    let mut world = ApplicationWorld::new();
    let (first, first_path) = dirty_saved_part(&mut world, "shared-cancel-first.opd");
    let first_bytes = std::fs::read(&first_path).unwrap();
    let second = world.add_part();
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            value_type: "numeric".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world
        .app()
        .owner
        .set_prompt_frequency_from_gui(
            ApplicationPrompt::SaveChanges,
            PromptFrequency::OncePerOperation,
        )
        .unwrap();
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    click_close_answer(&mut world, "Yes");
    assert!(super::window_geometry("Save As").is_some());
    super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(super::window_geometry("Save As").is_none());
    assert!(world.document_is_open(first) && world.document_is_open(second));
    assert!(world.state().document_storage[&first].unsaved_edits);
    assert!(world.state().document_storage[&second].unsaved_edits);
    assert_eq!(std::fs::read(first_path).unwrap(), first_bytes);
    super::x11_input::request_native_close();
    world.pump(Vec::new());
    world.expect_text("before closing");
    click_close_answer(&mut world, "Cancel");
    world.close();
}
