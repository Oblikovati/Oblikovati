// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;
use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportPhysicalKey};
#[path = "closing.rs"]
mod closing;
#[path = "detached.rs"]
mod detached;
#[path = "home_and_previews.rs"]
mod home_and_previews;
#[path = "home_preview_native.rs"]
mod home_preview_native;
#[path = "mixed_documents.rs"]
mod mixed_documents;
#[path = "reordering.rs"]
mod reordering;
#[path = "second_windows.rs"]
mod second_windows;
const TAB_STRIP: &str = "##open-document-tabs";
const EXIT_QUESTION: &str = "Exit the sketch environment and continue?";
/// More rows than the Select Window list can show, so its scrolling is measurable.
const OVERFLOWING_DOCUMENTS: usize = 16;

fn first(world: &ApplicationWorld) -> u64 {
    world.parts[0]
}
fn second(world: &ApplicationWorld) -> u64 {
    world.parts[1]
}
#[given("the MCAD Software has two open part documents")]
#[given("multiple documents are open in the MCAD Software")]
#[given("multiple part documents are open in the MCAD Software")]
fn open_two(world: &mut ApplicationWorld) {
    world.add_part();
    let active = world.add_part();
    world.previous_document = Some(active);
}
#[given("the first part contains an empty sketch")]
fn first_has_sketch(world: &mut ApplicationWorld) {
    world.activate_through_tab(first(world));
    world.add_finished_sketch();
}
#[given("the second part contains a rectangular sketch profile")]
fn second_has_rectangle(world: &mut ApplicationWorld) {
    world.activate_through_tab(second(world));
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
}
#[given("the first part is active")]
fn first_active(world: &mut ApplicationWorld) {
    world.activate_through_tab(first(world));
}
#[when("I select the second part's document tab")]
#[when("the user selects the second part's document tab")]
fn select_second(world: &mut ApplicationWorld) {
    world.click_tab(second(world));
}
#[when("I select the first part's document tab")]
#[when("the user selects the first part's document tab")]
fn select_first(world: &mut ApplicationWorld) {
    world.click_tab(first(world));
}
#[then("the second part becomes active")]
fn second_is_active(world: &mut ApplicationWorld) {
    world.expect_active(second(world));
}
#[then("its rectangular profile appears in the viewport")]
fn rectangle_appears(world: &mut ApplicationWorld) {
    assert!(
        super::rectangle_visibility::current_rectangle_visible(world),
        "expected four rectangle edges in the second part viewport"
    );
}
#[then("the model browser shows the second part")]
fn browser_second(world: &mut ApplicationWorld) {
    world.expect_browser_of(second(world));
    world.expect_browser_sketch();
}
#[then("the first part becomes active")]
fn first_is_active(world: &mut ApplicationWorld) {
    world.expect_active(first(world));
    world.expect_browser_sketch();
    assert!(
        !super::rectangle_visibility::current_rectangle_visible(world),
        "the first part viewport must not show the second part rectangle"
    );
}
#[then("its model browser and viewport are displayed")]
fn browser_and_viewport(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
    world.expect_blank_region([400, 600], [250, 300]);
}
#[then("the second part remains available as an open tab")]
fn second_tab_remains(world: &mut ApplicationWorld) {
    let name = world.document_name(second(world));
    world.expect_tab(&name);
}
#[when("I open the document tab's context menu")]
#[when("the user opens the document tab's context menu")]
fn open_menu(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.open_tab_menu(id);
}
#[then("the menu offers Close, Close Tabs, Close All, Save and Properties")]
fn menu_entries(world: &mut ApplicationWorld) {
    for entry in ["Close", "Close Tabs", "Close All", "Save", "Properties"] {
        world.expect_text(entry);
    }
}
#[then("it displays shortcuts for Close and Save")]
fn menu_shortcuts(world: &mut ApplicationWorld) {
    world.expect_text("Ctrl+W");
    world.expect_text("Ctrl+S");
}
#[when("I dismiss the menu")]
#[when("the user dismisses the menu")]
fn dismiss_menu(world: &mut ApplicationWorld) {
    world.escape();
    world.expect_no_text("Close Tabs");
}
#[then("the document remains active")]
fn document_remains(world: &mut ApplicationWorld) {
    let id = world.previous_document.expect("expected a document");
    world.expect_active(id);
}
#[given("document tabs are visible")]
fn tabs_visible(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_tab(&name);
}
fn toggle_tabs_option(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Document Tabs");
    world.escape();
}
#[when("I clear Document Tabs in the View tab's User Interface menu")]
#[when("I select Document Tabs in the same menu")]
#[when("the user clears Document Tabs in the View tab's User Interface menu")]
#[when("the user selects Document Tabs in the same menu")]
fn toggle_tabs(world: &mut ApplicationWorld) {
    toggle_tabs_option(world);
}
#[given("document tabs are hidden")]
fn tabs_hidden(world: &mut ApplicationWorld) {
    while world.parts.len() < OVERFLOWING_DOCUMENTS {
        world.add_part();
    }
    toggle_tabs_option(world);
    assert!(
        window_geometry(TAB_STRIP).is_none(),
        "expected a hidden tab strip"
    );
}
#[then("the document tab strip disappears")]
fn strip_gone(world: &mut ApplicationWorld) {
    assert!(
        window_geometry(TAB_STRIP).is_none(),
        "expected a hidden tab strip"
    );
    let id = world.active_id();
    let name = world.document_name(id);
    assert!(
        world.tab_run(&name).is_none(),
        "tab {name:?}: expected it hidden"
    );
}
#[then("the document tab strip reappears")]
fn strip_back(world: &mut ApplicationWorld) {
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_tab(&name);
}
#[then("the open documents are available as tabs")]
fn all_tabs(world: &mut ApplicationWorld) {
    for id in world.parts.clone() {
        let name = world.document_name(id);
        world.expect_tab(&name);
    }
}
#[when("I open the Switch menu on the View tab")]
#[when("the user opens the Switch menu on the View tab")]
fn open_switch(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("Switch");
}
#[then("the menu lists open windows and offers More Windows")]
fn switch_entries(world: &mut ApplicationWorld) {
    for id in world.parts.clone() {
        let name = world.document_name(id);
        world.expect_text(&name);
    }
    world.expect_text("More Windows");
}
#[when("I choose More Windows")]
#[when("the user chooses More Windows")]
fn more_windows(world: &mut ApplicationWorld) {
    world.click_text("More Windows");
}
#[then("a Select Window dialog displays a scrollable list of open documents")]
fn select_window_list(world: &mut ApplicationWorld) {
    world.expect_text("Select Window");
    // A middle row stays visible after scrolling, so the same row is measured twice.
    let name = world.document_name(world.parts[7]);
    let top = world.run(&name).center();
    let at = top.map(|value| value as i32);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
        position: at,
        modifiers: Default::default(),
    })]);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches: -2,
        position: at,
    })]);
    let scrolled = world.run(&name).center();
    assert!(
        scrolled[1] < top[1],
        "list row {top:?}, {scrolled:?}: expected a scrolled list"
    );
}
#[then("the dialog offers OK and Cancel")]
#[then("the dialog offers OK and cancels")]
fn ok_and_cancel(world: &mut ApplicationWorld) {
    world.expect_text("OK");
    world.expect_text("Cancel");
}
#[when("I press Control+Tab")]
#[when("the user presses Control+Tab")]
fn control_tab(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, true);
}
#[given("the second part has been saved")]
#[given("the second part is saved and has no unsaved changes")]
fn second_saved(world: &mut ApplicationWorld) {
    let id = second(world);
    let path = world.save_part_to_file(id, "second part.opd");
    world.bytes_before_save = Some(std::fs::read(&path).unwrap());
    world.saved_path = Some(path);
    world.activate_through_tab(first(world));
    assert!(!world.state().document_storage[&id].unsaved_edits);
}
#[when("I open the second part's document tab context menu")]
#[when("the user opens the second part's document tab context menu")]
fn open_second_menu(world: &mut ApplicationWorld) {
    world.open_tab_menu(second(world));
}
#[then("the first part remains active in the workspace")]
fn first_remains(world: &mut ApplicationWorld) {
    world.expect_active(first(world));
}
#[when("I choose Properties from the context menu")]
#[when("the user chooses Properties from the context menu")]
fn choose_properties(world: &mut ApplicationWorld) {
    world.click_text("Properties");
}
#[then("the properties dialog identifies the second part")]
fn properties_title(world: &mut ApplicationWorld) {
    let name = world.document_name(second(world));
    world.expect_text(&format!("{name} Properties"));
}
#[then("its General page displays the file name, location, size and timestamps")]
fn general_page(world: &mut ApplicationWorld) {
    world.expect_text("General");
    let folder = world.scratch_folder();
    world.expect_text("File name: second part.opd");
    world.expect_text(&format!("Location: {}", folder.display()));
    let size = std::fs::metadata(folder.join("second part.opd"))
        .unwrap()
        .len();
    world.expect_text(&format!("Size: {size} bytes"));
    for stamp in ["Created:", "Modified:", "Accessed:"] {
        world.expect_text(stamp);
    }
}
#[when("I close the properties dialog without making changes")]
#[when("the user closes the properties dialog without making changes")]
fn close_properties(world: &mut ApplicationWorld) {
    world.click_text("Close");
    world.expect_no_text("General");
}

#[test]
fn document_properties_title_close_preserves_the_active_part() {
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    let second = world.add_part();
    world.parts = vec![first, second];
    second_saved(&mut world);
    open_second_menu(&mut world);
    choose_properties(&mut world);
    properties_title(&mut world);
    let title = format!(
        "{} Properties###document-properties",
        world.document_name(second)
    );
    let (position, size) = window_geometry(&title).unwrap();
    let close =
        super::diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    assert!(window_geometry(&title).is_none());
    world.expect_active(first);
    assert!(world.document_is_open(second));
    world.close();
}
#[then("the second part remains open")]
fn second_open(world: &mut ApplicationWorld) {
    assert!(world.document_is_open(second(world)));
}
fn part_of(world: &ApplicationWorld) -> u64 {
    world.other_document.expect("expected the part")
}
fn drawing_of(world: &ApplicationWorld) -> u64 {
    world.previous_document.expect("expected the drawing")
}
#[then("the menu offers Arrange, Horizontal Tile, Vertical Tile and Restore All")]
fn arrangement_offered(world: &mut ApplicationWorld) {
    for entry in ["Arrange", "Horizontal Tile", "Vertical Tile", "Restore All"] {
        world.expect_text(entry);
    }
}
#[then("the menu offers Close Tabs and Close All")]
#[then("the menu offers separate Close Tabs and Close All items")]
fn list_offers_closes(world: &mut ApplicationWorld) {
    world.expect_text("Close Tabs");
    world.expect_text("Close All");
}
#[given("two part tabs are adjacent in the MCAD Software")]
#[given("two neighboring part tabs are visible in the MCAD Software")]
fn adjacent_tabs(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_part();
}
#[given("the left-hand part is active")]
fn left_hand_active(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.parts[0]);
}
#[then("the right-hand part becomes active")]
fn right_hand_active(world: &mut ApplicationWorld) {
    world.expect_active(world.parts[1]);
}
#[then("the left-hand part remains open")]
fn left_hand_open(world: &mut ApplicationWorld) {
    let name = world.document_name(world.parts[0]);
    world.expect_tab(&name);
}
#[given("the part on the right is active")]
fn right_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, Some(world.parts[1]));
}
#[when("I press Control+Shift+Tab")]
#[when("the user presses Control+Shift+Tab")]
fn control_shift_tab(world: &mut ApplicationWorld) {
    world.control_shift_key(ViewportPhysicalKey::Tab);
}
#[then("the part on the left becomes active")]
fn left_active(world: &mut ApplicationWorld) {
    world.expect_active(world.parts[0]);
}
#[then("the part on the right becomes active again")]
fn right_active_again(world: &mut ApplicationWorld) {
    world.expect_active(world.parts[1]);
}
#[then("its viewport and model browser are displayed")]
fn viewport_and_browser(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
    world.expect_blank_region([400, 600], [250, 300]);
}

#[given("one part document is active")]
fn one_part_active(world: &mut ApplicationWorld) {
    world.activate_through_tab(first(world));
    let id = world.active_id();
    assert!(
        world.parts.contains(&id),
        "active document {id}: expected an open part"
    );
    world.previous_document = Some(id);
}
#[then("another open part document becomes active")]
fn another_part_active(world: &mut ApplicationWorld) {
    let old = world
        .previous_document
        .expect("expected the previously active part");
    let current = world.active_id();
    assert_ne!(current, old);
    assert!(
        world.parts.contains(&current),
        "active document {current}: expected an open part"
    );
}
#[then("the previously active part remains open")]
fn previous_part_open(world: &mut ApplicationWorld) {
    let old = world
        .previous_document
        .expect("expected the previously active part");
    assert!(
        world.document_is_open(old),
        "previous document {old}: expected it open"
    );
}
