// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use cucumber::{then, when};
#[when("I select File")]
fn select_file(world: &mut ApplicationWorld) {
    world.click_text("File");
}
#[then("a menu opens with New, Open, Save, Save As, Export, Print and Close entries")]
fn menu_entries(world: &mut ApplicationWorld) {
    for entry in ["New", "Open", "Save", "Save As", "Export", "Print", "Close"] {
        world.expect_text(entry);
    }
}
#[then("a recent-documents region is visible")]
fn recent_documents(world: &mut ApplicationWorld) {
    world.expect_text("Recent documents");
}
#[then("the model workspace remains visible behind the menu")]
fn workspace_behind_menu(world: &mut ApplicationWorld) {
    let (position, size) = super::diagnostics::window_geometry("##application-ribbon").unwrap();
    let extent = world.extent();
    assert_eq!(position, [0.; 2]);
    assert!(
        size[1] < extent[1] as f32 / 2.,
        "File menu: expected the model region to remain available"
    );
    world.expect_text("Origin");
}
