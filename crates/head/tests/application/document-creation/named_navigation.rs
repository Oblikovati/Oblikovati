// SPDX-License-Identifier: GPL-2.0-only
//! Navigate the rendered tiles of the project's named Part catalog.
use super::*;
use imgui::sys;

const SHEET_DIN: &str = "Sheet Metal (DIN)";
const SHEET_MM: &str = "Sheet Metal (mm)";
const STANDARD_DIN: &str = "Standard (DIN)";
const STANDARD_MM: &str = "Standard (mm)";

pub(super) fn metric_icons(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    grouped_by_default(world);
    choose_template_view(world, "Medium Icons");
    assert_eq!(
        gallery_tile(world, SHEET_DIN).center()[1],
        gallery_tile(world, SHEET_MM).center()[1]
    );
    let size = gallery_icons::template_icon_size(world, SHEET_DIN);
    assert!(
        size.iter().all(|edge| (*edge - 40.).abs() < 1.),
        "expected a medium icon, got {size:?}"
    );
}

fn gallery_tile<'a>(
    world: &'a ApplicationWorld,
    title: &str,
) -> &'a super::super::super::diagnostics::RenderedRun {
    let details_x = world.run("Template details").minimum[0];
    let title = title.replace(' ', "");
    world
        .frame
        .runs
        .iter()
        .find(|run| run.minimum[0] < details_x && run.text == title)
        .unwrap_or_else(|| panic!("expected tile {title:?} in {}", world.text()))
}

struct GalleryScroll {
    position: [i32; 2],
    horizontal_track: [f32; 3],
    horizontal_range: f32,
    chooser_rect: [f32; 4],
}

fn gallery_scroll(world: &ApplicationWorld) -> GalleryScroll {
    let [x, y] = world.run("Display").center();
    let y = y + 30.;
    // SAFETY: the test owns the ImGui context and reads completed window state on its owner thread.
    unsafe {
        let context = sys::igGetCurrentContext()
            .as_ref()
            .expect("expected a live GUI context");
        assert!(context.Windows.Size > 0);
        let windows =
            std::slice::from_raw_parts(context.Windows.Data, context.Windows.Size as usize);
        let gallery = windows
            .iter()
            .filter_map(|ptr| ptr.as_ref())
            .find(|window| {
                window.Active
                    && window.ScrollbarX
                    && window.Flags & sys::ImGuiWindowFlags_ChildWindow as sys::ImGuiWindowFlags
                        != 0
                    && window.ParentWindow.as_ref().is_some_and(|parent| {
                        parent.Flags & sys::ImGuiWindowFlags_Modal as sys::ImGuiWindowFlags != 0
                    })
                    && (window.InnerRect.Min.x..window.InnerRect.Max.x).contains(&x)
                    && (window.InnerRect.Min.y..window.InnerRect.Max.y).contains(&y)
            })
            .expect("expected the visible template gallery child");
        let parent = gallery
            .ParentWindow
            .as_ref()
            .expect("expected the chooser modal");
        GalleryScroll {
            position: [gallery.Scroll.x as i32, gallery.Scroll.y as i32],
            horizontal_track: [
                gallery.InnerRect.Min.x,
                gallery.InnerRect.Max.x,
                gallery.Pos.y + gallery.Size.y - gallery.ScrollbarSizes.y * 0.5,
            ],
            horizontal_range: gallery.ScrollMax.x,
            chooser_rect: [parent.Pos.x, parent.Pos.y, parent.Size.x, parent.Size.y],
        }
    }
}

#[given("the part group displays Sheet Metal (mm) immediately before Standard (DIN)")]
fn adjacent_templates(world: &mut ApplicationWorld) {
    let first = gallery_tile(world, SHEET_DIN).center();
    let second = gallery_tile(world, SHEET_MM).center();
    inspect_adjacent_tiles(world, second[0] - first[0]);
}

fn inspect_adjacent_tiles(world: &mut ApplicationWorld, gap: f32) {
    let [left, right, y] = gallery_scroll(world).horizontal_track;
    let pixels = super::super::super::diagnostics::pointer_pixels;
    let width = right - left;
    world.drag(
        pixels([left + width * 0.25, y]),
        pixels([left + width * 0.5, y]),
    );
    let second = gallery_tile(world, SHEET_MM).center();
    let next = gallery_tile(world, STANDARD_DIN).center();
    assert_eq!(second[1], next[1]);
    assert!((next[0] - second[0] - gap).abs() < 2.);
    world.click(pixels([left + 6., y]));
    assert_eq!(gallery_scroll(world).position[0], 0);
}

#[when("the user selects the Sheet Metal (mm) part template")]
fn select_metric_sheet(world: &mut ApplicationWorld) {
    select_tile(world, SHEET_MM);
}

fn select_tile(world: &mut ApplicationWorld, title: &str) {
    let at = gallery_tile(world, title).center();
    world.click(super::super::super::diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
}

#[then("its tile is selected and its details identify millimeter units")]
fn metric_sheet(world: &mut ApplicationWorld) {
    expect_selected_template(world, SHEET_MM);
    expect_selected_template(world, "Length: millimeters (mm)");
    gallery_tile(world, SHEET_MM);
}

#[when("the user presses Right")]
fn press_right(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::RightArrow, false);
    world.pump(Vec::new());
}

#[when("the user presses Left")]
fn press_left(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::LeftArrow, false);
    world.pump(Vec::new());
}

#[then("the Standard (DIN) part tile is selected")]
fn standard_din_selected(world: &mut ApplicationWorld) {
    expect_selected_template(world, STANDARD_DIN);
    gallery_tile(world, STANDARD_DIN);
}

#[then("the details identify Standard (DIN) and millimeter units")]
fn standard_din_details(world: &mut ApplicationWorld) {
    expect_selected_template(world, STANDARD_DIN);
    expect_selected_template(world, "Length: millimeters (mm)");
}

#[then("the Sheet Metal (mm) part tile is selected again")]
#[then("Sheet Metal (mm) remains selected with its details displayed")]
fn sheet_selected(world: &mut ApplicationWorld) {
    metric_sheet(world);
}

#[then("its sheet-metal details are displayed")]
fn remember_before_down(world: &mut ApplicationWorld) {
    metric_sheet(world);
    world.observed_view = Some(gallery_scroll(world).position);
}

#[then("the template gallery scrolls down")]
fn gallery_scrolled_down(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    let before = world
        .observed_view
        .expect("expected gallery position before Down");
    let after = gallery_scroll(world).position;
    assert!(
        after[1] > before[1],
        "expected vertical scrolling from {before:?}, got {after:?}"
    );
}

#[given(
    "the user sizes the chooser so the gallery has a horizontal scrollbar and moves it to its leftmost position"
)]
fn narrow_chooser(world: &mut ApplicationWorld) {
    let [x, y, width, height] = gallery_scroll(world).chooser_rect;
    let pixels = super::super::super::diagnostics::pointer_pixels;
    let grip = [x + width - 4., y + height - 4.];
    world.drag(pixels(grip), pixels([grip[0] - 115., grip[1]]));
    let narrowed = gallery_scroll(world);
    assert!(narrowed.chooser_rect[2] < width && narrowed.horizontal_range > 0.);
    move_to_track_edge(world, false);
    at_leftmost_position(world);
}

fn move_to_track_edge(world: &mut ApplicationWorld, end: bool) {
    let [left, right, y] = gallery_scroll(world).horizontal_track;
    let x = if end { right - 6. } else { left + 6. };
    world.click(super::super::super::diagnostics::pointer_pixels([x, y]));
}

#[given("the Part group begins with Sheet Metal (DIN) and ends with Standard (mm)")]
fn part_endpoints(world: &mut ApplicationWorld) {
    assert!(gallery_tile(world, SHEET_DIN).center()[0] < gallery_tile(world, SHEET_MM).center()[0]);
    move_to_track_edge(world, true);
    assert!(
        gallery_tile(world, STANDARD_DIN).center()[0]
            < gallery_tile(world, STANDARD_MM).center()[0]
    );
    move_to_track_edge(world, false);
    at_leftmost_position(world);
}

fn reveal_rightmost_tile(world: &mut ApplicationWorld) {
    let gallery = gallery_scroll(world);
    let [left, right, y] = gallery.horizontal_track;
    let width = right - left;
    let track = width - 6.;
    let thumb = track * width / (width + gallery.horizontal_range);
    let start = left + 3. + thumb * 0.5;
    let end = start + (track - thumb) * 0.97;
    let pixels = super::super::super::diagnostics::pointer_pixels;
    world.drag(pixels([start, y]), pixels([end, y]));
}

#[when("the user selects Standard (mm) and presses Right")]
fn press_right_on_last_tile(world: &mut ApplicationWorld) {
    reveal_rightmost_tile(world);
    select_tile(world, STANDARD_MM);
    expect_selected_template(world, STANDARD_MM);
    let position = gallery_scroll(world).position;
    assert!(position[0] < gallery_scroll(world).horizontal_range as i32);
    world.observed_view = Some(position);
    press_right(world);
}

#[then("Standard (mm) remains selected and identified in the details pane")]
fn remains_last(world: &mut ApplicationWorld) {
    expect_selected_template(world, STANDARD_MM);
    gallery_tile(world, STANDARD_MM);
}

#[then("the gallery scroll position advances within the Part group")]
fn moves_toward_the_right(world: &mut ApplicationWorld) {
    let before = world
        .observed_view
        .expect("expected the gallery position before Right");
    let after = gallery_scroll(world).position;
    assert!(
        after[0] > before[0],
        "expected horizontal scrolling from {before:?}, got {after:?}"
    );
}

#[then("Standard (DIN) is selected and identified in the details pane")]
fn identifies_standard_din(world: &mut ApplicationWorld) {
    standard_din_selected(world);
}

#[when("the user presses Home and then Left")]
fn home_then_left(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Home, false);
    world.pump(Vec::new());
    press_left(world);
}

#[then("Sheet Metal (DIN) remains selected and identified in the details pane")]
fn remains_first(world: &mut ApplicationWorld) {
    expect_selected_template(world, SHEET_DIN);
    gallery_tile(world, SHEET_DIN);
}

#[then("the gallery returns to its leftmost horizontal position")]
fn at_leftmost_position(world: &mut ApplicationWorld) {
    assert_eq!(gallery_scroll(world).position[0], 0);
}
