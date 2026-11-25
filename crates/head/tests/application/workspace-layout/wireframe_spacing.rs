// SPDX-License-Identifier: GPL-2.0-only
//! Check shared workspace edges and spacing in the completed desktop image (#3897).
use super::super::{capture_framebuffer, diagnostics};
use super::{ApplicationWorld, browser_feature, window_geometry};
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportCapture;

const RIBBON: &str = "##application-ribbon";
const STATUS: &str = "##status-bar";
const TABS: &str = "##open-document-tabs";

fn bounds(name: &str) -> [u32; 4] {
    let (position, size) =
        window_geometry(name).unwrap_or_else(|| panic!("{name}: expected a workspace region"));
    let [left, top] = diagnostics::pointer_pixels(position).map(|value| value as u32);
    let [right, bottom] =
        diagnostics::pointer_pixels([position[0] + size[0], position[1] + size[1]])
            .map(|value| value as u32);
    [left, top, right, bottom]
}

#[given("the ribbon, model browser, viewport, document tabs and status area are visible")]
fn regions_are_visible(world: &mut ApplicationWorld) {
    for name in [RIBBON, "Model", TABS, STATUS] {
        let [left, top, right, bottom] = bounds(name);
        assert!(
            left < right && top < bottom,
            "{name}: expected a visible workspace region"
        );
    }
    browser_feature::expect_solid_visible(world);
}

#[when("the user inspects the spacing between workspace regions and their controls")]
fn inspect_spacing(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}

#[then("adjacent workspace regions share their intended borders without stray gutters")]
fn regions_share_borders(world: &mut ApplicationWorld) {
    let browser = bounds("Model");
    let viewport = world.viewport_bounds();
    assert_near(bounds(RIBBON)[3], browser[1]);
    assert_near(browser[1], viewport[1]);
    assert!(
        super::dock_splitter::right_splitter("Model", browser[2] as i32, viewport[0] as i32)
            .is_some(),
        "browser {browser:?}, viewport {viewport:?}: expected only the dock splitter"
    );
    assert_near(viewport[3], bounds(TABS)[1]);
    assert_near(bounds(TABS)[3], bounds(STATUS)[1]);
    assert_near(browser[3], bounds(STATUS)[1]);
}

fn assert_near(first: u32, second: u32) {
    assert!(
        first.abs_diff(second) <= 1,
        "edges {first} and {second}: expected adjacent pixels"
    );
}

#[then("controls within each region use a consistent inset from that region's border")]
fn controls_stay_inside_regions(world: &mut ApplicationWorld) {
    let file = world.run("File").minimum;
    let menu = world.run("Menu").minimum;
    let home = world.expect_tab("Home").minimum;
    let prompt = status_prompt(world);
    let insets = [
        (file, RIBBON),
        (menu, "Model"),
        (home, TABS),
        (prompt, STATUS),
    ]
    .map(|(anchor, region)| leading_inset(anchor, bounds(region)));
    let file_bottom = diagnostics::pointer_pixels(world.run("File").maximum)[1];
    let line_height = file_bottom - diagnostics::pointer_pixels(file)[1];
    assert!(
        insets.iter().max().unwrap() - insets.iter().min().unwrap() <= line_height as u32,
        "leading text insets {insets:?}: expected consistent spacing within one glyph height"
    );
}

fn status_prompt(world: &ApplicationWorld) -> [f32; 2] {
    let status = bounds(STATUS);
    let prompt = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let y = diagnostics::pointer_pixels(run.center())[1] as u32;
            status[1] < y && y < status[3]
        })
        .expect("status area: expected its prompt");
    prompt.minimum
}

fn leading_inset(anchor: [f32; 2], rect: [u32; 4]) -> u32 {
    let x = diagnostics::pointer_pixels(anchor)[0] as u32;
    let height = rect[3] - rect[1];
    assert!(
        rect[0] < x && x <= rect[0] + height,
        "control at {x} in {rect:?}: expected it near the leading edge"
    );
    x - rect[0]
}

#[then("the ribbon command groups use even spacing between neighboring groups")]
fn command_groups_use_even_spacing(world: &mut ApplicationWorld) {
    let row = diagnostics::pointer_pixels(world.run("Extrude").center())[1] as u32;
    let ribbon = bounds(RIBBON);
    let last_control = diagnostics::pointer_pixels(world.run("Simplify").maximum)[0] as u32;
    let background_x = last_control.midpoint(ribbon[2]);
    let capture = capture_framebuffer(&mut world.app().window.native);
    let gaps = ribbon_gaps(&capture, ribbon[0], background_x, row);
    let (min, max) = (gaps.iter().min(), gaps.iter().max());
    assert!(
        matches!((min, max), (Some(min), Some(max)) if gaps.len() >= 3 && max - min <= 2),
        "ribbon gap widths {gaps:?}: expected at least three even gaps"
    );
}

fn ribbon_gaps(image: &ViewportCapture, left: u32, end: u32, y: u32) -> Vec<u32> {
    let row = (y * image.extent[0] + left) as usize * 4;
    let span = (end - left + 1) as usize * 4;
    let pixels = &image.rgba[row..row + span];
    let background = &pixels[span - 4..span];
    let samples = pixels
        .as_chunks::<4>()
        .0
        .iter()
        .map(|pixel| pixel.as_slice() == background)
        .collect::<Vec<_>>();
    let Some(first) = samples.iter().position(|&is_background| !is_background) else {
        return Vec::new();
    };
    let last = samples
        .iter()
        .rposition(|&is_background| !is_background)
        .unwrap();
    samples[first..=last]
        .chunk_by(|a, b| a == b)
        .filter(|run| run[0])
        .map(|run| run.len() as u32)
        .collect()
}

#[then("the browser, viewport, document tabs and status area align to their shared edges")]
fn align_bottom_rows(world: &mut ApplicationWorld) {
    let viewport = world.viewport_bounds();
    assert_near(bounds(TABS)[0], viewport[0]);
    assert_near(bounds("##document-list-control")[2], viewport[2]);
    assert_near(bounds(STATUS)[0], bounds("Model")[0]);
    assert_near(bounds(STATUS)[2], world.extent()[0]);
    super::wireframe_tabs::viewport_fills_width(world);
}

#[then("no region has an unexplained extra margin or padding")]
fn no_extra_shell_margin(world: &mut ApplicationWorld) {
    let ribbon = bounds(RIBBON);
    let status = bounds(STATUS);
    assert_eq!(
        [ribbon[0], ribbon[2], status[3]],
        [0, world.extent()[0], world.extent()[1]]
    );
    assert_eq!(bounds("Model")[0], ribbon[0]);
    align_bottom_rows(world);
}
