// SPDX-License-Identifier: GPL-2.0-only
//! Native window-frame behavior driven by the shared application scenarios.
use super::ApplicationWorld;
use cucumber::{then, when};

#[when("the user invokes the desktop window Close shortcut")]
fn invoke_desktop_close(world: &mut ApplicationWorld) {
    world.application_close_requested = true;
    super::x11_input::press_alt_f4();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the main window remains open and the part retains its unsaved changes")]
fn unsaved_part_remains_open(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited part");
    assert!(world.document_is_open(document));
    assert!(world.state().document_storage[&document].unsaved_edits);
}
