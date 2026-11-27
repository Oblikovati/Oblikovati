// SPDX-License-Identifier: GPL-2.0-only
//! Verify the quarter-turn preview against a rendered RIGHT-view sketch.
use super::*;

#[test]
fn native_text_quarter_turn_rotates_right_view_preview_and_commits() {
    let mut world = ApplicationWorld::new();
    let anchor = open_right_view_text(&mut world);
    let vertical = preview_glyph_extent(&mut world, anchor);
    assert!(
        vertical[1] > vertical[0] * 2,
        "vertical glyphs {vertical:?}"
    );
    choose_quarter_turn(&mut world);
    text_format_choices::choose_format(&mut world, "Left", "Center");
    expect_upright_right_view(&mut world, anchor);
    let horizontal = preview_glyph_extent(&mut world, anchor);
    assert!(
        horizontal[0] > horizontal[1] * 2,
        "horizontal glyphs {horizontal:?}"
    );
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("sketch-text-right-view-quarter-turn");
    }
    commit_rotated_text(&mut world);
    world.close();
}

fn open_right_view_text(world: &mut ApplicationWorld) -> [i32; 2] {
    world.start_sketch_on("YZ Plane");
    let camera = super::super::super::viewport_pan::camera(world);
    let axes = crate::view_orientation::view_axes(&camera);
    assert_eq!(axes.toward, [1., 0., 0.]);
    activate_text(world);
    let [left, top, right, bottom] = world.viewport_bounds();
    let anchor =
        [left + (right - left) * 2 / 3, top + (bottom - top) * 3 / 4].map(|value| value as i32);
    open_editor(world, anchor);
    text_format_choices::choose_format(world, "Choose height", "6.10 mm");
    focus_content(world);
    for letter in b"abcde" {
        x11_input::press_keysym(u32::from(*letter));
        world.pump(Vec::new());
    }
    assert_eq!(world.state().sketch_text.unwrap().content, "abcde");
    anchor
}

fn choose_quarter_turn(world: &mut ApplicationWorld) {
    click_editor_button(world, "Chooserotation");
    for option in ["0°", "90°", "180°", "270°"] {
        world.run(option);
    }
    let position = world.run("270°").center();
    x11_input::click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
    assert_eq!(world.state().sketch_text.unwrap().rotation_degrees, "270");
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected live rotated Sketch Text preview");
    };
    assert_eq!(style.rotation, 270_f64.to_radians());
}

fn expect_upright_right_view(world: &mut ApplicationWorld, origin: [i32; 2]) {
    let [u, v] = world.sketch_point_of(origin);
    // At 270° the local baseline extends along -V, and glyphs rise along +U.
    let baseline = world.sketch_pixel_of([u, v - 0.5]);
    let ascent = world.sketch_pixel_of([u + 0.5, v]);
    assert!(baseline[0] > origin[0] && ascent[1] < origin[1]);
}

fn commit_rotated_text(world: &mut ApplicationWorld) {
    click_editor_button(world, "OK");
    x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_ready();
    let geometry = world.edited_geometry();
    assert_eq!(geometry.texts()[0].style.rotation, 270_f64.to_radians());
    assert_eq!(
        geometry.texts()[0].style.h_align,
        oblikovati_api::types::TEXT_ALIGN_CENTER
    );
}

fn preview_glyph_extent(world: &mut ApplicationWorld, anchor: [i32; 2]) -> [usize; 2] {
    let rgb = viewport_ink(world, &oblikovati_api::types::TOKEN_SKETCH_PREVIEW);
    let image = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let mut minimum = [usize::MAX; 2];
    let mut maximum = [0; 2];
    let width = image.extent[0] as usize;
    let [anchor_x, anchor_y] = anchor.map(|value| value as usize);
    for (y, row) in image
        .rgba
        .chunks_exact(width * 4)
        .enumerate()
        .skip(anchor_y.saturating_sub(100))
        .take(245)
    {
        for (x, pixel) in row
            .as_chunks::<4>()
            .0
            .iter()
            .enumerate()
            .skip(anchor_x.saturating_sub(85))
            .take(255)
        {
            if pixel[..3] == rgb {
                minimum = [minimum[0].min(x), minimum[1].min(y)];
                maximum = [maximum[0].max(x), maximum[1].max(y)];
            }
        }
    }
    assert!(
        minimum[0] < maximum[0] && minimum[1] < maximum[1],
        "expected visible preview pixels"
    );
    [maximum[0] - minimum[0], maximum[1] - minimum[1]]
}
