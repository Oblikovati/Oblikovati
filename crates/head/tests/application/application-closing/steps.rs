// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use crate::HeadWork;
use cucumber::{given, then, when};
#[cfg(not(target_os = "linux"))]
use oblikovati_viewport::ViewportEvent;

#[cfg(target_os = "linux")]
mod x11_close;

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
    x11_close::request_native_close();
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
        x11_close::click(position);
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
