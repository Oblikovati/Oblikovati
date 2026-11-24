// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use cucumber::{given, then, when};

#[cfg(test)]
#[path = "native_input.rs"]
mod native_input;
#[path = "saved_part.rs"]
mod saved_part;
#[given("the user selects File")]
#[when("I select File")]
#[when("the user selects File")]
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
    if let Some(document) = world.previous_document {
        let state = world.state();
        assert_eq!(state.document, Some(document));
        assert_eq!(
            state.document_type,
            Some(oblikovati_api::types::DOCUMENT_PART)
        );
        let browser = state.browser.expect("expected the active Part browser");
        assert!(browser.sketches.is_empty() && browser.bodies.is_empty());
    } else {
        world.previous_document = Some(world.add_part());
    }
}
#[given("the File menu is open")]
fn file_menu_open(world: &mut ApplicationWorld) {
    world.previous_document = Some(active_document(world));
    if !world.shows("Save As") {
        select_file(world);
    }
    world.expect_text("Recent documents");
}
#[when("I press Escape")]
#[when("the user press Escape")]
fn press_escape(world: &mut ApplicationWorld) {
    let active = active_document(world);
    world.previous_document.get_or_insert(active);
    world.escape();
}
#[when("I click an empty area of the viewport outside the menu")]
#[when("the user clicks an empty area of the viewport outside the menu")]
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
#[given("that opened part is listed in recent documents")]
fn opened_part_is_recent(world: &mut ApplicationWorld) {
    let document = active_document(world);
    let state = world.state();
    let path = state.document_storage[&document]
        .path
        .as_ref()
        .expect("expected the opened Part path");
    assert!(
        state
            .recent_documents
            .iter()
            .any(|entry| entry.path == *path),
        "expected the opened Part in recent documents"
    );
    world.saved_path = Some(std::path::PathBuf::from(path));
}

#[given("recent documents are available")]
fn recent_available(world: &mut ApplicationWorld) {
    if world.saved_path.is_some() {
        assert!(!world.state().recent_documents.is_empty());
        return;
    }
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
fn recent_name(world: &ApplicationWorld) -> &str {
    world
        .saved_path
        .as_ref()
        .and_then(|path| path.file_stem())
        .and_then(|stem| stem.to_str())
        .unwrap_or("Recent")
}
fn recent_row(world: &ApplicationWorld) -> &super::diagnostics::RenderedRun {
    let name = recent_name(world);
    let ordering = world.run("By Ordered List");
    let rows = world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == ordering.list);
    rows.clone()
        .find(|run| run.text.contains(name) && run.center()[1] > ordering.center()[1])
        .unwrap_or_else(|| panic!("expected {name:?} in the File menu rows {rows:?}"))
}
#[then("each visible recent-document row shows a document-type symbol and name on the left")]
fn row_left(world: &mut ApplicationWorld) {
    let row = recent_row(world);
    let (left, _) = row.text.split_once(recent_name(world)).unwrap();
    let symbol = left == "P"
        || world.frame.runs.iter().any(|run| {
            run.list == row.list
                && run.text == "P"
                && run.maximum[0] < row.minimum[0]
                && (run.center()[1] - row.center()[1]).abs() < 8.
        });
    assert!(symbol, "expected a type symbol before the name");
}
#[then("each visible row shows a pin control on the right")]
fn row_pin(world: &mut ApplicationWorld) {
    let row = recent_row(world);
    let (_, right) = row.text.split_once(recent_name(world)).unwrap();
    let pin = right == "Pin"
        || world.frame.runs.iter().any(|run| {
            run.list == row.list
                && run.text == "Pin"
                && run.minimum[0] > row.maximum[0]
                && (run.center()[1] - row.center()[1]).abs() < 8.
        });
    assert!(pin, "expected a pin control after the name");
}
#[then("the active part remains visible around the menu")]
fn part_around_menu(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
}

#[then("the region lists document names with document-type symbols and pin controls")]
fn recent_rows_with_controls(world: &mut ApplicationWorld) {
    row_left(world);
    row_pin(world);
}

#[test]
fn alt_then_f_opens_file_menu_without_changing_the_document() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    world.key(ViewportPhysicalKey::LeftAlt, false);
    assert!(
        world.frame.runs.iter().any(|run| run.text == "F"),
        "expected the File key tip"
    );
    world.key(ViewportPhysicalKey::F, false);
    world.pump(Vec::new());
    world.expect_text("Recent documents");
    world.expect_text("Save As");
    world.escape();
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    world.close();
}

fn press_alt_f(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{
        ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey,
    };
    let alt = ViewportModifiers {
        alt: true,
        ..Default::default()
    };
    let key = |key, pressed, modifiers| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key,
            layout_key: key,
            pressed,
            command: None,
            modifiers,
        })
    };
    world.pump(vec![key(ViewportPhysicalKey::LeftAlt, true, alt)]);
    world.pump(vec![
        key(ViewportPhysicalKey::F, true, alt),
        key(ViewportPhysicalKey::F, false, alt),
    ]);
    world.pump(vec![key(
        ViewportPhysicalKey::LeftAlt,
        false,
        Default::default(),
    )]);
}
#[when("I press Alt+F to open the File menu")]
#[when("the user presses Alt+F to open the File menu")]
fn open_file_with_alt_f(world: &mut ApplicationWorld) {
    press_alt_f(world);
}
#[test]
fn alt_f_chord_opens_file_menu_and_keeps_the_active_part() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    press_alt_f(&mut world);
    world.expect_text("Recent documents");
    world.expect_text("Save As");
    world.escape();
    world.expect_no_text("Recent documents");
    assert_eq!(world.active_id(), part);
    world.close();
}

#[given("the pointer is positioned over the New command location")]
fn pointer_over_new_location(world: &mut ApplicationWorld) {
    let file = world.run("File");
    let position = [file.minimum[0] + 15., file.maximum[1] + 12.];
    world.hover(super::diagnostics::pointer_pixels(position));
}
#[then("the New submenu is displayed")]
fn new_submenu_displayed(world: &mut ApplicationWorld) {
    world.expect_text("Create a file from the available templates");
}
#[then("the submenu describes creating a file from the available templates")]
fn new_description(world: &mut ApplicationWorld) {
    new_submenu_displayed(world);
}
#[then("it lists Assembly, Weldment, Drawing, Part, Sheet Metal and Presentation")]
fn new_document_types(world: &mut ApplicationWorld) {
    let list = world.run("Weldment").list;
    for label in [
        "Assembly",
        "Weldment",
        "Drawing",
        "Part",
        "Sheet Metal",
        "Presentation",
    ] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == label.replace(' ', "")),
            "expected {label:?} in the New submenu"
        );
    }
    assert!(!world.run_is_enabled("Weldment"));
}
#[then("Drawing displays a disclosure indicator")]
fn drawing_disclosure(world: &mut ApplicationWorld) {
    let row = world.run("Drawing");
    assert!(
        world.frame.marks.iter().any(|mark| mark.list == row.list
            && mark.position[0] > row.maximum[0]
            && (mark.position[1] - row.center()[1]).abs() < 8.),
        "expected a disclosure arrow beside Drawing"
    );
}
#[then("the active part remains visible behind the menu")]
fn part_behind_new_menu(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the original part");
    world.expect_browser_of(part);
}
#[then("no document type has been activated")]
fn new_menu_does_not_create_a_document(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the original part");
    assert_eq!(world.active_id(), part);
    assert_eq!(world.state().documents.len(), 1);
    world.expect_no_text("Template folders");
}

#[test]
fn new_submenu_part_opens_selected_template_and_creates_a_part() {
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    pointer_over_new_location(&mut world);
    press_alt_f(&mut world);
    new_submenu_displayed(&mut world);
    world.click_text("Part");
    world.pump(Vec::new());
    world.expect_text("Template folders");
    world.expect_text("Standard metric part");
    world.expect_text("Length: millimeters (mm)");
    assert_eq!(world.state().documents.len(), 1);
    world.click_text("Create");
    world.pump(Vec::new());
    assert_eq!(world.state().documents.len(), 2);
    assert_ne!(world.active_id(), first);
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.close();
}

#[test]
fn moving_from_new_to_save_as_switches_the_open_submenu() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    pointer_over_new_location(&mut world);
    press_alt_f(&mut world);
    new_submenu_displayed(&mut world);
    let save_as = world.run("Save As").center();
    world.hover(super::diagnostics::pointer_pixels(save_as));
    world.pump(Vec::new());
    world.expect_text("Save Copy As");
    world.expect_no_text("Create a file from the available templates");
    world.close();
}

#[test]
fn new_drawing_submenu_requires_a_template_preview_before_creation() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    pointer_over_new_location(&mut world);
    press_alt_f(&mut world);
    let drawing = world.run("Drawing").center();
    world.hover(super::diagnostics::pointer_pixels(drawing));
    world.pump(Vec::new());
    world.click_text("Choose drawing template");
    world.pump(Vec::new());
    world.expect_text("Metric drawing with title block");
    world.expect_text("Use Template");
    assert!(!world.run_is_enabled("Create"));
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert_eq!(world.state().documents.len(), 1);
    assert_eq!(world.active_id(), part);
    world.close();
}
