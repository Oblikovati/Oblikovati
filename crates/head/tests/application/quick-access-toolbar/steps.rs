// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use cucumber::{given, then, when};
mod material_browser;

const DEFAULT_ENTRIES: [&str; 6] = ["New", "Open", "Save", "Undo", "Redo", "Home"];
const CONTROL: &str = "...";
/// The row above the ribbon sits in the band's top row; below it, under the whole band.
const ROW_ABOVE_LIMIT: f32 = 58.;

fn open_customization(world: &mut ApplicationWorld) {
    world.click_text(CONTROL);
    world.pump(Vec::new());
}
/// A shortcut on the row, as distinct from the same word on the ribbon or in a menu.
fn on_row(world: &ApplicationWorld, label: &str) -> bool {
    world
        .frame
        .runs
        .iter()
        .any(|run| run.text == label && (run.center()[1] < ROW_ABOVE_LIMIT || below_row_holds(run)))
}
fn below_row_holds(run: &super::diagnostics::RenderedRun) -> bool {
    window_geometry("##quick-access-below").is_some_and(|(position, size)| {
        run.center()[1] >= position[1] && run.center()[1] <= position[1] + size[1]
    })
}
/// The row's own run: the ribbon names its commands the same way.
fn row_run<'a>(world: &'a ApplicationWorld, label: &str) -> &'a super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == label && (run.center()[1] < ROW_ABOVE_LIMIT || below_row_holds(run))
        })
        .unwrap_or_else(|| panic!("expected {label} on the row in {:?}", world.text()))
}
fn row_center(world: &ApplicationWorld, label: &str) -> [i32; 2] {
    row_run(world, label).center().map(|value| value as i32)
}
fn add_extrude(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    right_click_ribbon_extrude(world);
    world.click_text("Add to Quick Access Toolbar");
    world.pump(Vec::new());
}
fn right_click_ribbon_extrude(world: &mut ApplicationWorld) {
    let run = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Extrude" && run.center()[1] >= ROW_ABOVE_LIMIT)
        .cloned()
        .unwrap_or_else(|| panic!("expected Extrude on the ribbon in {:?}", world.text()));
    world.right_click(run.center().map(|value| value as i32));
    world.pump(Vec::new());
}
/// The only part open in these scenarios is the one that must stay active.
fn part_stays(world: &mut ApplicationWorld) {
    let part = world.previous_document.unwrap_or_else(|| world.active_id());
    world.expect_active(part);
    assert_eq!(world.state().documents.len(), 1, "expected one open part");
}
fn part_active(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
}
#[given("the quick access toolbar is above the ribbon")]
fn row_above(world: &mut ApplicationWorld) {
    let [_, y] = row_center(world, "Save");
    assert!(
        (y as f32) < ROW_ABOVE_LIMIT,
        "row at {y}: expected above the ribbon"
    );
}
#[given("Print is not present on the toolbar")]
fn print_absent(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Print"), "expected no Print on the row");
}
#[when("I open the quick access customization menu")]
#[when("I reopen the customization menu")]
fn open_menu(world: &mut ApplicationWorld) {
    open_customization(world);
}
#[then("Print is unchecked")]
fn print_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Print"), "expected Print unchecked");
}
#[given("New, Open, Save, Undo, Redo and Home are present on the toolbar")]
fn defaults_present(world: &mut ApplicationWorld) {
    for entry in DEFAULT_ENTRIES {
        assert!(on_row(world, entry), "expected {entry} on the toolbar");
    }
}
#[then("New, Open, Save, Undo, Redo and Home are checked")]
fn defaults_checked(world: &mut ApplicationWorld) {
    for entry in DEFAULT_ENTRIES {
        assert!(world.checked(entry), "expected {entry} checked");
    }
}
#[when("I select Print")]
#[when("I select Print again")]
fn select_print(world: &mut ApplicationWorld) {
    world.click_text("Print");
    world.pump(Vec::new());
}
#[then("a Print icon appears on the quick access toolbar")]
fn print_appears(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Print"), "expected Print on the row");
}
#[then("Print is checked")]
fn print_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Print"), "expected Print checked");
}
#[when("I choose Show Below the Ribbon")]
fn show_below(world: &mut ApplicationWorld) {
    world.click_text("Show Below the Ribbon");
    world.pump(Vec::new());
}
#[then("the Print icon is removed from the toolbar")]
fn print_removed(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Print"), "expected no Print on the row");
}
#[given("Print is visible on the toolbar")]
fn print_visible(world: &mut ApplicationWorld) {
    open_customization(world);
    select_print(world);
    assert!(on_row(world, "Print"), "expected Print on the row");
}
#[then("the quick access toolbar appears in a row below the ribbon")]
fn row_below(world: &mut ApplicationWorld) {
    let (position, _) = window_geometry("##quick-access-below").expect("expected the row below");
    let (_, ribbon) = window_geometry("##application-ribbon").expect("expected the ribbon");
    assert!(
        position[1] >= ribbon[1],
        "row at {position:?}: expected under the band"
    );
    let [_, y] = row_center(world, "Save");
    assert!(
        y as f32 >= ribbon[1],
        "Save at {y}: expected under the band"
    );
}
#[then("Print remains visible on the toolbar")]
fn print_remains(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Print"), "expected Print on the row");
}
#[then("Show Above the Ribbon is offered")]
fn above_offered(world: &mut ApplicationWorld) {
    world.expect_text("Show Above the Ribbon");
}
#[when("I choose Show Above the Ribbon")]
fn show_above(world: &mut ApplicationWorld) {
    world.click_text("Show Above the Ribbon");
    world.pump(Vec::new());
}
#[then("the quick access toolbar returns above the ribbon")]
fn row_returned(world: &mut ApplicationWorld) {
    assert!(
        window_geometry("##quick-access-below").is_none(),
        "expected no row below"
    );
    row_above(world);
}
// Ribbon command shortcuts
#[given("Extrude is available on the 3D Model ribbon")]
fn extrude_available(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.expect_text("Extrude");
}
#[given("Extrude is not on the quick access toolbar")]
fn extrude_absent(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Extrude"), "expected no Extrude on the row");
}
#[when("I right-click Extrude on the ribbon")]
#[when("I right-click Extrude on the ribbon again")]
fn right_click_extrude(world: &mut ApplicationWorld) {
    right_click_ribbon_extrude(world);
}
#[then("its context menu offers Add to Quick Access Toolbar")]
fn offers_add(world: &mut ApplicationWorld) {
    world.expect_text("Add to Quick Access Toolbar");
}
#[when("I choose Add to Quick Access Toolbar")]
fn choose_add(world: &mut ApplicationWorld) {
    world.click_text("Add to Quick Access Toolbar");
    world.pump(Vec::new());
}
#[then("an Extrude icon appears on the quick access toolbar")]
fn extrude_appears(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Extrude"), "expected Extrude on the row");
}
#[then("Extrude remains available on the ribbon")]
fn extrude_remains(world: &mut ApplicationWorld) {
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "Extrude" && run.center()[1] >= ROW_ABOVE_LIMIT),
        "expected Extrude on the ribbon"
    );
}
#[when("I right-click the added quick access icon")]
#[when("I right-click its quick access icon")]
fn right_click_shortcut(world: &mut ApplicationWorld) {
    let at = row_center(world, "Extrude");
    world.right_click(at);
    world.pump(Vec::new());
}
#[then("its context menu offers Remove from Quick Access Toolbar")]
fn offers_remove(world: &mut ApplicationWorld) {
    world.expect_text("Remove from Quick Access Toolbar");
}
#[when("I choose Remove from Quick Access Toolbar")]
fn choose_remove(world: &mut ApplicationWorld) {
    world.click_text("Remove from Quick Access Toolbar");
    world.pump(Vec::new());
}
#[then("the added icon disappears from the quick access toolbar")]
fn extrude_gone(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Extrude"), "expected no Extrude on the row");
}
#[given("Extrude has been added to the quick access toolbar")]
#[given("Extrude has been added to the quick access toolbar from 3D Model")]
fn extrude_added(world: &mut ApplicationWorld) {
    add_extrude(world);
    assert!(on_row(world, "Extrude"), "expected Extrude on the row");
}
#[then("the context menu offers Add Separator")]
fn offers_separator(world: &mut ApplicationWorld) {
    world.expect_text("Add Separator");
}
#[when("I choose Add Separator")]
fn choose_separator(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(row_gap_after_extrude(world));
    world.click_text("Add Separator");
    world.pump(Vec::new());
}
/// How far the row's control sits from Extrude: a divider takes room and draws in it.
fn row_gap_after_extrude(world: &ApplicationWorld) -> f32 {
    let extrude = row_run(world, "Extrude").maximum[0];
    let control = row_run(world, CONTROL).minimum[0];
    control - extrude
}
#[then("the quick access row gains a narrow divider beside the added command")]
fn divider_drawn(world: &mut ApplicationWorld) {
    let before = world.observed_row_gap.expect("expected the gap before");
    let after = row_gap_after_extrude(world);
    assert!(after > before, "gap {after}: expected more than {before}");
    let extrude = row_center(world, "Extrude");
    let run = row_run(world, "Extrude").clone();
    let x = run.maximum[0] as usize + 10;
    let lit = (0..14).any(|offset| {
        let at = [x + offset, extrude[1] as usize];
        let color = world.pixel_at(at);
        color[..3].iter().all(|channel| *channel > 120)
    });
    assert!(lit, "expected a drawn divider after Extrude");
}
// Invoking the shortcut
#[given("an empty part is active in the MCAD Software")]
fn empty_part(world: &mut ApplicationWorld) {
    part_active(world);
}
#[when("I select the Extrude quick access icon")]
fn select_shortcut(world: &mut ApplicationWorld) {
    let at = row_center(world, "Extrude");
    world.click(at);
    world.pump(Vec::new());
}
fn expect_planes(world: &mut ApplicationWorld) {
    for plane in ["XY Plane", "XZ Plane", "YZ Plane"] {
        let shown = world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == plane.replace(' ', ""))
            .count();
        assert_eq!(
            shown, 2,
            "{plane}: expected a label in the view and a browser row"
        );
    }
}
#[then("the workspace displays the origin planes for selection")]
#[then("the origin planes are displayed for selection")]
fn planes_shown(world: &mut ApplicationWorld) {
    expect_planes(world);
}
#[then("Start 2D Sketch is highlighted on the ribbon")]
fn start_highlighted(world: &mut ApplicationWorld) {
    let lit = world.pixel_left_of("Start 2D Sketch");
    let plain = world.pixel_left_of("Extrude");
    assert_ne!(
        lit, plain,
        "expected the button lit against a plain neighbour"
    );
}
#[then("the origin plane selection display closes")]
fn planes_closed(world: &mut ApplicationWorld) {
    let shown = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "XYPlane")
        .count();
    assert_eq!(shown, 1, "expected only the browser row of the XY plane");
}
#[then("the empty part remains active")]
fn empty_remains(world: &mut ApplicationWorld) {
    part_stays(world);
    assert_eq!(world.browser_sketch_count(), 0, "expected no new sketch");
}
#[then("the Extrude quick access icon remains visible")]
fn icon_visible(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Extrude"), "expected Extrude on the row");
}
#[then("the View ribbon remains displayed")]
fn view_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("User Interface");
}
// Document kinds
#[given("two parts and an assembly are open in the MCAD Software")]
fn parts_and_assembly(world: &mut ApplicationWorld) {
    let first = world.add_part();
    let second = world.add_part();
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    world.parts = vec![first, second];
    world.other_document = Some(assembly);
}
#[given("Extrude has been added to the quick access toolbar from its ribbon")]
fn extrude_from_ribbon(world: &mut ApplicationWorld) {
    add_extrude(world);
}
#[when("I activate the assembly")]
fn activate_assembly(world: &mut ApplicationWorld) {
    let assembly = world.other_document.expect("expected an assembly");
    world.click_tab(assembly);
    world.pump(Vec::new());
}
#[then("the Extrude shortcut is not shown on its quick access toolbar")]
fn not_on_assembly(world: &mut ApplicationWorld) {
    assert!(
        !on_row(world, "Extrude"),
        "expected no Extrude on the assembly"
    );
}
#[when("I reactivate the first part")]
fn reactivate_first(world: &mut ApplicationWorld) {
    let first = world.parts[0];
    world.click_tab(first);
    world.pump(Vec::new());
}
#[then("the Extrude shortcut is visible again")]
#[then("the Extrude shortcut is visible there as well")]
fn on_part(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Extrude"), "expected Extrude on the part");
}
#[when("I activate the second part")]
fn activate_second(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    world.click_tab(second);
    world.pump(Vec::new());
}
#[then("the second part remains active")]
fn second_active(world: &mut ApplicationWorld) {
    let second = world.parts[1];
    world.expect_active(second);
}
// Enablement of the ribbon's add option
#[then("Add to Quick Access Toolbar is enabled")]
fn add_enabled(world: &mut ApplicationWorld) {
    assert!(
        world.run_is_enabled("Add to Quick Access Toolbar"),
        "expected enabled"
    );
}
#[then("Add to Quick Access Toolbar is disabled")]
fn add_disabled(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("Add to Quick Access Toolbar"),
        "expected disabled"
    );
}
#[when("I dismiss the menu and remove the Extrude quick access icon")]
fn dismiss_and_remove(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    let at = row_center(world, "Extrude");
    world.right_click(at);
    world.pump(Vec::new());
    world.click_text("Remove from Quick Access Toolbar");
    world.pump(Vec::new());
}
// Standard shortcuts across document kinds
#[given("a part and an assembly are open in the MCAD Software")]
fn part_and_assembly(world: &mut ApplicationWorld) {
    let part = world.add_part();
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    world.parts = vec![part];
    world.other_document = Some(assembly);
}
#[given("Print is not selected in either document's quick access customization menu")]
fn print_unselected(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Print"), "expected no Print on the assembly");
    let part = world.parts[0];
    world.click_tab(part);
    world.pump(Vec::new());
    assert!(!on_row(world, "Print"), "expected no Print on the part");
    let assembly = world.other_document.expect("expected an assembly");
    world.click_tab(assembly);
    world.pump(Vec::new());
}
#[when("I activate the part and select Print in its quick access customization menu")]
fn select_print_on_part(world: &mut ApplicationWorld) {
    let part = world.parts[0];
    world.click_tab(part);
    world.pump(Vec::new());
    open_customization(world);
    select_print(world);
}
#[then("Print appears on the part's quick access toolbar")]
#[then("Print is visible on the part's quick access toolbar")]
fn print_on_part(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Print"), "expected Print on the part");
}
#[then("Print is absent from the assembly's quick access toolbar")]
fn print_off_assembly(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Print"), "expected no Print on the assembly");
}
#[when("I return to the part")]
fn return_to_part(world: &mut ApplicationWorld) {
    // The customization menu is still up from the step before; it takes the next click.
    world.escape();
    let part = world.parts[0];
    world.click_tab(part);
    world.pump(Vec::new());
}
#[then("Print is removed from the part's quick access toolbar")]
fn print_off_part(world: &mut ApplicationWorld) {
    assert!(!on_row(world, "Print"), "expected no Print on the part");
}
