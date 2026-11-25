// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_normal_modify_rows_fill_the_flyout_and_share_one_pitch() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    ribbon_panel_expansion::opened_unpinned(&mut world);
    let name = "##ribbon-additional-PartModify";
    let (position, size) = diagnostics::window_geometry(name).unwrap();
    let rows = ["MoveBodies", "BendPart", "CopyObject"]
        .map(|label| world.run_in_window(name, label).center());
    assert!(
        rows.windows(2)
            .all(|pair| (pair[1][1] - pair[0][1] - 32.).abs() < 1.)
    );
    world.save_capture("normal-modify-flyout");
    let right_side = diagnostics::pointer_pixels([position[0] + size[0] - 18., rows[0][1]]);
    world.click(right_side);
    assert!(diagnostics::window_geometry("Move Bodies").is_some());
    world.close();
}

#[test]
fn native_tab_strip_aligns_controls_and_separates_the_brand_mark() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    let (_, band) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert_eq!(band[1], 32. + 1. + 28. + 88.);
    let icons = ribbon_appearance::icon_quads::icon_quads_inside(([8., 0.], [95., 32.]));
    assert!(icons.len() >= 2);
    assert_eq!(icons[0].1[0] - icons[0].0[0], 16.);
    assert!(icons[1].0[0] - icons[0].1[0] >= 8.);
    world.save_capture("ribbon-chrome");
    world.close();
}
