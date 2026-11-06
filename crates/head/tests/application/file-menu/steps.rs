// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use cucumber::{given, then, when};
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
fn active_document(world: &mut ApplicationWorld) -> u64 {
    world.active_id()
}
#[then("the menu also shows Share, Manage and document properties entries")]
fn more_entries(world: &mut ApplicationWorld) {
    for entry in ["Share", "Manage", "Document Properties"] {
        world.expect_text(entry);
    }
}
#[then("application options and an exit control appear at the bottom of the menu")]
fn bottom_controls(world: &mut ApplicationWorld) {
    let recent = world.run("Recent documents").center()[1];
    for entry in ["Options", "Exit"] {
        let bottom = world.run(entry).center()[1];
        assert!(
            bottom > recent,
            "{entry}: expected it below the recent documents"
        );
    }
}
#[given("an empty part document is active in the MCAD Software")]
fn empty_part(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
}
#[given("the File menu is open")]
fn file_menu_open(world: &mut ApplicationWorld) {
    world.previous_document = Some(active_document(world));
    world.click_text("File");
    world.expect_text("Recent documents");
}
#[when("I press Escape")]
fn press_escape(world: &mut ApplicationWorld) {
    let active = active_document(world);
    world.previous_document.get_or_insert(active);
    world.escape();
}
#[when("I click an empty area of the viewport outside the menu")]
fn click_outside(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
}
#[then("the File menu and recent-documents region close")]
#[then("the File menu closes")]
fn file_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Recent documents");
    world.expect_no_text("Save As");
}
#[then("the same part document remains active")]
#[then("the same drawing document remains active")]
fn same_document(world: &mut ApplicationWorld) {
    let id = world.previous_document.expect("expected a prior document");
    assert_eq!(active_document(world), id);
}
#[given("a drawing document is active in the MCAD Software")]
fn drawing_active(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_drawing());
}
#[then(
    "the menu shows New, Open, Save, Save As, Export, Share, Manage, document properties, Print and Close entries"
)]
fn drawing_menu_entries(world: &mut ApplicationWorld) {
    for entry in [
        "New",
        "Open",
        "Save",
        "Save As",
        "Export",
        "Share",
        "Manage",
        "Document Properties",
        "Print",
        "Close",
    ] {
        world.expect_text(entry);
    }
}
#[then("the drawing sheet remains visible around the menu")]
fn sheet_around_menu(world: &mut ApplicationWorld) {
    let sheet = world.pixel_at([1000, 500]);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "sheet {sheet:?}: expected the light sheet beside the menu"
    );
}
#[given("recent documents are available")]
fn recent_available(world: &mut ApplicationWorld) {
    let part = world.active_id();
    world.app().owner.forget_recent_documents();
    world.save_part_to_file(part, "Recent.opd");
}
#[then("the recent-documents region shows an ordering control labeled By Ordered List")]
fn ordering_control(world: &mut ApplicationWorld) {
    world.expect_text("By Ordered List");
}
#[then("a separate display control is visible beside it")]
fn display_control(world: &mut ApplicationWorld) {
    let ordering = world.run("By Ordered List").clone();
    let display = world.run("Name").clone();
    assert!(
        display.minimum[0] > ordering.maximum[0],
        "display at {:?}: expected beside the ordering control",
        display.minimum
    );
}
fn recent_row(world: &ApplicationWorld) -> super::diagnostics::RenderedRun {
    world.run("Recent").clone()
}
#[then("each visible recent-document row shows a document-type symbol and name on the left")]
fn row_left(world: &mut ApplicationWorld) {
    let name = recent_row(world);
    let symbol = world.frame.runs.iter().any(|run| {
        run.text == "P"
            && run.maximum[0] < name.minimum[0]
            && (run.center()[1] - name.center()[1]).abs() < 8.
    });
    assert!(symbol, "expected a type symbol before the name");
}
#[then("each visible row shows a pin control on the right")]
fn row_pin(world: &mut ApplicationWorld) {
    let name = recent_row(world);
    let pin = world.frame.runs.iter().any(|run| {
        run.text == "Pin"
            && run.minimum[0] > name.maximum[0]
            && (run.center()[1] - name.center()[1]).abs() < 8.
    });
    assert!(pin, "expected a pin control after the name");
}
#[then("the active part remains visible around the menu")]
fn part_around_menu(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
}
