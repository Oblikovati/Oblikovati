// SPDX-License-Identifier: GPL-2.0-only
//! The Home tab, tab previews and a strip too narrow for its tabs.
use super::super::ApplicationWorld;
use super::super::diagnostics::window_geometry;
use cucumber::{given, then, when};
#[path = "preview_probe.rs"]
mod preview_probe;

const PREVIEW: &str = "##tab-preview";
/// Enough open documents to overflow the strip at the scenario window's width.
const MANY_DOCUMENTS: usize = 16;
/// Part2: a tab is found by the start of its label, and Part1 also starts Part10 and on.
const HIDDEN_PART: usize = 1;

fn open_switch(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.click_text("Switch");
    world.pump(Vec::new());
}
fn home_tab_center(world: &ApplicationWorld) -> [i32; 2] {
    world.expect_tab("Home").center().map(|value| value as i32)
}
fn expect_home_shown(world: &ApplicationWorld) {
    assert!(
        window_geometry("##home-workspace").is_some(),
        "expected the Home workspace in {:?}",
        world.text()
    );
    world.expect_text("Recent");
}
fn expect_home_hidden() {
    assert!(
        window_geometry("##home-workspace").is_none(),
        "expected no Home workspace"
    );
}
fn expect_preview_above(world: &mut ApplicationWorld, label: &str) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let (position, size) = window_geometry(PREVIEW).expect("expected a tab preview");
    let tab = world.expect_tab(label);
    assert!(
        position[1] + size[1] <= tab.minimum[1],
        "preview bottom {}: expected above the tab at {}",
        position[1] + size[1],
        tab.minimum[1]
    );
    let image_top = preview_label_bottom(world, label, position, size) + 4.;
    assert!(
        preview_probe::picture_has_detail(&capture, position, size, image_top),
        "expected varied pixels inside the preview picture"
    );
    world.last_capture = Some(capture);
}

fn preview_label_bottom(
    world: &ApplicationWorld,
    label: &str,
    position: [f32; 2],
    size: [f32; 2],
) -> f32 {
    world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| {
            run.text.starts_with(label)
                && (0..2).all(|axis| {
                    run.minimum[axis] >= position[axis]
                        && run.maximum[axis] <= position[axis] + size[axis]
                })
        })
        .unwrap_or_else(|| panic!("preview label {label:?}: expected a row inside the preview"))
        .maximum[1]
}

fn preview_text(world: &ApplicationWorld) -> String {
    world.window_text(PREVIEW)
}
fn start_sketch_on(world: &mut ApplicationWorld, part: u64) {
    world.activate_through_tab(part);
    world.start_sketch();
}
fn expect_sketch_environment(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch,
        "expected the sketch environment"
    );
    expect_home_hidden();
}
// Switch menu
#[given("several documents are open in the MCAD Software")]
fn several_open(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_part();
    let active = world.add_part();
    world.previous_document = Some(active);
}
#[given("a part is displayed in the workspace")]
fn part_displayed(world: &mut ApplicationWorld) {
    let part = world.active_id();
    world.expect_browser_of(part);
}
fn choose_more(world: &mut ApplicationWorld) {
    open_switch(world);
    world.click_text("More Windows");
    world.pump(Vec::new());
}
fn highlight_other(world: &mut ApplicationWorld) {
    let other = world.parts[0];
    let name = world.document_name(other);
    let row = world.run(&name).center().map(|value| value as i32);
    world.click(row);
}
#[given("the Select Window dialog is open with another part highlighted")]
fn dialog_with_highlight(world: &mut ApplicationWorld) {
    choose_more(world);
    highlight_other(world);
}
#[then("the Select Window dialog closes")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Select Window");
}
#[then("the original part remains displayed in the workspace")]
fn original_displayed(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_active(part);
}
#[given("two parts are open in the MCAD Software")]
fn two_parts(world: &mut ApplicationWorld) {
    world.add_part();
    world.add_part();
}
#[given("the first part is displayed in the workspace")]
fn first_displayed(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    world.activate_through_tab(first);
    world.expect_browser_of(first);
}
#[given("the second part has an active sketch editing environment")]
fn second_in_sketch(world: &mut ApplicationWorld) {
    let [first, second] = [world.parts[0], world.parts[1]];
    start_sketch_on(world, second);
    world.activate_through_tab(first);
}
#[when("I select the second part from the menu")]
fn select_second_row(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    let name = world.document_name(second);
    world.click_text(&name);
    world.pump(Vec::new());
}
#[then("its sketch viewport and model browser are displayed")]
fn sketch_and_browser(world: &mut ApplicationWorld) {
    let part = world.active_id();
    world.expect_browser_of(part);
    expect_sketch_environment(world);
}
#[then("its sketch editing environment remains active")]
fn sketch_remains(world: &mut ApplicationWorld) {
    expect_sketch_environment(world);
}
#[then("the first part remains open")]
fn first_open(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    assert!(
        world.document_is_open(first),
        "expected the first part open"
    );
}
// Home with a sketch
#[given("a part is open in the MCAD Software")]
fn part_open(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
    world.other_document = Some(part);
}
#[when("I choose Home from the View tab's Switch menu")]
#[when("I choose Home from the View ribbon Switch menu")]
fn home_from_switch(world: &mut ApplicationWorld) {
    open_switch(world);
    world.click_text("Home");
    world.pump(Vec::new());
}
#[then("the Home workspace displays its recent documents")]
#[then("the Home workspace and its recent documents are displayed")]
#[then("the Home workspace becomes active")]
#[then("the Home workspace becomes active again")]
fn home_displayed(world: &mut ApplicationWorld) {
    expect_home_shown(world);
}
#[then("the part remains available in the document tab strip")]
#[then("the part document remains available as an open tab")]
fn part_in_strip(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    let name = world.document_name(part);
    world.expect_tab(&name);
}
#[when("I select the part's document tab")]
#[when("I select the remaining part tab")]
fn select_part_tab(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.click_tab(part);
    world.pump(Vec::new());
}
#[then("the part's sketch viewport and model browser are displayed")]
#[then("the original sketch viewport and model browser are displayed")]
#[then("the part window's sketch viewport and model browser are displayed")]
fn part_sketch_and_browser(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_browser_of(part);
    expect_sketch_environment(world);
    world.expect_blank_region([400, 600], [250, 300]);
}
// Previews
#[given("the first part is active and the second has an empty sketch visible")]
fn first_active_second_sketched(world: &mut ApplicationWorld) {
    let [first, second] = [world.parts[0], world.parts[1]];
    start_sketch_on(world, second);
    world.pump(Vec::new());
    world.activate_through_tab(first);
    world.previous_document = Some(first);
}
fn hover_tab(world: &mut ApplicationWorld, id: u64) {
    let name = world.document_name(id);
    let at = world.expect_tab(&name).center().map(|value| value as i32);
    world.hover(at);
    world.pump(Vec::new());
}
#[when("I hover over the second part's document tab")]
#[when("the user hovers over the second part's document tab")]
fn hover_second(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    hover_tab(world, second);
}
#[then("a thumbnail of the second part's sketch appears above the tab")]
#[then("a thumbnail of the second part appears above the tab")]
fn thumbnail_second(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    let name = world.document_name(second);
    expect_preview_above(world, &name);
}
#[then("the second part's name appears above the thumbnail")]
fn name_above(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    let name = world.document_name(second);
    assert!(
        preview_text(world).contains(&name),
        "expected {name} in the preview"
    );
}
#[then("the first part remains active")]
fn first_active_still(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    world.expect_active(first);
}
#[when("I move the pointer away from the tab")]
#[when("the user moves the pointer away from the tab")]
fn move_away(world: &mut ApplicationWorld) {
    world.hover([900, 500]);
    world.pump(Vec::new());
}
#[then("the thumbnail and its name disappear")]
fn thumbnail_gone(_: &mut ApplicationWorld) {
    assert!(
        window_geometry(PREVIEW).is_none(),
        "expected no tab preview"
    );
}
#[given("the Home workspace is available in the document tab strip")]
fn home_in_strip(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
    // Home has a picture only once it has been in front.
    world.click(home_tab_center(world));
    world.pump(Vec::new());
    let part = world.previous_document.expect("expected a remembered part");
    world.click_tab(part);
    world.pump(Vec::new());
    world.expect_active(part);
}
#[when("I hover over the Home tab")]
#[when("the user hovers over the Home tab")]
fn hover_home(world: &mut ApplicationWorld) {
    world.hover(home_tab_center(world));
    world.pump(Vec::new());
}
#[then("a thumbnail of the Home workspace appears above the tab")]
fn thumbnail_home(world: &mut ApplicationWorld) {
    expect_preview_above(world, "Home");
}
#[then("a Home label appears above the thumbnail")]
fn home_label(world: &mut ApplicationWorld) {
    assert!(
        preview_text(world).contains("Home"),
        "expected Home in the preview"
    );
}
#[then("the part remains active in the workspace")]
fn part_active_workspace(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_active(part);
    expect_home_hidden();
}
#[given("the second part has been saved to a file")]
fn second_saved(world: &mut ApplicationWorld) {
    let [first, second] = [world.parts[0], world.parts[1]];
    world.saved_path = Some(world.save_part_to_file(second, "Saved.opd"));
    world.activate_through_tab(second);
    world.pump(Vec::new());
    world.activate_through_tab(first);
}
#[given("the part is being edited in a sketch")]
#[given("a sketch is being edited in the part")]
fn part_in_edit(world: &mut ApplicationWorld) {
    world.start_sketch();
    world.pump(Vec::new());
}
#[when("I hover over the active part's document tab")]
#[when("the user hovers over the active part's document tab")]
fn hover_active(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    hover_tab(world, part);
}
#[then("a thumbnail of its sketch viewport appears above the tab")]
fn thumbnail_active(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    let name = world.document_name(part);
    expect_preview_above(world, &name);
}
#[then("the preview label displays the saved file's full path and part name")]
fn preview_path_name(world: &mut ApplicationWorld) {
    let path = world.saved_path.clone().expect("expected a saved path");
    let part = world.previous_document.expect("expected a remembered part");
    let name = world.document_name(part);
    let text = preview_text(world);
    let wanted = path.display().to_string().replace(' ', "");
    assert!(
        text.contains(&wanted) && text.contains(&name),
        "preview {text:?}"
    );
}
// Home tab menu
#[given("documents are open in the tab strip")]
fn documents_in_strip(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
    assert!(!world.parts.is_empty(), "expected open document tabs");
    for part in world.parts.clone() {
        let name = world.document_name(part);
        world.expect_tab(&name);
    }
}
#[when("I right-click the Home tab")]
#[when("the user right-clicks the Home tab")]
fn right_click_home(world: &mut ApplicationWorld) {
    world.right_click(home_tab_center(world));
    world.pump(Vec::new());
}
#[then("the context menu offers Close Tabs and Close All")]
#[then("its context menu offers Close Tabs and Close All")]
#[then("the context menu offers separate Close Tabs and Close All items")]
#[then("its context menu offers separate Close Tabs and Close All items")]
fn home_menu(world: &mut ApplicationWorld) {
    world.expect_text("Close Tabs");
    world.expect_text("Close All");
}
#[then("the context menu closes")]
fn home_menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Close Tabs");
    world.expect_no_text("Undock Ribbon");
}
#[then("the Home workspace remains active")]
fn home_remains(world: &mut ApplicationWorld) {
    expect_home_shown(world);
}
#[then("the open documents remain available in the tab strip")]
fn documents_remain(world: &mut ApplicationWorld) {
    documents_in_strip(world);
}
#[given("the Home tab is visible but inactive")]
fn home_inactive(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
    expect_home_hidden();
}
#[then("the part's sketch remains displayed and active")]
#[then("the part's sketch remains active")]
fn part_sketch_active(world: &mut ApplicationWorld) {
    expect_sketch_environment(world);
}
// Overflow
#[given("many documents are open in the MCAD Software")]
fn many_open(world: &mut ApplicationWorld) {
    for _ in 0..MANY_DOCUMENTS {
        world.add_part();
    }
}
#[given("the tab strip overflows its available width")]
fn strip_overflows(world: &mut ApplicationWorld) {
    world.expect_text("<");
}
#[given("a presentation is active while earlier tabs are out of view")]
fn presentation_active(world: &mut ApplicationWorld) {
    let presentation = world.add_template(oblikovati_gui::GuiTemplateKind::Presentation);
    world.previous_document = Some(presentation);
    world.pump(Vec::new());
    assert!(world.tab_run("Home").is_none(), "expected Home out of view");
}
#[when("I click the left navigation arrow at the end of the tab strip")]
fn click_left_arrow(world: &mut ApplicationWorld) {
    for _ in 0..=MANY_DOCUMENTS {
        if world.tab_run("Home").is_some() {
            break;
        }
        world.click_text("<");
        world.pump(Vec::new());
    }
}
#[then("earlier tabs, including Home, become visible")]
fn earlier_visible(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
    let earlier = world.parts[HIDDEN_PART];
    let name = world.document_name(earlier);
    world.expect_tab(&name);
}
#[then("the presentation remains active in the workspace")]
fn presentation_stays(world: &mut ApplicationWorld) {
    let presentation = world.previous_document.expect("expected a presentation");
    assert_eq!(
        world.active_id(),
        presentation,
        "expected the presentation in front"
    );
}
#[given("an open part's tab is outside the visible portion of the strip")]
fn part_outside(world: &mut ApplicationWorld) {
    let hidden = world.parts[HIDDEN_PART];
    let name = world.document_name(hidden);
    assert!(
        world.tab_run(&name).is_none(),
        "expected {name} out of view"
    );
}
#[when("I select that part from the View tab's Switch menu")]
fn select_from_switch(world: &mut ApplicationWorld) {
    let hidden = world.parts[HIDDEN_PART];
    let name = world.document_name(hidden);
    open_switch(world);
    world.click_text(&name);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the tab strip shifts to show its active tab")]
fn strip_shows_active(world: &mut ApplicationWorld) {
    let hidden = world.parts[HIDDEN_PART];
    world.expect_active(hidden);
    let name = world.document_name(hidden);
    world.expect_tab(&name);
}
#[then("the part's viewport and model browser are displayed")]
fn part_viewport_browser(world: &mut ApplicationWorld) {
    let hidden = world.parts[HIDDEN_PART];
    world.expect_browser_of(hidden);
}
// Keyboard ring with Home
#[given("Home and a part window are adjacent in the MCAD Software tab strip")]
fn home_adjacent(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
    world.expect_tab("Home");
}
#[given("the part window is active with a sketch being edited")]
fn window_with_sketch(world: &mut ApplicationWorld) {
    world.start_sketch();
    world.pump(Vec::new());
}
#[then("the part window remains open")]
fn window_open(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    let name = world.document_name(part);
    world.expect_tab(&name);
}
