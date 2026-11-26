// SPDX-License-Identifier: GPL-2.0-only
//! Observe the File menu's persistent state, row geometry and composited shadow.
use super::super::ribbon_appearance::icon_quads::icon_quads_inside;
use super::super::{capture_framebuffer, diagnostics};
use super::*;
use oblikovati_viewport::ViewportCapture;

const POPUP: &str = "##Menu_00";

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open and active in the MCAD Software")]
fn saved_part_with_closed_menu(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    super::super::viewport_orientation::open_saved_navigation_block(world, &path);
    world.hover([1000, 600]);
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
}

#[then("the File tab displays an active highlight while the menu overlay is open")]
fn file_trigger_is_active(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(POPUP).is_some());
    world.hover([1000, 600]);
    let file = world.run("File");
    let point = [file.minimum[0] - 3., file.center()[1]];
    let active = capture_framebuffer(&mut world.app().window.native);
    let before = world.last_capture.as_ref().unwrap();
    assert_ne!(pixel(before, point), pixel(&active, point));
    let [r, g, b, _] = pixel(&active, point);
    assert!(
        b > r + 60 && r > g,
        "expected the active brand fill, saw {r}/{g}/{b}"
    );
}

#[then(
    "Options and Exit display leading icons aligned to the same row height and text column as New through Close"
)]
fn footer_matches_command_rows(world: &mut ApplicationWorld) {
    let command_rows = ["New", "Open...", "Save", "Save As", "Close"];
    let footer_rows = ["Options...", "Exit"];
    let [commands, footer] = [command_rows.as_slice(), footer_rows.as_slice()]
        .map(|names| names.iter().map(|name| world.run(name)).collect::<Vec<_>>());
    for rows in [&commands, &footer] {
        for pair in rows.windows(2) {
            let pitch = pair[1].center()[1] - pair[0].center()[1];
            assert!((pitch - 32.).abs() <= 1., "File row pitch: {pitch}");
        }
    }
    let icon_left = row_icon_left(commands[0]);
    for row in commands.iter().chain(&footer) {
        assert!(
            (row.minimum[0] - commands[0].minimum[0]).abs() <= 2.,
            "unaligned text {}",
            row.text
        );
        assert!(
            (row_icon_left(row) - icon_left).abs() <= 1.,
            "unaligned icon {}",
            row.text
        );
    }
}

fn row_icon_left(row: &diagnostics::RenderedRun) -> f32 {
    let [x, y] = [row.minimum[0], row.center()[1]];
    let icons = icon_quads_inside(([x - 35., y - 11.], [x - 2., y + 11.]));
    let [(min, max)] = icons.as_slice() else {
        panic!("{}: expected one leading icon, saw {icons:?}", row.text);
    };
    assert_eq!([max[0] - min[0], max[1] - min[1]], [16.; 2]);
    min[0]
}

#[then(
    "the currently active Recent ordering option is visually distinguished from inactive ordering controls"
)]
fn ordering_retains_its_accent(world: &mut ApplicationWorld) {
    for (ordering, inactive) in [
        ("By Ordered List", "By Access Date"),
        ("By Access Date", "By Ordered List"),
    ] {
        world.expect_no_text(inactive);
        let row = world.run(ordering).clone();
        let [x, y] = row.center();
        let capture = capture_framebuffer(&mut world.app().window.native);
        let accent =
            unsafe { imgui::sys::igGetColorU32_Col(imgui::sys::ImGuiCol_NavHighlight as i32, 1.) }
                .to_le_bytes();
        assert!(
            (2..=9).any(|offset| {
                let at = row.maximum[1] + offset as f32;
                [-10., 0., 10.]
                    .into_iter()
                    .all(|dx| pixel(&capture, [x + dx, at])[..3] == accent[..3])
            }),
            "{ordering}: expected an accent underline while the pointer is away"
        );
        world.click(diagnostics::pointer_pixels([x, y]));
        world.hover([1000, 600]);
    }
}

#[then("the File menu overlay renders with floating elevation shadow over the workspace")]
fn shadow_darkens_the_workspace(world: &mut ApplicationWorld) {
    let (origin, size) = diagnostics::window_geometry(POPUP).expect("expected the File popup");
    let [x, y] = [origin[0] + size[0], origin[1] + size[1]];
    let open = capture_framebuffer(&mut world.app().window.native);
    let closed = world.last_capture.as_ref().unwrap();
    for positions in [
        [1., 6., 17.].map(|offset| [x + offset, y - 35.]),
        [1., 9., 22.].map(|offset| [origin[0] + size[0] * 0.75, y + offset]),
    ] {
        let difference = positions.map(|point| {
            let [before, after] = [closed, &open].map(|image| pixel(image, point)[1]);
            i16::from(before) - i16::from(after)
        });
        assert!(
            difference[0] > difference[1] && difference[1] > 0 && difference[2] == 0,
            "expected shadow falloff outside the File menu at {positions:?}, saw {difference:?}"
        );
    }
    world.save_capture("file-menu-active-values-and-shadow");
}

fn pixel(image: &ViewportCapture, point: [f32; 2]) -> [u8; 4] {
    let [x, y] = diagnostics::pointer_pixels(point).map(|axis| axis as usize);
    let index = (y * image.extent[0] as usize + x) * 4;
    image.rgba[index..index + 4].try_into().unwrap()
}
