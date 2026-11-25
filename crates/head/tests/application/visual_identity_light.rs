// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_light_properties_and_read_only_theme_values_use_studio_contrast() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    world.app().owner.select_theme_from_gui("Light").unwrap();
    world.pump(Vec::new());
    world.click_text("Chamfer");
    let accept = world.run_in_window("Chamfer", "OK");
    assert_eq!(accept.color[..3], [0x6B, 0x72, 0x80]);
    let point = diagnostics::pointer_pixels(accept.center());
    world.click(point);
    assert!(diagnostics::window_geometry("Chamfer").is_some());
    world.save_capture("light-disabled-chamfer");
    let cancel = diagnostics::pointer_pixels(world.run_in_window("Chamfer", "Cancel").center());
    world.click(cancel);

    world.click_text("View");
    world.click_text("Edit Theme");
    let theme = "Theme editor";
    let label = world.run_in_window(theme, "Windowbackground");
    assert_eq!(label.color[..3], [0x4B, 0x52, 0x63]);
    let value = world.run_in_window(theme, "R:246");
    assert_eq!(value.color[..3], [0x4B, 0x52, 0x63]);
    let field_label = world.run_in_window(theme, "Newname").center();
    let duplicate = world.run_in_window(theme, "Duplicate").center();
    assert!(field_label[0] + 230. < duplicate[0]);
    world.save_capture("light-theme-editor");
    world.close();
}
