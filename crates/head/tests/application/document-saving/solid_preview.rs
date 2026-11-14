// SPDX-License-Identifier: GPL-2.0-only
//! First-save preview of an unsaved solid, driven through the native ribbon.
use super::*;

#[given("an unsaved part containing a visible rectangular solid is active in the MCAD Software")]
fn unsaved_solid(world: &mut ApplicationWorld) {
    let document = world.add_part();
    world.start_sketch();
    world.click_text("Rectangle");
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let origin = world
        .app()
        .window
        .native
        .view
        .sketch_pixel(edit.plane, [0., 0.])
        .unwrap();
    world.click(origin);
    world.pump(vec![oblikovati_viewport::ViewportEvent::Input(
        oblikovati_viewport::ViewportInput::Motion {
            position: [1000, 600],
            modifiers: Default::default(),
        },
    )]);
    for key in [
        oblikovati_viewport::ViewportPhysicalKey::Tab,
        oblikovati_viewport::ViewportPhysicalKey::Enter,
    ] {
        world.type_text("10 mm");
        world.key(key, false);
    }
    world.finish_sketch();
    world.click_text("Extrude");
    let (position, _) = super::super::diagnostics::window_geometry("Extrude").unwrap();
    let distance =
        super::super::diagnostics::pointer_pixels([position[0] + 70., position[1] + 85.]);
    world.click(distance);
    world.type_text("10 mm");
    world.click_text("Apply");
    world.pump(Vec::new());
    assert!(world.state().document_storage[&document].path.is_none());
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
    world.observed_view_camera = Some(super::super::viewport_pan::camera(world));
}

#[given("the first-save preview settings are configured to save an isometric component view")]
fn default_preview_settings(world: &mut ApplicationWorld) {
    world.press_save_shortcut();
    assert!(world.checked("Save Preview Picture"));
    assert!(world.radio_selected(ICONIC));
    world.click_text("Cancel");
    world.expect_no_text(SAVE_IN);
}

#[then("the Save As dialog opens with a proposed part filename selected")]
fn save_dialog_with_proposal(world: &mut ApplicationWorld) {
    save_as_opens(world);
    proposed_selected(world);
}

#[then("Save Preview Picture is checked")]
fn preview_is_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Save Preview Picture"));
}

#[then("the preview area displays a shaded isometric block")]
fn preview_is_shaded(world: &mut ApplicationWorld) {
    world.expect_no_text("Picture captured on save");
    world.expect_no_text(NO_PREVIEW);
    assert_eq!(
        super::super::viewport_pan::camera(world),
        world.observed_view_camera.unwrap(),
        "the preview must not change the active camera"
    );
    let checkbox = world.run("Save Preview Picture").clone();
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let left = checkbox.minimum[0] as usize;
    let bottom = checkbox.minimum[1] as usize - 8;
    let top = bottom.saturating_sub(90);
    let mut colours = std::collections::BTreeSet::new();
    let mut solid_shades = std::collections::BTreeSet::new();
    for y in top..bottom {
        for x in left..(left + 220).min(width) {
            let offset = (y * width + x) * 4;
            colours.insert([
                capture.rgba[offset] / 8,
                capture.rgba[offset + 1] / 8,
                capture.rgba[offset + 2] / 8,
            ]);
            let pixel = &capture.rgba[offset..offset + 4];
            if super::super::solid_visibility::neutral_fixture_hue(pixel) {
                solid_shades.insert(pixel[0] / 8);
            }
        }
    }
    assert!(
        colours.len() >= 4,
        "expected shaded preview, found {} colours",
        colours.len()
    );
    assert!(
        solid_shades.len() >= 2,
        "expected differently shaded solid faces"
    );
    world.save_capture("save-isometric-preview");
}

#[test]
fn first_save_keeps_the_isometric_picture_for_reopening() {
    let mut world = ApplicationWorld::new();
    unsaved_solid(&mut world);
    world.press_save_shortcut();
    preview_is_shaded(&mut world);
    let path = world.accept_save_as();
    let picture = oblikovati_persistence::package::preview_picture_of_file(&path)
        .expect("expected the saved part's preview picture");
    let colours: std::collections::BTreeSet<_> = picture.rgb.iter().copied().collect();
    assert!(colours.len() >= 4, "expected a shaded saved picture");
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    super::super::workspace_layout::browser_feature::expect_solid_visible(&mut world);
    world.close();
}
