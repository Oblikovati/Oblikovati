// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_visual_identity_uses_three_distinct_palettes_on_the_same_part() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let document = world.state().document.unwrap();
    let mut surfaces = Vec::new();
    let mut dock_tabs = Vec::new();

    for theme in ["Dark", "Light", "High Contrast"] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        world.pump(Vec::new());
        world.pump(Vec::new());
        assert_eq!(world.state().document, Some(document));
        let point = viewport_pan::bar_icon_center(viewport_pan::NavigationControl::Pan);
        let probe = diagnostics::pointer_pixels([point[0] - 9., point[1]]);
        surfaces.push(world.pixel_at(probe.map(|value| value as usize)));
        let (min, _) = diagnostics::window_dock_tab_bounds("Model").unwrap();
        let probe = diagnostics::pointer_pixels([min[0] + 3., min[1] + 7.]);
        dock_tabs.push(world.pixel_at(probe.map(|value| value as usize)));
        world.save_capture(&format!("identity-{theme}"));
    }

    assert_ne!(surfaces[0], surfaces[1]);
    assert_ne!(surfaces[0], surfaces[2]);
    assert_ne!(surfaces[1], surfaces[2]);
    assert!(dock_tabs[0][0] < 64);
    assert!(dock_tabs[1][0] > 192);
    assert!(dock_tabs[2][0] < 64);
    world.close();
}

#[test]
fn native_light_theme_refreshes_the_default_view_and_keeps_explicit_overrides() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    world.click_text("View");
    world.click_text("Dark");
    world.click_text("Light");
    let [left, top, _, _] = world.viewport_bounds().map(|value| value as usize);
    let sample = [left + 32, top + 48];
    let status = [140, world.extent()[1] as usize - 8];
    assert!(world.pixel_at(sample)[0] > 225);
    assert_eq!(&world.pixel_at(status)[..3], &[246, 247, 249]);
    world.save_capture("light-studio-default");

    select_scheme(&mut world, "High Contrast");
    assert_eq!(&world.pixel_at(sample)[..3], &[0; 3]);
    assert_eq!(&world.pixel_at(status)[..3], &[246, 247, 249]);
    world.save_capture("light-high-contrast-override");
    select_scheme(&mut world, "Default");
    assert!(world.pixel_at(sample)[0] > 225);
    world.close();
}

fn select_scheme(world: &mut ApplicationWorld, scheme: &str) {
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .color_schemes()
        .set_active(&oblikovati_api::wire::SetColorSchemeArgs {
            name: scheme.into(),
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[test]
fn native_feature_drafts_are_exclusive_and_extrude_actions_survive_a_short_panel() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    world.click_text("Chamfer");
    assert!(diagnostics::window_pixel_bounds("Chamfer").is_some());
    world.click_text("Extrude");
    assert!(diagnostics::window_pixel_bounds("Chamfer").is_none());
    let [_, top, right, bottom] = diagnostics::window_pixel_bounds("Extrude").unwrap();
    world.drag(
        [right as i32 - 2, bottom as i32 - 2],
        [right as i32 - 2, top as i32 + 138],
    );
    let [_, top, _, bottom] = diagnostics::window_pixel_bounds("Extrude").unwrap();
    assert!(
        bottom - top < 170.,
        "shortened Extrude bounds {top}..{bottom}"
    );
    world.run_in_window("Extrude", "OK");
    let cancel = world.run_in_window("Extrude", "Cancel").center();
    world.save_capture("extrude-short-properties");
    world.click(diagnostics::pointer_pixels(cancel));
    assert!(diagnostics::window_pixel_bounds("Extrude").is_none());

    world.click_text("Extrude");
    assert!(diagnostics::window_pixel_bounds("Extrude").is_some());
    world.click_text("Chamfer");
    assert!(diagnostics::window_pixel_bounds("Extrude").is_none());
    assert!(diagnostics::window_pixel_bounds("Chamfer").is_some());
    world.close();
}

#[test]
fn native_origin_plane_picker_draws_behind_navigation_and_enters_the_sketch() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    viewport_orientation::press_part(&mut world, |layout| layout.home);
    world.pump(vec![super::world::pointer_motion([700, 780])]);
    let [left, top, right, bottom] =
        diagnostics::window_pixel_bounds("##orientation-control").unwrap();
    let cube = [
        ((left + right) / 2.) as usize,
        ((top + bottom) / 2.) as usize,
    ];
    let face = world.pixel_at(cube);
    let nav = viewport_pan::bar_icon_center(viewport_pan::NavigationControl::Pan);
    let nav = diagnostics::pointer_pixels(nav).map(|value| value as usize);
    let icon = world.pixel_at(nav);
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    assert_eq!(world.pixel_at(nav), icon);
    assert_eq!(world.pixel_at(cube), face);
    let shown = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "XYPlane")
        .count();
    assert_eq!(shown, 2, "expected viewport label and browser row");
    world.save_capture("origin-plane-picker");
    world.click_text("XY Plane");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}

#[test]
fn native_sketch_ribbon_keeps_group_labels_and_the_last_create_row_visible() {
    let mut world = ApplicationWorld::new();
    world.start_sketch_on("XY Plane");
    let [_, top, _, _] = world.viewport_bounds();
    for label in [
        "Line",
        "Project Geometry",
        "Finish Sketch",
        "Constrain",
        "Modify",
        "Insert",
        "Format",
    ] {
        let run = world.run(label);
        let bottom = diagnostics::pointer_pixels(run.maximum)[1];
        assert!(
            bottom < top as i32,
            "{label}: {bottom} should be above the viewport at {top}"
        );
    }
    world.save_capture("sketch-ribbon-compaction");
    world.click_text("Finish Sketch");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.close();
}
