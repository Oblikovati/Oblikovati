// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;
use oblikovati_model::application_prompts::{ApplicationPrompt, PromptFrequency};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportPhysicalKey};
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
fn select_second(world: &mut ApplicationWorld) {
    world.click_tab(second(world));
}
#[when("I select the first part's document tab")]
fn select_first(world: &mut ApplicationWorld) {
    world.click_tab(first(world));
}
#[then("the second part becomes active")]
fn second_is_active(world: &mut ApplicationWorld) {
    world.expect_active(second(world));
}
#[then("its rectangular profile appears in the viewport")]
fn rectangle_appears(world: &mut ApplicationWorld) {
    world.expect_drawn_region();
}
#[then("the model browser shows the second part")]
fn browser_second(world: &mut ApplicationWorld) {
    world.expect_browser_of(second(world));
    world.expect_browser_sketch();
}
#[then("the first part becomes active")]
fn first_is_active(world: &mut ApplicationWorld) {
    world.expect_active(first(world));
}
#[then("its model browser and viewport are displayed")]
fn browser_and_viewport(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[then("the second part remains available as an open tab")]
fn second_tab_remains(world: &mut ApplicationWorld) {
    let name = world.document_name(second(world));
    world.expect_tab(&name);
}
#[when("I open the document tab's context menu")]
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
fn toggle_tabs(world: &mut ApplicationWorld) {
    toggle_tabs_option(world);
}
#[given("document tabs are hidden")]
fn tabs_hidden(world: &mut ApplicationWorld) {
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
fn more_windows(world: &mut ApplicationWorld) {
    world.click_text("More Windows");
}
#[then("a Select Window dialog displays a scrollable list of open documents")]
fn select_window_list(world: &mut ApplicationWorld) {
    world.expect_text("Select Window");
    while world.parts.len() < OVERFLOWING_DOCUMENTS {
        world.add_part();
    }
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
fn ok_and_cancel(world: &mut ApplicationWorld) {
    world.expect_text("OK");
    world.expect_text("Cancel");
}
#[given("one part document is active")]
fn one_active(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.active_id());
}
#[when("I press Control+Tab")]
fn control_tab(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, true);
}
#[then("another open part document becomes active")]
fn another_active(world: &mut ApplicationWorld) {
    let before = world.previous_document.expect("expected a prior document");
    let now = world.active_id();
    assert_ne!(
        now, before,
        "Control+Tab: expected a different active document"
    );
    world.expect_active(now);
}
#[then("the previously active part remains open")]
fn previous_open(world: &mut ApplicationWorld) {
    let before = world.previous_document.expect("expected a prior document");
    assert!(world.document_is_open(before));
    let name = world.document_name(before);
    world.expect_tab(&name);
}
#[given("the second part has been saved")]
#[given("the second part is saved and has no unsaved changes")]
fn second_saved(world: &mut ApplicationWorld) {
    let id = second(world);
    world.save_part_to_file(id, "second part.opd");
    world.activate_through_tab(first(world));
    assert!(!world.state().document_storage[&id].unsaved_edits);
}
#[when("I open the second part's document tab context menu")]
fn open_second_menu(world: &mut ApplicationWorld) {
    world.open_tab_menu(second(world));
}
#[then("the first part remains active in the workspace")]
fn first_remains(world: &mut ApplicationWorld) {
    world.expect_active(first(world));
}
#[when("I choose Properties from the context menu")]
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
fn close_properties(world: &mut ApplicationWorld) {
    world.click_text("Close");
    world.expect_no_text("General");
}
#[then("the second part remains open")]
fn second_open(world: &mut ApplicationWorld) {
    assert!(world.document_is_open(second(world)));
}
#[when("I click the close button on the second part's document tab")]
fn close_second_mark(world: &mut ApplicationWorld) {
    world.click_close_mark(second(world));
}
fn expect_second_gone_without_prompt(world: &mut ApplicationWorld) {
    let id = second(world);
    assert!(
        !world.document_is_open(id),
        "expected the second part closed"
    );
    world.expect_no_text("before closing");
}
#[then("the second part's tab disappears")]
fn second_tab_gone(world: &mut ApplicationWorld) {
    expect_second_gone_without_prompt(world);
}
#[then("no save prompt is displayed")]
fn no_prompt(world: &mut ApplicationWorld) {
    world.expect_no_text("before closing");
}
#[given("other documents are also open")]
fn others_open(world: &mut ApplicationWorld) {
    // Opened first, so the active part is neither the first nor the last tab that closes back to it.
    let active = world.active_id();
    world.add_part();
    world.activate_through_tab(active);
    world.previous_document = Some(active);
}
#[when("I create a new empty part from a standard metric template")]
fn create_from_template(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric");
    world.click_text("Standard metric part");
    world.click_text("Create");
}
#[then("the new part becomes active and has its own document tab")]
fn new_part_active(world: &mut ApplicationWorld) {
    let id = world.active_id();
    assert_ne!(Some(id), world.previous_document);
    world.closed_document = Some(id);
    let name = world.document_name(id);
    world.expect_tab(&name);
}
#[when("I click the close button on the new part's tab without editing it")]
#[when("I click the close button on the new part's document tab")]
#[when("I click the close button on the part's document tab")]
fn close_active_mark(world: &mut ApplicationWorld) {
    world.closed_document = Some(world.active_id());
    world.close_active_tab();
}
#[then("the new part's tab disappears without a save prompt")]
fn new_part_gone(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected a closed document");
    assert!(!world.document_is_open(id), "expected the new part closed");
    world.expect_no_text("before closing");
}
#[then("the previously active part is displayed again")]
#[then("the previously active part is displayed")]
fn previous_again(world: &mut ApplicationWorld) {
    let id = world.previous_document.expect("expected a prior document");
    world.expect_active(id);
}
#[then("its model browser and viewport are restored")]
fn restored(world: &mut ApplicationWorld) {
    let id = world.active_id();
    world.expect_browser_of(id);
}
#[given("a new part is open in the MCAD Software")]
fn new_part_open(world: &mut ApplicationWorld) {
    world.add_part();
}
#[given("I am editing a sketch in the part")]
#[given("I am editing that sketch")]
fn editing_sketch(world: &mut ApplicationWorld) {
    if world.state().authoring != GuiAuthoringContext::Sketch {
        world.start_sketch();
    }
}
#[given("the environment-exit message is enabled")]
fn exit_enabled(world: &mut ApplicationWorld) {
    world
        .app()
        .owner
        .set_prompt_frequency_from_gui(
            ApplicationPrompt::ExitEnvironment,
            PromptFrequency::AlwaysShow,
        )
        .unwrap();
}
#[then("a dialog asks whether to exit the sketch environment and continue")]
fn exit_asked(world: &mut ApplicationWorld) {
    world.expect_text(EXIT_QUESTION);
}
#[then("the part remains open")]
fn part_open(world: &mut ApplicationWorld) {
    // The part a close asked about, else the part of a drawing and part pair.
    let id = world
        .closed_document
        .or(world.other_document)
        .expect("expected the part");
    world.expect_no_text(EXIT_QUESTION);
    world.expect_no_text("before closing");
    assert!(world.document_is_open(id));
}
#[then("the sketch editing environment remains active")]
fn sketch_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    world.expect_text("Finish Sketch");
}
#[given("a new part contains an unsaved sketch in the MCAD Software")]
fn part_with_open_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
}
#[when("I choose OK to exit the sketch environment and continue")]
fn exit_ok(world: &mut ApplicationWorld) {
    world.click_text("OK");
}
#[then("a save-changes prompt identifies the part")]
fn prompt_names_part(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    let name = world.document_name(id);
    world.expect_text(&format!("Save changes to {name}"));
}
#[then("the prompt offers Yes, No and Cancel")]
fn yes_no_cancel(world: &mut ApplicationWorld) {
    for label in ["Yes", "No", "Cancel"] {
        world.expect_text(label);
    }
}
#[then("its sketch remains listed in the model browser")]
#[then("its sketch is listed in the model browser")]
fn sketch_listed(world: &mut ApplicationWorld) {
    world.expect_browser_sketch();
}
#[then("the part modeling environment is active")]
fn modeling_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    world.expect_text("Sketch XY");
}
#[given("a new part with an unsaved sketch is active in the MCAD Software")]
fn new_part_with_sketch(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_finished_sketch();
}
#[given("the part is in the modeling environment")]
fn in_modeling(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
}
#[then("a Save As dialog opens with a suggested file name")]
fn suggested_name(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    let name = format!("{}.opd", world.document_name(id));
    world.expect_text("Save in");
    assert!(
        world.text_is_selected(&name),
        "file name {name:?}: expected a selected proposal"
    );
}
#[when("I cancel the Save As dialog")]
fn cancel_save_as(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
}
#[then("the part remains open and active")]
fn open_and_active(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected the part");
    world.expect_no_text("Save in");
    world.expect_active(id);
}
#[given("another part was active before the new part was created")]
fn previous_part(world: &mut ApplicationWorld) {
    // Given order puts the new part first; a second open part keeps both facts true.
    let new_part = world.active_id();
    let other = world.add_part();
    world.activate_through_tab(new_part);
    world.previous_document = Some(other);
}
#[when("I accept the suggested file name in the Save As dialog")]
fn accept_suggested(world: &mut ApplicationWorld) {
    world.accept_save_as();
}
#[then("the part file exists at the selected location")]
fn file_exists(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved file");
    assert!(path.exists(), "saved file {path:?}: expected it on disk");
}
#[then("the new part's tab disappears")]
fn new_tab_gone(world: &mut ApplicationWorld) {
    let id = world.closed_document.expect("expected a closed document");
    assert!(
        !world.document_is_open(id),
        "document {id}: expected it closed"
    );
}
#[when("I choose Close")]
fn choose_close(world: &mut ApplicationWorld) {
    world.click_text("Close");
}
fn part_of(world: &ApplicationWorld) -> u64 {
    world.other_document.expect("expected the part")
}
fn drawing_of(world: &ApplicationWorld) -> u64 {
    world.previous_document.expect("expected the drawing")
}
#[given("the part has an active sketch editing environment")]
fn part_in_sketch(world: &mut ApplicationWorld) {
    world.activate_through_tab(part_of(world));
    world.start_sketch();
}
#[given("the drawing is active with its drawing resources branch collapsed")]
fn drawing_active_collapsed(world: &mut ApplicationWorld) {
    world.activate_through_tab(drawing_of(world));
    world.press_disclosure("Drawing Resources");
    world.expect_no_text("Sheet Formats");
}
#[when("I select the part document tab")]
#[when("I select the part document tab again")]
fn select_part_tab(world: &mut ApplicationWorld) {
    world.click_tab(part_of(world));
}
#[then("the part sketch viewport and model browser are displayed")]
fn part_sketch_shown(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
    let centre = world.pixel_at([700, 450]);
    assert!(
        centre[..3].iter().all(|channel| *channel < 150),
        "viewport {centre:?}: expected no sheet"
    );
}
#[then("the sketch ribbon is active")]
fn sketch_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Finish Sketch");
}
#[when("I select the drawing document tab")]
fn select_drawing_tab(world: &mut ApplicationWorld) {
    world.click_tab(drawing_of(world));
}
fn expect_drawing_shown(world: &mut ApplicationWorld) {
    let sheet = world.pixel_at([1000, 500]);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "sheet {sheet:?}"
    );
    world.expect_text("Drawing Resources");
}
#[then("the drawing sheet and drawing browser are displayed")]
#[then("its sheet and drawing browser are displayed")]
fn drawing_shown(world: &mut ApplicationWorld) {
    expect_drawing_shown(world);
}
#[then("the drawing ribbon is active")]
fn drawing_ribbon_active(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.expect_no_text("Finish Sketch");
}
#[then("the part remains available as an open tab")]
fn part_tab_remains(world: &mut ApplicationWorld) {
    let name = world.document_name(part_of(world));
    world.expect_tab(&name);
}
#[given("the part is active")]
fn part_active(world: &mut ApplicationWorld) {
    world.activate_through_tab(part_of(world));
}
#[when("I open the document list at the end of the tab strip")]
fn open_document_list(world: &mut ApplicationWorld) {
    world.click_text("Documents");
    world.pump(Vec::new());
}
#[then("the list contains the open part and drawing")]
fn list_contains(world: &mut ApplicationWorld) {
    for id in [part_of(world), drawing_of(world)] {
        let name = world.document_name(id);
        world.expect_text(&name);
    }
}
#[then("the active part is marked in the list")]
fn active_marked(world: &mut ApplicationWorld) {
    let name = world.document_name(part_of(world));
    assert!(world.checked(&name), "list row {name:?}: expected a mark");
}
#[when("I select the drawing from the list")]
fn select_drawing_row(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    world.click_text(&name);
}
#[then("the drawing becomes active")]
fn drawing_is_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, Some(drawing_of(world)));
}
#[then("the menu offers Arrange, Horizontal Tile, Vertical Tile and Restore All")]
fn arrangement_offered(world: &mut ApplicationWorld) {
    for entry in ["Arrange", "Horizontal Tile", "Vertical Tile", "Restore All"] {
        world.expect_text(entry);
    }
}
#[then("the menu offers Close Tabs and Close All")]
fn list_offers_closes(world: &mut ApplicationWorld) {
    world.expect_text("Close Tabs");
    world.expect_text("Close All");
}
const NEIGHBOR_PARTS: usize = 3;
fn open_parts_then_drawing(world: &mut ApplicationWorld) {
    for _ in 0..NEIGHBOR_PARTS {
        world.add_part();
    }
    world.previous_document = Some(world.add_drawing());
}
fn tab_x(world: &mut ApplicationWorld, id: u64) -> f32 {
    let name = world.document_name(id);
    world.expect_tab(&name).center()[0]
}
fn drag_tab_to_first_place(world: &mut ApplicationWorld, id: u64) {
    world.dragged_tab_x = Some(tab_x(world, id));
    let name = world.document_name(id);
    let from = world.expect_tab(&name).center().map(|value| value as i32);
    let first = tab_x(world, world.parts[0]) as i32;
    world.drag(from, [first - 30, from[1]]);
    world.pump(Vec::new());
}
fn expect_tab_moved_left(world: &mut ApplicationWorld, id: u64) {
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, id);
    assert!(
        now < before - 100.,
        "tab at {now}: expected well left of {before}"
    );
    let first_part = tab_x(world, world.parts[0]);
    assert!(
        now < first_part,
        "tab at {now}: expected before the first part at {first_part}"
    );
}
#[given("a drawing and several other documents are open in the MCAD Software")]
#[given("a drawing and several parts are open in the MCAD Software")]
fn parts_and_drawing(world: &mut ApplicationWorld) {
    open_parts_then_drawing(world);
}
#[given("the drawing is active and its tab is visible")]
#[given("the drawing is active")]
fn drawing_tab_visible(world: &mut ApplicationWorld) {
    let drawing = drawing_of(world);
    assert_eq!(world.state().document, Some(drawing));
    tab_x(world, drawing);
}
#[when("I drag the drawing tab left across several neighboring tabs")]
fn drag_drawing_tab_left(world: &mut ApplicationWorld) {
    drag_tab_to_first_place(world, drawing_of(world));
}
#[then("the drawing tab appears at its new position among the document tabs")]
fn drawing_tab_moved(world: &mut ApplicationWorld) {
    expect_tab_moved_left(world, drawing_of(world));
}
#[then("its sheet and browser remain displayed")]
fn sheet_and_browser(world: &mut ApplicationWorld) {
    expect_drawing_shown(world);
}
#[when("I drag the drawing tab back to its earlier position")]
fn drag_drawing_tab_back(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let from = world.expect_tab(&name).center().map(|value| value as i32);
    let target = world.dragged_tab_x.expect("expected a starting position") as i32;
    world.drag(from, [target + 30, from[1]]);
    world.pump(Vec::new());
}
#[then("the drawing tab returns to its earlier position")]
fn drawing_tab_back(world: &mut ApplicationWorld) {
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, drawing_of(world));
    assert!((now - before).abs() < 6., "tab at {now}: expected {before}");
}
#[given("an inactive part tab is visible")]
fn inactive_part_visible(world: &mut ApplicationWorld) {
    let part = world.parts[NEIGHBOR_PARTS - 1];
    tab_x(world, part);
}
#[when("I drag the inactive part tab left across several neighboring tabs")]
fn drag_part_tab_left(world: &mut ApplicationWorld) {
    drag_tab_to_first_place(world, world.parts[NEIGHBOR_PARTS - 1]);
}
#[then("the part tab appears at its new position among the document tabs")]
fn part_tab_moved(world: &mut ApplicationWorld) {
    let part = world.parts[NEIGHBOR_PARTS - 1];
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, part);
    assert!(
        now < before - 60.,
        "tab at {now}: expected well left of {before}"
    );
    let first_part = tab_x(world, world.parts[0]);
    assert!(
        now < first_part,
        "tab at {now}: expected before the first part at {first_part}"
    );
}
#[then("the part viewport and model browser are displayed")]
fn dragged_part_shown(world: &mut ApplicationWorld) {
    world.expect_active(world.parts[NEIGHBOR_PARTS - 1]);
    world.expect_blank_region(super::documents::VIEWPORT_X, super::documents::VIEWPORT_Y);
}
#[then("the drawing remains open")]
fn drawing_open(world: &mut ApplicationWorld) {
    assert!(world.document_is_open(drawing_of(world)));
}
#[given("the drawing is active in the main document tab strip")]
fn drawing_in_main_strip(world: &mut ApplicationWorld) {
    let drawing = drawing_of(world);
    assert_eq!(world.state().document, Some(drawing));
    tab_x(world, drawing);
}
#[when("I drag the drawing tab from the strip into the workspace")]
fn drag_drawing_out(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let from = world.expect_tab(&name).center().map(|value| value as i32);
    world.drag(from, [800, 500]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the drawing is displayed in a separate window with its own document tab")]
fn drawing_detached(world: &mut ApplicationWorld) {
    expect_drawing_window(world);
}
#[then("the drawing sheet remains visible in that window")]
fn sheet_in_window(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let title = world.run(&name).center();
    let at = [title[0] as usize + 200, title[1] as usize + 200];
    let sheet = world.pixel_at(at);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "pixel {at:?} {sheet:?}"
    );
}
#[then("the part remains open in the main workspace")]
fn part_in_main(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I select another open part tab in the main workspace")]
#[when("I select the part tab in the main workspace")]
fn select_main_part_tab(world: &mut ApplicationWorld) {
    world.click_tab(part_of(world));
}
#[then("that part becomes active in the main workspace")]
fn that_part_active(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
fn expect_drawing_window(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    assert!(
        world.tab_run(&name).is_none(),
        "tab {name:?}: expected it out of the strip"
    );
    let labels = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == name)
        .count();
    assert!(
        labels >= 2,
        "expected the window title and its tab to read {name:?}"
    );
}
#[then("the drawing remains open in its separate window")]
#[then("the detached drawing remains open")]
#[given("the drawing has a document tab in that window")]
fn drawing_window_remains(world: &mut ApplicationWorld) {
    expect_drawing_window(world);
}
#[given("a drawing is open in a separate window in the MCAD Software")]
fn drawing_in_window(world: &mut ApplicationWorld) {
    world.other_document = Some(world.add_part());
    world.previous_document = Some(world.add_drawing());
    drag_drawing_out(world);
}
#[when("I open the detached drawing tab context menu")]
fn open_detached_menu(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let center = world.run(&name).center().map(|value| value as i32);
    world.right_click(center);
    world.pump(Vec::new());
}
#[given("a part is open in the main workspace")]
#[given("the part context is active in the main workspace")]
fn part_context(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I click the detached drawing sheet")]
fn click_detached_sheet(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let title = world.run(&name).center();
    world.click([title[0] as i32 + 200, title[1] as i32 + 200]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the main application browser displays the drawing tree")]
fn main_drawing_tree(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_no_text("Origin");
}
#[then("the main application ribbon displays drawing-context tabs")]
fn main_drawing_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Place Views");
}
#[then("the main application browser displays the part tree")]
fn main_part_tree(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_no_text("Drawing Resources");
}
#[then("the main application ribbon displays part-context tabs")]
fn main_part_tabs(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
}
#[given("two part tabs are adjacent in the MCAD Software")]
fn adjacent_tabs(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_part();
}
#[given("the part on the right is active")]
fn right_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, Some(world.parts[1]));
}
#[when("I press Control+Shift+Tab")]
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
}
