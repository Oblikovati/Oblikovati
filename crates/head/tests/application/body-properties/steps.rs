// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/body-properties`: a solid's values on request and its appearance.
use super::workspace_layout::browser_feature::{
    SOLID_COLOR, choose_menu_row, click_branch_arrow, colored_pixels, expect_rows,
    open_visible_extruded_part, right_click_row,
};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::CreateAppearanceArgs};

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
    world.app().owner.load_project_for_test(&path);
}
fn part_with_body(world: &mut ApplicationWorld) {
    load_library_project(world);
    open_visible_extruded_part(world);
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
fn expect_physical_calculated(world: &ApplicationWorld) {
    world.expect_no_text("Mass: N/A");
    for row in [MASS, AREA, VOLUME] {
        world.expect_text(row);
    }
    for row in CENTER {
        world.expect_text(row);
    }
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
// Nothing has asked for the values yet, in a session that has just opened the part.
#[given("the body's physical properties are unavailable in the current session")]
fn unavailable(_: &mut ApplicationWorld) {}
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
fn right_click_body(world: &mut ApplicationWorld) {
    right_click_row(world, BODY);
}
#[when("I choose Properties")]
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
fn choose_update(world: &mut ApplicationWorld) {
    world.click_text("Update");
    world.pump(Vec::new());
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
fn reopen(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[when("I open the Body Appearance list")]
fn open_list(world: &mut ApplicationWorld) {
    open_appearance_list(world);
}
#[then("a scrollable list of named appearances is displayed")]
fn named_list(world: &mut ApplicationWorld) {
    world.expect_text(BLACK);
    world.expect_text(AS_PART);
}
#[when("I choose Black")]
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
fn select_black(world: &mut ApplicationWorld) {
    select_appearance(world, BLACK);
}
#[when("I select As Part in Body Appearance")]
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
    assert_eq!(
        colored_pixels(world, SOLID_COLOR),
        0,
        "expected no part colour left"
    );
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    assert!(
        super::solid_visibility::visible_appearance_fill(&capture, &windows, BLACK_COLOR),
        "expected the model's Black appearance outside the GUI windows"
    );
}
#[then("the solid is displayed with the part appearance")]
fn solid_part(world: &mut ApplicationWorld) {
    assert!(
        colored_pixels(world, SOLID_COLOR) > 0,
        "expected the part's colour"
    );
}
#[then("the body row remains selected in the model browser")]
fn row_selected(world: &mut ApplicationWorld) {
    assert!(
        world.row_highlighted(BODY),
        "expected the body row's accent"
    );
}
#[when("I close the dialog using its title-bar close control")]
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
