// SPDX-License-Identifier: GPL-2.0-only
//! Verify the controls and their actual draw geometry in the expanded ribbon.
use super::super::{diagnostics, ribbon_appearance::icon_quads::icon_quads_inside};
use super::layout_contract::{
    COMMANDS, TAB_GROUPS, labeled_run, resize, select_tab, wide_restored,
};
use super::*;

const RIBBON: &str = "##application-ribbon";
const SKETCH_SPLITS: [&str; 4] = ["Circle", "Center Arc", "3-Pt Rect.", "Line"];

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open with the full ribbon displayed")]
fn full_ribbon_on_part(world: &mut ApplicationWorld) {
    wide_restored(world);
    world.expect_text("Work Features");
}

#[when("the user inspects the 3D Model, Sketch, Tools, Manage, View, and Annotate ribbons")]
fn inspect_full_ribbons(world: &mut ApplicationWorld) {
    for (tab, groups) in TAB_GROUPS {
        select_tab(world, tab);
        world.expect_text(groups[0]);
        world.save_capture(&format!(
            "ribbon-{}",
            tab.to_ascii_lowercase().replace(' ', "-")
        ));
    }
}

#[then(
    "every split-button dropdown marker stays strictly inside its own command bounds without overlapping adjacent command icons"
)]
fn split_markers_fit(world: &mut ApplicationWorld) {
    select_tab(world, "3D Model");
    let plane = split_marker(world, "Plane");
    world.click(diagnostics::pointer_pixels(plane.center()));
    world.expect_text("Offset from Plane");
    world.escape();
    select_tab(world, "Sketch");
    for label in SKETCH_SPLITS {
        split_marker(world, label);
    }
}

fn split_marker<'a>(world: &'a ApplicationWorld, label: &str) -> &'a diagnostics::RenderedRun {
    let primary = command_run(world, label);
    let [_, y] = primary.center();
    let x = primary.minimum[0];
    // The caption begins 2 px after the icon's left padding and 3 px after the icon.
    let [[icon_left, _], _] = find_inline_icon(primary, x - 32., y);
    let host_right = icon_left - 2. + 108.;
    let marker = world.frame.runs.iter().find(|run| {
        run.list == primary.list
            && run.text == "▼"
            && (run.center()[1] - y).abs() < 5.
            && run.minimum[0] > primary.maximum[0]
            && run.maximum[0] < host_right
    });
    let marker = marker.unwrap_or_else(|| {
        panic!("{label}: missing split marker inside cell ending at {host_right}")
    });
    assert_marker_clears_icons(marker);
    marker
}

fn command_run<'a>(world: &'a ApplicationWorld, label: &str) -> &'a diagnostics::RenderedRun {
    let row = diagnostics::child_scroll(RIBBON, COMMANDS).expect("visible ribbon command row");
    labeled_run(world, row.list, label)
}

fn assert_marker_clears_icons(marker: &diagnostics::RenderedRun) {
    let icons = icon_quads_inside((
        marker.minimum.map(|v| v - 50.),
        marker.maximum.map(|v| v + 50.),
    ));
    assert!(
        icons.iter().all(|(min, max)| {
            (0..2)
                .any(|axis| max[axis] <= marker.minimum[axis] || min[axis] >= marker.maximum[axis])
        }),
        "dropdown at {marker:?} overlaps one of {icons:?}"
    );
}

fn find_inline_icon(run: &diagnostics::RenderedRun, left: f32, y: f32) -> [[f32; 2]; 2] {
    let pair = icon_quads_inside(([left, y - 13.], [run.minimum[0], y + 13.]))
        .into_iter()
        .next()
        .unwrap_or_else(|| panic!("missing icon beside {:?}", run.text));
    [pair.0, pair.1]
}

#[then(
    "single-command groups use full-height large buttons rather than leaving vertical empty space below a single row"
)]
fn single_commands_use_large_icons(world: &mut ApplicationWorld) {
    for (tab, labels) in [
        ("3D Model", &["Start 2D Sketch", "Simplify"][..]),
        ("Sketch", &["Project Geometry"][..]),
        ("Tools", &["Measure"][..]),
        ("Manage", &["Styles Editor", "Create iPart"][..]),
        ("View", &[super::sketch_visibility_label(world), "New"][..]),
    ] {
        select_tab(world, tab);
        for label in labels {
            assert_large_icon(world, label);
        }
    }
}

fn assert_large_icon(world: &ApplicationWorld, label: &str) {
    let run = command_run(world, label);
    let [x, y] = run.center();
    let icons = icon_quads_inside(([x - 24., y - 65.], [x + 24., run.minimum[1]]));
    assert!(
        icons
            .iter()
            .any(|(a, b)| b[1] - a[1] >= 36. && (a[0] + b[0] - x * 2.).abs() < 6.),
        "{label}: expected a full-height icon above its caption, got {icons:?}"
    );
}

#[then(
    "labeled ribbon combo boxes on Tools and View consistently place each label to the left of its selector"
)]
fn combo_labels_are_leading(world: &mut ApplicationWorld) {
    select_tab(world, "Tools");
    let state = world.state();
    let browser = state.browser.expect("active Part browser");
    let material = asset_name(&browser.materials, browser.part_material.as_deref());
    let appearance = asset_name(&browser.appearances, browser.part_appearance.as_deref());
    check_combo(world, "Material", material);
    check_combo(world, "Appearance", appearance);
    select_tab(world, "View");
    check_combo(world, "Theme", "Dark");
    check_combo(world, "Colour scheme", "Default");
}

fn asset_name<'a>(choices: &'a [oblikovati_gui::GuiAssetChoice], id: Option<&'a str>) -> &'a str {
    choices
        .iter()
        .find(|choice| Some(choice.id.as_str()) == id)
        .map_or_else(|| id.unwrap_or("Default"), |choice| choice.name.as_str())
}

fn check_combo(world: &ApplicationWorld, label: &str, preview: &str) {
    let label = command_run(world, label);
    let value = world.frame.runs.iter().find(|run| {
        run.list == label.list
            && run.minimum[0] > label.maximum[0]
            && (run.center()[1] - label.center()[1]).abs() < 5.
            && run.text == super::super::world::squeezed(preview)
    });
    assert!(
        value.is_some(),
        "no selector showing {preview:?} to the right of {label:?}"
    );
}

#[then(
    "the trailing ribbon display control menu opens clamped within the application window bounds"
)]
fn display_menu_is_clamped(world: &mut ApplicationWorld) {
    resize(world, 640);
    click_display_menu(world);
    let row = world.run("Cycle through All");
    let bounds @ [left, top, right, bottom] =
        diagnostics::draw_list_window_bounds(row.list).unwrap();
    let [width, height] = world.extent().map(|value| value as i32);
    let [left, top] = diagnostics::pointer_pixels([left, top]);
    let [right, bottom] = diagnostics::pointer_pixels([right, bottom]);
    assert!(
        left > 0 && top > 0 && right < width && bottom < height,
        "display menu {bounds:?} outside {width} x {height}"
    );
    world.save_capture("ribbon-display-policy-menu");
}
