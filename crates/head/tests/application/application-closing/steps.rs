// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
#[cfg(not(target_os = "linux"))]
use oblikovati_viewport::ViewportEvent;

#[given("the MCAD Software has an open part document with unsaved changes")]
fn dirty_part(world: &mut ApplicationWorld) {
    let document = world.add_part();
    world.start_sketch();
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

#[when("the user selects the application window's Close control")]
fn request_application_close(world: &mut ApplicationWorld) {
    world.application_close_requested = true;
    let can_close = world.app().owner.close_allowed().unwrap();
    #[cfg(target_os = "linux")]
    super::x11_input::request_native_close();
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
    {
        let position = super::rendered_label_center("Cancel");
        super::x11_input::click(position);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.click_text("Cancel");
    world.application_close_requested = false;
}

#[then("a save-changes prompt identifies the part document")]
fn prompt_identifies_part(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited part");
    let name = world.document_name(document);
    world.expect_text(&format!("Save changes to {name}"));
    world.save_capture("application-closing-dirty-prompt");
}

#[then("the save-changes prompt closes")]
fn prompt_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("before closing");
}

#[then("the part document remains open")]
fn document_remains_open(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited part");
    assert!(world.document_is_open(document));
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
