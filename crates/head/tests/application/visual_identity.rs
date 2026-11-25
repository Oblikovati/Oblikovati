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

#[test]
fn native_feature_properties_align_blend_fields_and_keep_extrude_directions_in_one_row() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    world.click_text("Chamfer");
    let (position, size) = diagnostics::window_geometry("Chamfer").unwrap();
    let label = world.run_in_window("Chamfer", "Distance").center();
    let value = world.run_in_window("Chamfer", "1mm").center();
    assert!(label[0] < position[0] + size[0] * 0.4);
    assert!(value[0] > position[0] + size[0] * 0.4);
    assert!((label[1] - value[1]).abs() < 2.);
    world.save_capture("chamfer-properties");

    world.click_text("Extrude");
    let row = ["Default", "Flipped", "Symmetric", "Asymmetric"]
        .map(|name| world.run_in_window("Extrude", name).center());
    assert!(
        row.windows(2)
            .all(|pair| { pair[0][0] < pair[1][0] && (pair[0][1] - pair[1][1]).abs() < 2. })
    );
    world.save_capture("extrude-properties");
    world.close();
}

#[test]
fn native_title_bar_centers_the_document_and_highlights_the_close_control() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let center = world.run("— Oblikovati").center();
    assert!(world.shows("<Primary> — Oblikovati"));
    assert!((center[0] - world.extent()[0] as f32 / 2.).abs() < 2.);
    let [left, top, right, bottom] =
        diagnostics::window_pixel_bounds("##application-window-controls").unwrap();
    assert_eq!(right - left, 3. * 46.);
    assert_eq!(bottom - top, 32.);
    world.hover([right as i32 - 23, top as i32 + 16]);
    assert_eq!(
        &world.pixel_at([right as usize - 9, top as usize + 8])[..3],
        &[225, 29, 72]
    );
    world.save_capture("centered-title-bar");
    world.close();
}

#[test]
fn native_dock_indicator_follows_the_active_asset_tab() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let panel_names = ["Material Browser", "Appearance Browser"];
    for name in panel_names {
        let (left, right) = diagnostics::window_dock_tab_bounds(name).unwrap();
        let center = std::array::from_fn(|axis| (left[axis] + right[axis]) / 2.);
        world.click(diagnostics::pointer_pixels(center));
        assert_eq!(diagnostics::window_dock_tab_visible(name), Some(true));
        for tab in panel_names {
            let (min, max) = diagnostics::window_dock_tab_bounds(tab).unwrap();
            let at = diagnostics::pointer_pixels([min[0] + 4., max[1] - 1.]);
            let color = world.pixel_at(at.map(|value| value as usize));
            assert_eq!(
                &color[..3] == [129, 114, 250],
                tab == name,
                "{tab} {color:?}"
            );
        }
        world.save_capture(name);
    }
    world.close();
}

#[test]
fn native_studio_colors_keep_cube_labels_readable_and_drawing_paper_neutral() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    assert!(world.run("FRONT").color[0] > 200);
    world.app().owner.select_theme_from_gui("Light").unwrap();
    world.pump(Vec::new());
    assert!(world.run("FRONT").color[0] < 80);
    world.save_capture("light-viewcube");
    world.add_template(oblikovati_gui::GuiTemplateKind::Drawing);
    assert!(diagnostics::window_dock_tab_bounds("Appearance Browser").is_none());
    let [left, top, right, bottom] = world.viewport_bounds().map(|side| side as usize);
    let center = [(left + right) / 2, (top + bottom) / 2];
    assert_eq!(&world.pixel_at(center)[..3], &[255; 3]);
    world.app().owner.select_theme_from_gui("Dark").unwrap();
    world.pump(Vec::new());
    assert_eq!(&world.pixel_at(center)[..3], &[248, 249, 251]);
    world.save_capture("dark-drawing-paper");
    world.close();
}

#[test]
fn native_document_strip_and_status_bar_keep_labels_clear() {
    let mut world = ApplicationWorld::new();
    workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    let tabs = diagnostics::window_pixel_bounds("##open-document-tabs").unwrap();
    let status = diagnostics::window_pixel_bounds("##status-bar").unwrap();
    assert_eq!(tabs[3] - tabs[1], 26.);
    assert_eq!(status[3] - status[1], 24.);
    let [left, top, right, bottom] = world.viewport_bounds();
    world.hover([((left + right) / 2) as i32, ((top + bottom) / 2) as i32]);
    let run = world.frame.runs.iter().find(|run| run.text.ends_with('%')).unwrap();
    world.run_in_window("##status-bar", "mm");
    assert!(
        world.frame.runs.iter().any(|run| {
            run.center()[1] > status[1] && run.text.starts_with('X') && run.text.contains('Y')
        }),
        "the viewport pointer must have model coordinates"
    );
    let zoom = run.text.clone();
    let mut camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    camera.scale *= 0.5;
    crate::HeadWindow::set_view_camera(&mut world.app().window, camera).unwrap();
    world.pump(Vec::new());
    world.run_in_window("##status-bar", "200%");
    assert_eq!(zoom, "100%");
    world.save_capture("document-strip-and-status-telemetry");
    world.close();
}
