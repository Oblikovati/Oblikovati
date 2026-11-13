// SPDX-License-Identifier: GPL-2.0-only
use super::*;

// Open options
#[given("a recent part with a Front design view is displayed in List View")]
fn front_view_part(world: &mut ApplicationWorld) {
    let part = world.parts[0];
    world
        .app()
        .owner
        .document_commands(part)
        .create_design_view("Front")
        .unwrap();
    world.save_part_to_file(part, &format!("{SEARCHED}.opd"));
    world.pump(Vec::new());
    assert!(home_shows(world, SEARCHED), "expected the part on the list");
}
#[when("I choose Open with options")]
fn choose_open_options(world: &mut ApplicationWorld) {
    world.click_text("Open with options");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a File Open Options dialog is displayed")]
fn options_shown(world: &mut ApplicationWorld) {
    world.expect_text("File Open Options");
}
#[then("Model State shows the primary state")]
fn primary_state(world: &mut ApplicationWorld) {
    world.expect_text("Model State: Primary");
}
#[then("Design View shows Last Active")]
fn last_active(world: &mut ApplicationWorld) {
    world.expect_text("Design View: Last Active");
}
#[when("I open the Design View list")]
#[when("the user opens the Design View list")]
fn open_view_list(world: &mut ApplicationWorld) {
    world.click_text("Design View");
    world.pump(Vec::new());
}
#[then("the list includes the primary view and the Front view")]
fn view_list_rows(world: &mut ApplicationWorld) {
    world.expect_text("Primary");
    world.expect_text("Front");
}
#[when("I choose Front")]
fn choose_front(world: &mut ApplicationWorld) {
    world.click_text("Front");
    world.pump(Vec::new());
}
#[then("Design View shows Front")]
fn shows_front(world: &mut ApplicationWorld) {
    world.expect_text("Design View: Front");
}
#[then("the dialog closes and the Recent list is visible")]
fn dialog_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("File Open Options");
    assert!(home_shows(world, SEARCHED), "expected the list");
}
#[given("a recent assembly is displayed in List View")]
fn assembly_listed(world: &mut ApplicationWorld) {
    assert!(home_shows(world, ASSEMBLY), "expected {ASSEMBLY}");
}
#[when("I open the assembly row's actions menu")]
fn assembly_actions(world: &mut ApplicationWorld) {
    open_actions_of(world, ASSEMBLY);
}
#[then("the menu offers Open Express")]
fn offers_express(world: &mut ApplicationWorld) {
    world.expect_text("Open Express");
}
#[then("File Open Options shows Full as the opening mode")]
fn mode_full(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Full");
}
#[then("the dialog shows Model State, Design View and Position View fields")]
fn assembly_fields(world: &mut ApplicationWorld) {
    for field in ["Model State:", "Design View:", "Position View:"] {
        world.expect_text(field);
    }
}
#[then("Skip all unresolved files is unchecked")]
fn skip_unchecked(world: &mut ApplicationWorld) {
    assert!(
        !world.checked("Skip all unresolved files"),
        "expected it clear"
    );
}
fn change_mode(world: &mut ApplicationWorld, mode: &str) {
    world.click_text("Mode: Full");
    world.pump(Vec::new());
    world.click_text(mode);
    world.pump(Vec::new());
}
#[when("I change the opening mode to Express")]
fn mode_express(world: &mut ApplicationWorld) {
    change_mode(world, "Express");
}
#[then("the opening mode shows Express")]
fn shows_express(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Express");
}
#[then("the Design View and Position View fields are disabled")]
fn views_disabled(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("Design View:"),
        "expected Design View disabled"
    );
    assert!(
        !world.run_is_enabled("Position View:"),
        "expected Position View disabled"
    );
}
fn save_recent_drawing(world: &mut ApplicationWorld) {
    let drawing = world.add_drawing();
    world.save_part_to_file(drawing, "Sheet.odd");
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[given("a recent drawing is displayed in List View")]
fn drawing_listed(world: &mut ApplicationWorld) {
    save_recent_drawing(world);
    assert!(
        home_shows(world, "Sheet"),
        "expected the drawing on the list"
    );
}
#[when("I open the drawing row's actions menu")]
fn drawing_actions(world: &mut ApplicationWorld) {
    open_actions_of(world, "Sheet");
}
#[then("the menu offers Defer Open")]
fn offers_defer(world: &mut ApplicationWorld) {
    world.expect_text("Defer Open");
}
#[when("I change the opening mode to Defer")]
fn mode_defer(world: &mut ApplicationWorld) {
    change_mode(world, "Defer");
}
#[then("the opening mode shows Defer")]
fn shows_defer(world: &mut ApplicationWorld) {
    world.expect_text("Mode: Defer");
}
#[then("Skip all unresolved files is disabled")]
fn skip_disabled(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled("Skip all unresolved files"),
        "expected it disabled"
    );
}
// Blank sizes
const BLANK: &str = "Gone";
#[given("one displayed entry has no size value")]
fn entry_without_size(world: &mut ApplicationWorld) {
    let part = world.add_part();
    let path = world.save_part_to_file(part, &format!("{BLANK}.opd"));
    std::fs::remove_file(&path).unwrap();
    // A later save refreshes the list, which reads each file's facts again.
    let other = world.add_part();
    world.save_part_to_file(other, "Later.opd");
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(home_shows(world, BLANK), "expected the entry on the list");
}
#[given("other displayed entries have known file sizes")]
fn others_have_sizes(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "KB"), "expected displayed sizes");
}
#[when("I select the Size column header twice")]
fn size_twice(world: &mut ApplicationWorld) {
    click_home(world, "Size");
    click_home(world, "Size");
}
#[then("the entry with no displayed size appears before the entries with known sizes")]
fn blank_first(world: &mut ApplicationWorld) {
    let blank = row_run(world, BLANK).center()[1];
    let known = row_run(world, SEARCHED).center()[1];
    assert!(
        blank < known,
        "blank at {blank}: expected above the known size at {known}"
    );
}
#[then("the entries with known sizes are ordered from smaller to larger")]
fn known_ascending(world: &mut ApplicationWorld) {
    let mut rows = rows_by_size(world);
    rows.retain(|row| row.1 > 0);
    assert!(
        rows.windows(2).all(|pair| pair[0].1 <= pair[1].1),
        "sizes {rows:?}"
    );
}
// Row hover
#[given("a recent part row is visible")]
fn part_row_visible(world: &mut ApplicationWorld) {
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[when("I hover over the part row")]
fn hover_row(world: &mut ApplicationWorld) {
    let at = row_run(world, SEARCHED).center().map(|value| value as i32);
    world.observed_row_plain = Some(row_ground(world, SEARCHED));
    world.hover(at);
    world.pump(Vec::new());
}
/// The row's colour well past its text, where only an accent reaches.
fn row_ground(world: &mut ApplicationWorld, name: &str) -> [u8; 4] {
    let run = row_run(world, name);
    world.pixel_at([run.minimum[0] as usize - 20, run.minimum[1] as usize - 1])
}
#[then("the row is visually highlighted")]
fn row_lit(world: &mut ApplicationWorld) {
    let plain = world
        .observed_row_plain
        .expect("expected the row's plain colour");
    assert_ne!(row_ground(world, SEARCHED), plain, "expected the row lit");
}
#[then("the row shows a selection checkbox and an actions control")]
fn row_controls(world: &mut ApplicationWorld) {
    let row = row_run(world, SEARCHED).center()[1];
    let control = world
        .frame
        .runs
        .iter()
        .any(|run| in_home(run) && run.text == "..." && (run.center()[1] - row).abs() < 12.);
    assert!(control, "expected the row's actions control");
}
#[when("I open the row's actions control")]
fn open_row_control(world: &mut ApplicationWorld) {
    open_row_actions(world);
}
#[then("the individual document actions are displayed")]
fn individual_actions(world: &mut ApplicationWorld) {
    world.expect_text("Remove from Recent");
    world.expect_text("Document Properties");
}
// Metadata
#[given("the default columns are displayed")]
fn default_columns(world: &mut ApplicationWorld) {
    location_after_name(world);
}
#[given("the recent list contains a part and a drawing")]
fn part_and_drawing(world: &mut ApplicationWorld) {
    save_recent_drawing(world);
    assert!(
        home_shows(world, SEARCHED) && home_shows(world, "Sheet"),
        "expected both rows"
    );
}
#[then("the list has Name, Location, Date Modified, Last Opened and Size headers")]
fn all_headers(world: &mut ApplicationWorld) {
    for column in ["Name", "Location", "Date Modified", "Last Opened", "Size"] {
        assert!(home_shows(world, column), "expected the {column} header");
    }
}
#[then("the part row shows its name, location, dates and size")]
fn part_row_facts(world: &mut ApplicationWorld) {
    let row = row_run(world, SEARCHED).center()[1];
    let on_row: Vec<String> = world
        .frame
        .runs
        .iter()
        .filter(|run| in_home(run) && (run.center()[1] - row).abs() < 8.)
        .map(|run| run.text.clone())
        .collect();
    let joined = on_row.join(" ");
    for part in ["/tmp", "UTC", "KB"] {
        assert!(
            joined.contains(part),
            "expected {part} on the row {joined:?}"
        );
    }
}
#[then("the part and drawing rows show different document-type symbols")]
fn different_symbols(world: &mut ApplicationWorld) {
    let mark = |name: &str| {
        let row = row_run(world, name);
        world
            .frame
            .runs
            .iter()
            .find(|run| {
                in_home(run)
                    && run.text.len() == 1
                    && run.maximum[0] < row.minimum[0]
                    && (run.center()[1] - row.center()[1]).abs() < 8.
            })
            .map(|run| run.text.clone())
    };
    let (part, drawing) = (mark(SEARCHED), mark("Sheet"));
    assert!(
        part.is_some() && part != drawing,
        "marks {part:?} and {drawing:?}"
    );
}
