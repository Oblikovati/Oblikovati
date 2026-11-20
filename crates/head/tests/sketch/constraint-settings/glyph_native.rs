// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the horizontal fixture's glyph controls with native input (#4158).
use super::super::ApplicationWorld;

#[test]
fn horizontal_fixture_glyph_hide_show_and_hide_all_preserve_the_line() {
    let mut world = ApplicationWorld::new();
    open_horizontal_fixture(&mut world);
    let original = world.edited_geometry();
    let witness = world.line_point(0, 0.25);
    super::super::x11_input::focus_window();
    super::super::x11_input::press_keysym(0xffc5);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_annotation("H");
    choose_glyph_command(&mut world, "Hide");
    world.expect_no_annotation("H");
    assert_line_visible(&mut world, witness);
    super::super::x11_input::press_keysym(0xffc5);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_annotation("H");
    choose_glyph_command(&mut world, "Hide All Constraints");
    world.expect_no_annotation("H");
    assert_line_visible(&mut world, witness);
    assert_eq!(world.edited_geometry(), original);
    world.close();
}

fn open_horizontal_fixture(world: &mut ApplicationWorld) {
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/constraint-settings-line.opd");
    world.reopen_through_dialog(&path);
    world.pump(Vec::new());
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected saved sketch row")
        .center()
        .map(|value| value as i32);
    world.double_click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn choose_glyph_command(world: &mut ApplicationWorld, command: &str) {
    super::super::x11_input::focus_window();
    super::super::x11_input::right_click(world.exact_run_center("H"));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let position = super::super::diagnostics::pointer_pixels(world.run(command).center());
    super::super::x11_input::click(position);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn assert_line_visible(world: &mut ApplicationWorld, point: [i32; 2]) {
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let colors = [
        oblikovati_api::types::TOKEN_SKETCH_GEOMETRY,
        oblikovati_api::types::TOKEN_SKETCH_SELECTED,
    ];
    let expected = colors.map(|token| {
        let color = &palette[&token];
        [color.r, color.g, color.b]
    });
    let frame = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = frame.extent[0] as i32;
    let matches = (-3..=3).any(|y| {
        (-3..=3).any(|x| {
            let offset = (((point[1] + y) * width + point[0] + x) * 4) as usize;
            expected.iter().any(|color| {
                let error: f32 = frame.rgba[offset..offset + 3]
                    .iter()
                    .zip(color)
                    .map(|(channel, target)| (f32::from(*channel) / 255. - target).powi(2))
                    .sum();
                error < 0.03
            })
        })
    });
    if !matches {
        world.save_capture("constraint-glyph-line-color-4158");
        let offset = ((point[1] * width + point[0]) * 4) as usize;
        panic!(
            "expected sketch geometry color {expected:?} near {point:?}; center pixel {:?}",
            &frame.rgba[offset..offset + 4]
        );
    }
}
