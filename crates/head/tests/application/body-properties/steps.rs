// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/body-properties`: a solid's values on request and its appearance.
use super::workspace_layout::browser_feature::{
    choose_menu_row, click_branch_arrow, expect_rows, open_visible_extruded_part, right_click_row,
};
use super::{ApplicationWorld, diagnostics, scenario_root};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::CreateAppearanceArgs};
use oblikovati_viewport::{ViewportEvent, ViewportInput};
use std::path::Path;

#[cfg(test)]
#[path = "native_input.rs"]
mod native_input;

const TITLE: &str = "Body Properties";
const BODY: &str = "Solid1";
const AS_PART: &str = "As Part";
const BLACK: &str = "Black";
/// The shipped look the fixture's library copies as "Black", and how that renders.
const BLACK_BASE: &str = "abs-black";
// Catalog RGB 0.0065849577 is linear; the sRGB framebuffer stores byte 19.
const BLACK_COLOR: [u8; 3] = [19, 19, 19];
/// The solid is 40 by 30 by 20 mm from the origin; the catalog's default density is 1 g/cm^3.
const VOLUME: &str = "Volume: 24000.000 mm^3";
const AREA: &str = "Area: 5200.000 mm^2";
const MASS: &str = "Mass: 24.000 g";
const CENTER: [&str; 3] = ["X: 20.000 mm", "Y: 15.000 mm", "Z: 10.000 mm"];
const GENERATED_PHYSICAL: [&str; 6] = [MASS, AREA, VOLUME, CENTER[0], CENTER[1], CENTER[2]];
const SAVED_PART: &str = "body-properties-start.opd";
const CURRENT_SAVED_PART: &str = "docking-visible-solid-current.opd";
// The saved 40 × 25 × 20 mm prism extends from the origin toward negative X and Y.
const SAVED_PHYSICAL: [&str; 6] = [
    "Mass: 20.000 g",
    "Area: 4600.000 mm^2",
    "Volume: 20000.000 mm^3",
    "X: -20.000 mm",
    "Y: -12.500 mm",
    "Z: 10.000 mm",
];
/// The view between the browser and where the dialog opens.
const VIEW_COLUMNS: [usize; 2] = [320, 860];
const VIEW_ROWS: [usize; 2] = [170, 860];
const HUE_MARGIN: u8 = 20;
const PHYSICAL_LABELS: [&str; 6] = ["Mass", "Area", "Volume", "X", "Y", "Z"];
/// Distance of the title bar's close control from the window's top right corner.
const CLOSE_CONTROL: [f32; 2] = [13., 10.];

/// A project keeps the library the fixture's "Black" is made in.
fn load_library_project(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    std::fs::create_dir_all(folder.join("design")).unwrap();
    let path = folder.join("library.opj");
    std::fs::write(
        &path,
        "schemaVersion: 1\nname: Library\nworkspace: work\ndesignData: design\n",
    )
    .unwrap();
    world.app().owner.load_desktop_project(&path).unwrap();
}
fn part_with_body(world: &mut ApplicationWorld) {
    load_library_project(world);
    open_visible_extruded_part(world);
    prepare_body_appearance(world);
}
fn prepare_body_appearance(world: &mut ApplicationWorld) {
    Client::new(world.app().owner.document_host())
        .appearances()
        .create(&CreateAppearanceArgs {
            base_id: BLACK_BASE.into(),
            name: BLACK.into(),
        })
        .unwrap();
    world.pump(Vec::new());
    click_branch_arrow(world, "Solid Bodies");
    world.expect_text(BODY);
    let point = super::ribbon_keyboard_navigation::visible_face_point(world);
    world.observed_view = Some(point);
    assert!(super::solid_visibility::neutral_fixture_hue(
        &body_face_pixel(world)
    ));
}
fn open_dialog(world: &mut ApplicationWorld) {
    right_click_row(world, BODY);
    choose_menu_row(world, "Properties");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text(TITLE);
}
fn open_appearance_list(world: &mut ApplicationWorld) {
    let label = world.run("Body Appearance").clone();
    // The field runs left of its label.
    world.click([label.minimum[0] as i32 - 40, label.center()[1] as i32]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn select_appearance(world: &mut ApplicationWorld, name: &str) {
    open_appearance_list(world);
    world.click_text(name);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
/// Pixels of the view left of the dialog where a test holds, dimmed behind it or not.
fn view_pixels(world: &mut ApplicationWorld, holds: fn(&[u8]) -> bool) -> usize {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    capture
        .rgba
        .chunks_exact(width * 4)
        .skip(VIEW_ROWS[0])
        .take(VIEW_ROWS[1] - VIEW_ROWS[0])
        .map(|row| {
            let span = &row[VIEW_COLUMNS[0] * 4..VIEW_COLUMNS[1] * 4];
            span.as_chunks::<4>()
                .0
                .iter()
                .filter(|pixel| holds(&pixel[..]))
                .count()
        })
        .sum()
}
fn expect_highlighted(world: &mut ApplicationWorld) {
    let outlined = view_pixels(world, |pixel| {
        pixel[1] > pixel[0].saturating_add(HUE_MARGIN)
            && pixel[2] > pixel[1].saturating_add(HUE_MARGIN)
    });
    assert!(outlined > 0, "expected the solid's edges outlined");
}
fn expected_physical_rows(world: &mut ApplicationWorld) -> [&'static str; 6] {
    let document = world.active_id();
    let state = world.state();
    match state.document_storage[&document]
        .path
        .as_deref()
        .and_then(|path| Path::new(path).file_name())
    {
        Some(name) if name == SAVED_PART => SAVED_PHYSICAL,
        _ => GENERATED_PHYSICAL,
    }
}
fn expect_physical_calculated(world: &mut ApplicationWorld) {
    let expected = expected_physical_rows(world);
    validate_physical_rows(world.window_text(TITLE).split_whitespace(), expected).unwrap();
}
fn validate_physical_rows<'a>(
    rows: impl Iterator<Item = &'a str>,
    expected: [&str; 6],
) -> Result<(), String> {
    let rows: Vec<String> = rows
        .map(|row| row.chars().filter(|ch| !ch.is_whitespace()).collect())
        .collect();
    for expected in expected {
        let expected: String = expected.chars().filter(|ch| !ch.is_whitespace()).collect();
        let label = expected.split(':').next().unwrap();
        let matching: Vec<_> = rows
            .iter()
            .filter(|row| row.starts_with(&format!("{label}:")))
            .collect();
        if matching.len() != 1 || matching[0] != &expected {
            return Err(format!(
                "physical field {label:?} is {matching:?}: expected exactly {expected:?}"
            ));
        }
    }
    Ok(())
}
fn expect_selector_reads(world: &ApplicationWorld, name: &str) {
    let label = world.run("Body Appearance").clone();
    let chosen = world.run(name).clone();
    assert!(
        (chosen.center()[1] - label.center()[1]).abs() < 4. && chosen.maximum[0] < label.minimum[0],
        "expected {name} in the selector, at {chosen:?} beside {label:?}"
    );
}
#[given("a part containing one visible solid body is active in the MCAD Software")]
#[given("a part with a visible solid body is active in the MCAD Software")]
fn given_part(world: &mut ApplicationWorld) {
    part_with_body(world);
}
#[given(
    "`fixtures/cad/body-properties-start.opd` is open in the MCAD Software with `Solid1` visible"
)]
fn given_saved_part(world: &mut ApplicationWorld) {
    open_saved_body_part(world, SAVED_PART);
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open in the MCAD Software with its solid body visible"
)]
fn given_current_saved_part(world: &mut ApplicationWorld) {
    open_saved_body_part(world, CURRENT_SAVED_PART);
}

fn open_saved_body_part(world: &mut ApplicationWorld, filename: &str) {
    let source = scenario_root().join("fixtures/cad").join(filename);
    let copy = world.scratch_folder().join(filename);
    std::fs::copy(&source, &copy).unwrap_or_else(|error| {
        panic!("body fixture {source:?}: expected a readable native Part: {error}")
    });
    load_library_project(world);
    world.open_fixture(&copy);
    unavailable(world);
    prepare_body_appearance(world);
}
// Nothing has asked for the values yet, in a session that has just opened the part.
#[given("the body's physical properties are unavailable in the current session")]
#[given(
    "no physical-property update has been requested for `Solid1` since this document instance was opened"
)]
#[given(
    "no physical-property update has been requested for the body since this document instance was opened"
)]
fn unavailable(world: &mut ApplicationWorld) {
    let state = world.state();
    let browser = state.browser.expect("expected the active part browser");
    assert_eq!(browser.bodies.len(), 1, "expected one solid body");
    let body = &browser.bodies[0];
    assert_eq!(body.name, BODY);
    assert!(body.visible, "expected the solid body to be visible");
    assert!(
        body.physical.is_none(),
        "expected unavailable physical properties"
    );
}
#[given("the body appearance is set to As Part")]
fn as_part(world: &mut ApplicationWorld) {
    open_dialog(world);
    expect_selector_reads(world, AS_PART);
    world.click_text("Cancel");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[given("Body Properties is open for the body")]
fn given_open(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[when("I right-click the body in the model browser")]
#[when("the user right-clicks the body in the model browser")]
fn right_click_body(world: &mut ApplicationWorld) {
    right_click_row(world, BODY);
}
#[when("I choose Properties")]
#[when("the user chooses Properties")]
fn choose_properties(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Properties");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the Body Properties dialog opens with the body name")]
fn dialog_opens(world: &mut ApplicationWorld) {
    world.expect_text(TITLE);
    world.expect_text("General Properties");
    world.expect_text(BODY);
}
#[then("mass, area, volume, and the X, Y, and Z coordinates of the center of gravity show N/A")]
fn shows_not_available(world: &mut ApplicationWorld) {
    for label in PHYSICAL_LABELS {
        world.expect_text(&format!("{label}: N/A"));
    }
}
#[then("the dialog displays a body appearance selector and OK and Cancel buttons")]
fn selector_and_buttons(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Body Appearance", "OK", "Cancel", "Clear All Overrides"],
    );
}
#[when("I choose Update")]
#[when("the user chooses Update")]
fn choose_update(world: &mut ApplicationWorld) {
    let caption = world.window_text(TITLE);
    let label = caption
        .split_whitespace()
        .find(|run| run.ends_with("Update"))
        .expect("expected the dialog's Update control");
    let run = world.run_in_window(TITLE, label);
    // The adjacent body name can share the Update caption's rendered text run.
    world.click(diagnostics::pointer_pixels([
        run.maximum[0] - 3.,
        run.center()[1],
    ]));
    world.pump(Vec::new());
}
#[then("mass, area, volume, and the center of gravity coordinates display calculated values")]
#[then("the calculated physical properties remain displayed")]
fn calculated(world: &mut ApplicationWorld) {
    expect_physical_calculated(world);
}
#[then("the solid remains highlighted behind the dialog")]
#[then("the solid remains highlighted in the viewport")]
fn highlighted_behind(world: &mut ApplicationWorld) {
    expect_highlighted(world);
}
#[when("I reopen Body Properties for the same body in the same session")]
#[when("I reopen Body Properties for the same body")]
#[when("I open Body Properties for the body from the model browser")]
#[when("the user reopens Body Properties for the same body in the same session")]
#[when("the user reopens Body Properties for the same body")]
#[when("the user opens Body Properties for the body from the model browser")]
fn reopen(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[when("I open the Body Appearance list")]
#[when("the user opens the Body Appearance list")]
fn open_list(world: &mut ApplicationWorld) {
    open_appearance_list(world);
}
#[then("a scrollable list of named appearances is displayed")]
fn named_list(world: &mut ApplicationWorld) {
    world.expect_text(BLACK);
    world.expect_text(AS_PART);
    let first = world.run(BLACK).clone();
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == first.list && run.center()[1] > first.center()[1] + 90.)
        .expect("expected a second visible appearance below Black")
        .clone();
    let before = row.center();
    let position = before.map(|value| value as i32);
    scroll_appearance_list(world, position, -1);
    let after = world.run(&row.text).center();
    assert!(
        after[1] < before[1],
        "appearance row {before:?}, {after:?}: expected scrolling"
    );
    scroll_appearance_list(world, position, 1);
    world.expect_text(BLACK);
}
fn scroll_appearance_list(world: &mut ApplicationWorld, position: [i32; 2], notches: i32) {
    world.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers: Default::default(),
    })]);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches,
        position,
    })]);
}
#[when("I choose Black")]
#[when("the user chooses Black")]
fn choose_black(world: &mut ApplicationWorld) {
    world.click_text(BLACK);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("Body Appearance displays Black")]
fn displays_black(world: &mut ApplicationWorld) {
    expect_selector_reads(world, BLACK);
}
#[then("Body Appearance displays As Part")]
fn displays_as_part(world: &mut ApplicationWorld) {
    expect_selector_reads(world, AS_PART);
}
#[when("I select Black in Body Appearance")]
#[when("the user selects Black in Body Appearance")]
fn select_black(world: &mut ApplicationWorld) {
    select_appearance(world, BLACK);
}
#[when("I select As Part in Body Appearance")]
#[when("the user selects As Part in Body Appearance")]
fn select_as_part(world: &mut ApplicationWorld) {
    select_appearance(world, AS_PART);
}
#[when("I choose OK")]
fn choose_ok(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the solid is displayed with the Black appearance")]
fn solid_black(world: &mut ApplicationWorld) {
    let pixel = body_face_pixel(world);
    assert!(
        (0..3).all(|channel| pixel[channel].abs_diff(BLACK_COLOR[channel]) <= 10),
        "expected the sampled face to show Black, found {pixel:?}"
    );
}
#[then("the solid is displayed with the part appearance")]
fn solid_part(world: &mut ApplicationWorld) {
    let pixel = body_face_pixel(world);
    assert!(
        super::solid_visibility::neutral_fixture_hue(&pixel),
        "expected the sampled face to show the part appearance, found {pixel:?}"
    );
}
fn body_face_pixel(world: &mut ApplicationWorld) -> [u8; 4] {
    let point = world.observed_view.expect("expected the sampled body face");
    world.pixel_at(point.map(|coordinate| coordinate as usize))
}
#[then("the body row remains selected in the model browser")]
fn row_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(BODY),
        "expected the body row's accent"
    );
}
#[when("I close the dialog using its title-bar close control")]
#[when("the user closes the dialog using its title-bar close control")]
fn close_control(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(TITLE).expect("expected the dialog");
    world.click([
        (position[0] + size[0] - CLOSE_CONTROL[0]) as i32,
        (position[1] + CLOSE_CONTROL[1]) as i32,
    ]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[test]
fn body_appearance_applies_and_restores_the_part_color() {
    let mut world = ApplicationWorld::new();
    part_with_body(&mut world);
    open_dialog(&mut world);
    select_appearance(&mut world, BLACK);
    choose_ok(&mut world);
    solid_black(&mut world);
    open_dialog(&mut world);
    select_appearance(&mut world, AS_PART);
    choose_ok(&mut world);
    solid_part(&mut world);
    world.close();
}

#[test]
fn body_properties_title_close_discards_the_pending_appearance() {
    let mut world = ApplicationWorld::new();
    part_with_body(&mut world);
    open_dialog(&mut world);
    select_appearance(&mut world, BLACK);
    displays_black(&mut world);
    close_control(&mut world);
    assert!(diagnostics::window_geometry(TITLE).is_none());
    open_dialog(&mut world);
    displays_as_part(&mut world);
    close_control(&mut world);
    world.close();
}

#[test]
fn body_properties_closes_when_the_active_document_changes() {
    let mut world = ApplicationWorld::new();
    part_with_body(&mut world);
    let original = world.active_id();
    open_dialog(&mut world);
    select_appearance(&mut world, BLACK);
    world.add_part();
    assert!(diagnostics::window_geometry(TITLE).is_none());
    world.click_tab(original);
    open_dialog(&mut world);
    displays_as_part(&mut world);
    close_control(&mut world);
    world.close();
}

#[test]
fn body_properties_invalidates_calculated_values_after_an_edit() {
    let mut world = ApplicationWorld::new();
    part_with_body(&mut world);
    unavailable(&mut world);
    open_dialog(&mut world);
    choose_update(&mut world);
    expect_physical_calculated(&mut world);
    let body = world.state().browser.unwrap().bodies.remove(0);
    assert!(body.physical.is_some());
    select_appearance(&mut world, BLACK);
    choose_ok(&mut world);
    unavailable(&mut world);
    assert_eq!(world.state().browser.unwrap().bodies[0].key, body.key);
    open_dialog(&mut world);
    shows_not_available(&mut world);
    close_control(&mut world);
    world.close();
}

#[test]
fn calculated_physical_rows_reject_missing_non_numeric_and_partial_values() {
    let valid = GENERATED_PHYSICAL;
    assert!(validate_physical_rows(valid.into_iter(), valid).is_ok());
    for index in 0..valid.len() {
        let mut missing = valid.to_vec();
        missing.remove(index);
        assert!(validate_physical_rows(missing.into_iter(), valid).is_err());
        let mut invalid = valid.to_vec();
        invalid[index] = [
            "Mass: N/A",
            "Area: N/A",
            "Volume: N/A",
            "X: N/A",
            "Y: N/A",
            "Z: N/A",
        ][index];
        assert!(validate_physical_rows(invalid.into_iter(), valid).is_err());
    }
    let mut partial = valid;
    partial[1] = "Area: 5200.000 mm^2 extra";
    assert!(validate_physical_rows(partial.into_iter(), valid).is_err());
}
