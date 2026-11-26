// SPDX-License-Identifier: GPL-2.0-only
//! Change the preview from the editor menu and verify the rendered sample.
use super::super::{asset_browser_shared, capture_framebuffer};
use super::*;

#[when("I open the preview options menu")]
fn open_options(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "PreviewOptions");
}

#[then("it contains Scene, Environment and Render Settings submenus")]
fn submenus(world: &mut ApplicationWorld) {
    for label in ["Scene", "Environment", "RenderSettings"] {
        menu_row(world, label);
    }
}

#[when("I choose Cube from the Scene submenu")]
fn choose_cube(world: &mut ApplicationWorld) {
    click_row(world, "Scene");
    assert!(world.menu_row_checked("Sphere"));
    click_row(world, "Cube");
}

#[then("the appearance preview shows a cube")]
fn cube_preview(world: &mut ApplicationWorld) {
    let capture = capture_framebuffer(&mut world.app().window.native);
    let (at, size) = diagnostics::window_geometry(EDITOR).unwrap();
    let [x, y] = diagnostics::pointer_pixels([at[0] + size[0] / 2., at[1] + 82.]);
    let sample = |dx, dy| capture_rgb(&capture, (x + dx) as usize, (y + dy) as usize)[0];
    let [top, left, right] = [sample(0, -29), sample(-29, 7), sample(23, 11)];
    assert!(
        top > right + 15 && right > left + 15,
        "expected three lit cube faces, got {top}, {left}, {right}"
    );
    assert!(
        sample(37, 16) > sample(47, 16) + 60,
        "expected the cube silhouette's straight right edge"
    );
    assert!(
        sample(37, -9) > sample(47, -9) + 60,
        "expected the same edge above the center"
    );
    clean_dialog(world);
}

#[when("I open the preview Environment submenu")]
fn open_environment(world: &mut ApplicationWorld) {
    open_options(world);
    click_row(world, "Environment");
}

#[then("it offers Grid Light, Plaza, Snowfield and Warm Light")]
fn environment_choices(world: &mut ApplicationWorld) {
    for label in ["GridLight", "Plaza", "Snowfield", "WarmLight"] {
        menu_row(world, label);
    }
    assert!(world.menu_row_checked("GridLight"));
}

#[when("I open the preview Render Settings submenu")]
fn open_render_settings(world: &mut ApplicationWorld) {
    click_row(world, "RenderSettings");
}

#[then("Quick is shown as the selected setting")]
fn quick_selected(world: &mut ApplicationWorld) {
    menu_row(world, "Quick");
    assert!(world.menu_row_checked("Quick"));
}

#[test]
fn warm_light_updates_preview_without_modifying_the_appearance() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    let before = capture_framebuffer(&mut world.app().window.native);
    open_environment(&mut world);
    click_row(&mut world, "WarmLight");
    let after = capture_framebuffer(&mut world.app().window.native);
    let warmed = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(before, after)| after[0] > before[0] + 15 && after[2] + 15 < before[2])
        .count();
    assert!(
        warmed > 900,
        "expected Warm Light to warm the preview, changed {warmed} pixels"
    );
    clean_dialog(&mut world);
    assert!(owned_asset(&mut world).appearance_metadata.is_none());
    open_environment(&mut world);
    assert!(world.menu_row_checked("WarmLight"));
}

fn click_row(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(menu_row(world, label).center());
    world.click(point);
    world.pump(Vec::new());
}

fn menu_row<'a>(world: &'a ApplicationWorld, label: &str) -> &'a diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .unwrap_or_else(|| panic!("expected a popup row {label:?} in {:?}", world.text()))
}
