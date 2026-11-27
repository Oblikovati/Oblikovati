// SPDX-License-Identifier: GPL-2.0-only
//! Verify the rendered controls and the shared viewport of an arranged native dock.
use super::super::{capture_framebuffer, desktop_input};
use super::browser_scenarios::select_tab;
use super::*;
use crate::HeadWork;
use oblikovati_gui::GuiPartAssetKind;
use oblikovati_viewport::ViewportCapture;

const APPEARANCE: &str = "Appearance Browser";
const MODEL: &str = "Model";

#[given(
    "Model Browser is docked on the left and Material Browser and Appearance Browser share a right dock"
)]
fn docked_assets_with_long_values(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let browser = world.state().browser.unwrap();
    for (kind, options) in [
        (GuiPartAssetKind::Material, browser.materials),
        (GuiPartAssetKind::Appearance, browser.appearances),
    ] {
        let longest = options.iter().max_by_key(|asset| asset.name.len()).unwrap();
        world
            .app()
            .owner
            .assign_part_asset_from_gui(document, kind, &longest.id)
            .unwrap();
    }
    world.pump(Vec::new());
    super::restart_scenario::arrange_tabbed_panels(world);
    assert_ne!(
        diagnostics::window_dock_id(MODEL),
        diagnostics::window_dock_id(PANEL)
    );
    world.observed_counts = side_widths().map(|width| width.round() as usize).into();
}

fn side_widths() -> [f32; 2] {
    [MODEL, PANEL].map(|panel| diagnostics::window_geometry(panel).unwrap().1[0])
}

#[when("the user narrows the main application window")]
fn narrow_native_window(world: &mut ApplicationWorld) {
    drag_window_corner(world, [960, 650]);
    world
        .observed_counts
        .extend(side_widths().map(|width| width.round() as usize));
    drag_window_corner(world, [792, 580]);
}

fn drag_window_corner(world: &mut ApplicationWorld, target: [i32; 2]) {
    let [width, height] = world.extent().map(|value| value as i32);
    desktop_input::drag(world, [width - 2, height - 2], target);
    world.pump(Vec::new());
    let actual = world.extent().map(|pixel| pixel as i32);
    assert!(
        actual
            .into_iter()
            .zip(target)
            .all(|(size, goal)| size.abs_diff(goal) <= 4),
        "resized to {actual:?}, target {target:?}"
    );
}

#[then("left and right docks scale proportionally so the center viewport remains usable")]
fn docks_leave_a_viewport(world: &mut ApplicationWorld) {
    let width = world.extent()[0] as f32;
    let [left, _, right, _] = viewport(world);
    let after = side_widths();
    let &[initial_left, initial_right, middle_left, middle_right] = &world.observed_counts[..]
    else {
        panic!("expected both preparatory width measurements");
    };
    // Resize distributes remaining room in proportion to the preferred widths above the floor.
    let [left_scale, right_scale] = [[initial_left, middle_left], [initial_right, middle_right]]
        .map(|[before, middle]| (middle as f32 - 180.) / (before as f32 - 180.));
    assert!(
        (0. ..1.).contains(&left_scale) && (left_scale - right_scale).abs() <= 0.06,
        "proportions above the dock minimum: {left_scale} and {right_scale}"
    );
    // At the narrowest width the viewport reservation takes precedence over both dock minimums.
    assert!(
        after.iter().all(|size| *size < 180.) && (after[0] - after[1]).abs() <= 2.,
        "{after:?}"
    );
    assert!(
        (right - left) as f32 + 2. >= 0.55 * width,
        "center width {} in {width}",
        right - left
    );
    assert_eq!(world.state().document, world.previous_document);
    workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then(
    "dropdown labels inside the right dock truncate cleanly without colliding with dropdown chevrons"
)]
fn inspect_both_selectors(world: &mut ApplicationWorld) {
    for (panel, caption) in [(PANEL, "Material"), (APPEARANCE, "Appearance")] {
        select_tab(world, panel);
        let label = world.run_in_window(panel, caption);
        let cut = shortened_value(world, label);
        let arrow = selector_chevron(world, label);
        assert!(
            label.maximum[0] < cut.minimum[0] && cut.maximum[0] < arrow[0],
            "{label:?} {cut:?} {arrow:?}"
        );
        world.save_capture(&format!("resized-{caption}-selector"));
    }
}

fn shortened_value<'a>(
    world: &'a ApplicationWorld,
    label: &diagnostics::RenderedRun,
) -> &'a diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == label.list
                && run.minimum[0] > label.maximum[0]
                && (run.center()[1] - label.center()[1]).abs() < 4.
                && run.text.ends_with("...")
        })
        .unwrap_or_else(|| {
            panic!(
                "expected an ellipsized selector value beside {}: {:?}",
                label.text,
                world.text()
            )
        })
}

pub(in super::super) fn selector_chevron(
    world: &ApplicationWorld,
    label: &diagnostics::RenderedRun,
) -> [f32; 2] {
    let middle = label.center()[1];
    let point = world
        .frame
        .marks
        .iter()
        .filter(|mark| {
            let [x, y] = mark.position;
            mark.list == label.list && x > label.maximum[0] + 20. && (y - middle).abs() <= 4.
        })
        .max_by(|a, b| a.position[0].total_cmp(&b.position[0]))
        .unwrap_or_else(|| panic!("expected selector chevron after {label:?}"));
    [point.position[0] - 2., middle]
}

#[then(
    "inactive dock tab labels and dark/light material swatches remain clearly legible with visible swatch borders"
)]
fn check_tab_contrast_and_swatches(world: &mut ApplicationWorld) {
    for (theme, border) in [("Dark", [63, 69, 86]), ("Light", [174, 182, 198])] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        world.pump(Vec::new());
        for (active, inactive) in [(PANEL, APPEARANCE), (APPEARANCE, PANEL)] {
            select_tab(world, active);
            world.hover([400, 400]);
            assert_inactive_tab_readable(world, inactive);
        }
        check_swatch_palette(world, border);
        world.save_capture(&format!("resized-dock-{}", theme.to_lowercase()));
        world.escape();
    }
}

fn assert_inactive_tab_readable(world: &mut ApplicationWorld, panel: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(panel).unwrap();
    let prefix = if panel == APPEARANCE { "Appe" } else { "Mat" };
    assert!(
        world.frame.runs.iter().any(|row| {
            row.text.contains(prefix)
                && row.minimum[0] < max[0]
                && row.maximum[0] >= min[0]
                && (min[1]..max[1]).contains(&row.center()[1])
        }),
        "expected readable inactive {panel}: {:?}",
        world.text()
    );
    let capture = capture_framebuffer(&mut world.app().window.native);
    let fill = luminance(rgb_at(&capture, [min[0] + 3., (min[1] + max[1]) / 2.]));
    let count = (min[0] as i32 + 5..max[0] as i32 - 6)
        .flat_map(|x| (min[1] as i32 + 3..max[1] as i32 - 3).map(move |y| [x as f32, y as f32]))
        .filter(|&point| {
            let ink = luminance(rgb_at(&capture, point));
            (ink.max(fill) + 0.05) / (ink.min(fill) + 0.05) >= 4.5
        })
        .count();
    assert!(
        count >= 4,
        "{panel}: found {count} label pixels clearing 4.5:1"
    );
}

fn rgb_at(capture: &ViewportCapture, point: [f32; 2]) -> [u8; 3] {
    let [x, y] = diagnostics::pointer_pixels(point).map(|axis| axis as usize);
    let at = (y * capture.extent[0] as usize + x) * 4;
    capture.rgba[at..at + 3].try_into().unwrap()
}

fn luminance(rgb: [u8; 3]) -> f32 {
    rgb.into_iter()
        .zip([0.2126, 0.7152, 0.0722])
        .map(|(byte, weight)| {
            let srgb = f32::from(byte) / 255.;
            weight
                * if srgb <= 0.04045 {
                    srgb / 12.92
                } else {
                    ((srgb + 0.055) / 1.055).powf(2.4)
                }
        })
        .sum()
}

fn check_swatch_palette(world: &mut ApplicationWorld, border: [u8; 3]) {
    let label = world.run_in_window(APPEARANCE, "Appearance");
    world.click(diagnostics::pointer_pixels(selector_chevron(world, label)));
    world.pump(Vec::new());
    let browser = world.state().browser.unwrap();
    let capture = capture_framebuffer(&mut world.app().window.native);
    for name in ["ABS (Black)", "ABS (White)"] {
        let row = world.run(name);
        let fill = browser
            .appearances
            .iter()
            .find(|asset| asset.name == name)
            .unwrap()
            .swatch
            .unwrap();
        assert!(
            has_swatch_border(&capture, row, fill, border),
            "{name}: expected a border around the {fill:?} swatch"
        );
    }
}

fn has_swatch_border(
    capture: &ViewportCapture,
    row: &diagnostics::RenderedRun,
    fill: [u8; 3],
    border: [u8; 3],
) -> bool {
    let [x, y] = diagnostics::pointer_pixels([row.minimum[0], row.center()[1]]);
    let width = capture.extent[0] as usize;
    let pixel = |x: i32, y: i32| {
        let at = (y as usize * width + x as usize) * 4;
        &capture.rgba[at..at + 3]
    };
    (x - 32..x - 11).any(|left| {
        pixel(left, y) == border && pixel(left + 5, y) == fill && pixel(left + 11, y) == border
    })
}
