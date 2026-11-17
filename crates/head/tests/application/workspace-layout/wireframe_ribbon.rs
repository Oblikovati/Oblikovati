// SPDX-License-Identifier: GPL-2.0-only
//! Read the application and ribbon rows from the completed native frame (#4062).
use super::{ApplicationWorld, diagnostics, quick_access_icons};
use cucumber::{given, then};

pub(super) fn assert_controls_above_tabs(world: &ApplicationWorld) {
    let file = world.run("File");
    let images = quick_access_icons::toolbar_images();
    assert_eq!(images.len(), 3, "expected default application-bar icons");
    assert!(images.iter().all(|image| image.center[1] < file.minimum[1]));
}

pub(super) fn assert_tabs_share_row(world: &ApplicationWorld) {
    let file = world.run("File");
    for label in ["3D Model", "Sketch", "View"] {
        let matches_tab = world.frame.runs.iter().any(|run| {
            run.text.replace(' ', "") == label.replace(' ', "")
                && file.maximum[0] < run.minimum[0]
                && (file.center()[1] - run.center()[1]).abs() < 12.
        });
        assert!(
            matches_tab,
            "{label}: expected a tab following File on its row"
        );
    }
}

pub(super) fn assert_panels_below_tabs(world: &ApplicationWorld) {
    let tab = world.run("3D Model");
    for label in ["Start 2D Sketch", "Extrude", "Sketch", "Create"] {
        let below = world.frame.runs.iter().any(|run| {
            run.text.replace(' ', "") == label.replace(' ', "") && run.minimum[1] > tab.maximum[1]
        });
        assert!(
            below,
            "{label:?}: expected a grouped command or panel below the tab row"
        );
    }
}

#[then("the application controls occupy the top row")]
fn application_controls_top(world: &mut ApplicationWorld) {
    assert_controls_above_tabs(world);
}
#[then("the ribbon tabs appear below the application controls")]
fn ribbon_tabs_below(world: &mut ApplicationWorld) {
    assert_controls_above_tabs(world);
    assert_tabs_share_row(world);
}
#[then("the active ribbon tab displays grouped commands below the tab row")]
fn active_panel_row(world: &mut ApplicationWorld) {
    assert_panels_below_tabs(world);
}

#[given("the quick access toolbar is configured to appear above the ribbon")]
fn quick_access_above(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##quick-access-below").is_none());
    assert_controls_above_tabs(world);
}
#[given("the ribbon is docked at the top in its expanded Normal appearance")]
fn normal_expanded(world: &mut ApplicationWorld) {
    assert_panels_below_tabs(world);
}

#[then("quick access remains in the top application bar alongside document context")]
fn application_bar_keeps_context(world: &mut ApplicationWorld) {
    assert_controls_above_tabs(world);
    let active = world.active_id();
    let document = world.document_name(active);
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == document && run.center()[1] < world.run("File").minimum[1]);
    assert!(run.is_some(), "expected active document in application bar");
}
#[then("File appears at the leading edge of the ribbon tab row beneath that bar")]
#[then("the ribbon tabs share that row with File")]
fn file_starts_tab_row(world: &mut ApplicationWorld) {
    assert_tabs_share_row(world);
}

fn tab_background(world: &mut ApplicationWorld, label: &str) -> [u8; 3] {
    let run = world.run(label).clone();
    let [x, y] = diagnostics::pointer_pixels([run.minimum[0] - 4., run.minimum[1] - 3.]);
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let offset = (y as usize * capture.extent[0] as usize + x as usize) * 4;
    capture.rgba[offset..offset + 3].try_into().unwrap()
}
#[then("the selected View tab is visually distinct from the other tabs")]
fn selected_tab_distinct(world: &mut ApplicationWorld) {
    let selected = tab_background(world, "View");
    let other = tab_background(world, "3D Model");
    assert_ne!(
        selected, other,
        "View tab must have a visible selected background"
    );
}
#[then("the View command panels occupy a separate row beneath the tabs")]
fn view_panels_below_tabs(world: &mut ApplicationWorld) {
    let tab_bottom = world.run("View").maximum[1];
    for label in ["Visibility", "Appearance", "Windows"] {
        assert!(
            world.run(label).minimum[1] > tab_bottom,
            "{label} panel must be below tabs"
        );
    }
}
#[then("each visible panel title sits below its commands")]
fn panel_titles_below_commands(world: &mut ApplicationWorld) {
    for (title, command) in [
        ("Visibility", "Show Sketches"),
        ("Appearance", "Theme"),
        ("Windows", "User Interface"),
        ("Navigate", "Home"),
    ] {
        let title = world.run(title);
        let matching = world.frame.runs.iter().filter(|run| {
            run.text.replace(' ', "") == command.replace(' ', "")
                && run.minimum[1] > world.run("View").maximum[1]
        });
        assert!(
            matching
                .into_iter()
                .any(|run| run.maximum[1] < title.minimum[1]),
            "{command} must be above panel title {}",
            title.text
        );
    }
}
