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
