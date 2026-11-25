// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_visual_identity_uses_three_distinct_palettes_on_the_same_part() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let document = world.state().document.unwrap();
    let mut surfaces = Vec::new();
    let mut dock_tabs = Vec::new();

    for (theme, scheme) in [
        ("Dark", "Default"),
        ("Light", "Presentation"),
        ("High Contrast", "High Contrast"),
    ] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        oblikovati_api::client::Client::new(world.app().owner.document_host())
            .color_schemes()
            .set_active(&oblikovati_api::wire::SetColorSchemeArgs {
                name: scheme.into(),
            })
            .unwrap();
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
