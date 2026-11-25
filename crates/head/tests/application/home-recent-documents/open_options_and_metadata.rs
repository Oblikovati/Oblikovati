// SPDX-License-Identifier: GPL-2.0-only
use super::*;

// Open options
#[given("a recent part with a Front design view is displayed in List View")]
#[given(
    "Part26.opd from the disposable package is displayed in List View with a Front design-view name"
)]
fn front_view_part(world: &mut ApplicationWorld) {
    // Native parts already write Front as a standard design view.
    let state = world.state();
    let entry = state
        .recent_documents
        .iter()
        .find(|entry| entry.name == SEARCHED);
    assert!(
        entry.is_some_and(|entry| entry.design_views.iter().any(|view| view == "Front")),
        "expected the saved {SEARCHED} to have a Front design view"
    );
    assert!(home_shows(world, SEARCHED), "expected the part on the list");
}
#[when("the user chooses Open with options")]
#[when("I choose Open with options")]
fn choose_open_options(world: &mut ApplicationWorld) {
    world.click_text("Open with options");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[test]
fn recent_file_options_title_close_keeps_home_open() {
    let mut world = ApplicationWorld::new();
    home_active(&mut world);
    open_actions_of(&mut world, SEARCHED);
    choose_open_options(&mut world);
    let (position, size) = window_geometry("File Open Options").unwrap();
    let close = super::super::super::diagnostics::pointer_pixels([
        position[0] + size[0] - 13.,
        position[1] + 10.,
    ]);
    world.click(close);
    assert!(window_geometry("File Open Options").is_none());
    assert!(home_shows(&world, SEARCHED));
    world.close();
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
#[when("the user chooses Front")]
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
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_some(),
        "expected {assembly}"
    );
}
#[when("the user opens the assembly row's actions menu")]
#[when("I open the assembly row's actions menu")]
fn assembly_actions(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    open_actions_of(world, assembly);
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
#[when("the user changes the opening mode to Express")]
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
fn drawing_name(world: &ApplicationWorld) -> &'static str {
    if published_profile::uses_published_profile(world) {
        "RecentDrawing"
    } else {
        "Sheet"
    }
}
#[given("a recent drawing is displayed in List View")]
fn drawing_listed(world: &mut ApplicationWorld) {
    if !published_profile::uses_published_profile(world) {
        save_recent_drawing(world);
    }
    let drawing = drawing_name(world);
    assert!(
        recent_name(world, drawing).is_some(),
        "expected the {drawing} row"
    );
}
#[when("the user opens the drawing row's actions menu")]
#[when("I open the drawing row's actions menu")]
fn drawing_actions(world: &mut ApplicationWorld) {
    let drawing = drawing_name(world);
    open_actions_of(world, drawing);
}
#[then("the menu offers Defer Open")]
fn offers_defer(world: &mut ApplicationWorld) {
    world.expect_text("Defer Open");
}
#[when("the user changes the opening mode to Defer")]
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
#[when("the user selects the Size column header twice")]
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
    let rows = rows_by_size(world);
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
#[when("the user hovers over the part row")]
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
#[when("the user opens the row's actions control")]
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
    if published_profile::uses_published_profile(world) {
        drawing_listed(world);
        assert!(recent_name(world, SEARCHED).is_some());
        return;
    }
    let part_path = world.scratch_folder().join(format!("{SEARCHED}.opd"));
    std::fs::OpenOptions::new()
        .write(true)
        .open(part_path)
        .unwrap()
        .set_modified(std::time::UNIX_EPOCH + std::time::Duration::from_secs(1_600_000_000))
        .unwrap();
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
    for part in ["/tmp", "KB"] {
        assert!(
            joined.contains(part),
            "expected {part} on the row {joined:?}"
        );
    }
    let facts = world
        .state()
        .recent_documents
        .into_iter()
        .find(|entry| entry.name == SEARCHED)
        .unwrap();
    assert_ne!(
        facts.modified_text, facts.opened_text,
        "expected different fixture timestamps"
    );
    expect_recent_timestamp(
        world,
        row,
        "Date Modified",
        "Last Opened",
        &facts.modified_text,
    );
    expect_recent_timestamp(world, row, "Last Opened", "Size", &facts.opened_text);
}

fn expect_recent_timestamp(
    world: &mut ApplicationWorld,
    row: f32,
    heading: &str,
    next: &str,
    full: &str,
) {
    let date = full.split_once(' ').expect("expected an ISO timestamp").0;
    let left = home_run(world, heading)
        .expect("expected the timestamp heading")
        .minimum[0];
    let right = home_run(world, next)
        .expect("expected the next heading")
        .minimum[0];
    let cell = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let [x, y] = run.center();
            in_home(run)
                && (y - row).abs() < 8.
                && x > left
                && x < right
                && run.text.starts_with(date)
        })
        .unwrap_or_else(|| panic!("expected {heading} to display {date}"));
    if cell.text == full.replace(' ', "") {
        return;
    }
    assert_eq!(cell.text, date, "expected the complete date in {heading}");
    world.hover(super::super::super::diagnostics::pointer_pixels(
        cell.center(),
    ));
    world.pump(Vec::new());
    world.expect_text(full);
}
#[then("the part and drawing rows show different document-type symbols")]
fn different_symbols(world: &mut ApplicationWorld) {
    let positions = [SEARCHED, drawing_name(world)].map(|name| {
        let row = row_run(world, name);
        super::super::super::diagnostics::pointer_pixels([
            row.minimum[0] - 22.,
            row.center()[1] - 8.,
        ])
        .map(|coordinate| coordinate as usize)
    });
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let stride = capture.extent[0] as usize * 4;
    let [part, drawing] = positions.map(|[x, y]| {
        let rows: Vec<&[u8]> = (y..y + 16)
            .map(|row| &capture.rgba[row * stride + 4 * x..row * stride + 4 * (x + 16)])
            .collect();
        assert!(
            rows.iter()
                .any(|line| line.as_chunks::<4>().0.iter().any(|pixel| pixel[2] > 140))
        );
        rows
    });
    assert_ne!(
        part, drawing,
        "expected different Part and Drawing icon pixels"
    );
}
