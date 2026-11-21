// SPDX-License-Identifier: GPL-2.0-only
//! Verify actual frame icon pixels and the platform accessibility tree.
use super::*;
use cucumber::then;

#[then("the controls are distinguishable by their icons and accessible names")]
fn frame_icons_and_native_names(world: &mut ApplicationWorld) {
    let (position, size) = assert_drawn_frame_controls(world);
    assert_platform_names(world, position, size);
}

/// Verify the three rendered frame controls, for example at the minimum window size.
pub(super) fn assert_drawn_frame_controls(world: &mut ApplicationWorld) -> ([f32; 2], [f32; 2]) {
    let height = world.app().window.native.drawable_extent().unwrap()[1] as i32;
    super::x11_input::move_pointer([5, height - 30]);
    world.pump(Vec::new());
    let (position, size) = super::window_geometry("##application-window-controls")
        .expect("expected the frame controls");
    let masks = icon_masks(world, position, size);
    assert!(
        masks.iter().all(|mask| mask.iter().any(|pixel| *pixel)),
        "each control must draw an icon"
    );
    assert!(
        masks[0] != masks[1] && masks[1] != masks[2] && masks[0] != masks[2],
        "frame icons must differ"
    );
    (position, size)
}

fn icon_masks(world: &mut ApplicationWorld, position: [f32; 2], size: [f32; 2]) -> Vec<Vec<bool>> {
    let capture = world
        .app()
        .window
        .native
        .capture_window()
        .expect("capture the presented application");
    let width = size[0] / 3.;
    (0..3)
        .map(|index| {
            let center = [
                position[0] + (index as f32 + 0.5) * width,
                position[1] + size[1] * 0.5,
            ];
            let background = pixel(
                &capture,
                super::pointer_pixels([position[0] + index as f32 * width + 3., center[1]]),
            );
            (-8..=8)
                .flat_map(|y| (-8..=8).map(move |x| [x, y]))
                .map(|[x, y]| {
                    let color = pixel(
                        &capture,
                        super::pointer_pixels([center[0] + x as f32, center[1] + y as f32]),
                    );
                    (0..3).any(|channel| color[channel].abs_diff(background[channel]) > 32)
                })
                .collect()
        })
        .collect()
}
fn pixel(capture: &oblikovati_viewport::ViewportCapture, point: [i32; 2]) -> [u8; 3] {
    let [x, y] = point.map(|value| usize::try_from(value).expect("nonnegative frame point"));
    assert!(
        x < capture.extent[0] as usize && y < capture.extent[1] as usize,
        "frame point outside capture"
    );
    let index = (y * capture.extent[0] as usize + x) * 4;
    [
        capture.rgba[index],
        capture.rgba[index + 1],
        capture.rgba[index + 2],
    ]
}
fn assert_platform_names(world: &mut ApplicationWorld, position: [f32; 2], size: [f32; 2]) {
    let origin = world
        .app()
        .window
        .native
        .view
        .window_screen_bounds()
        .expect("native screen bounds");
    let width = size[0] / 3.;
    let mut command = std::process::Command::new("/usr/bin/python3");
    command.arg(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/tests/accessibility_atspi/assert_names.py"
    ));
    command.arg(std::process::id().to_string());
    for index in 0..3 {
        let min = super::pointer_pixels([position[0] + index as f32 * width, position[1]]);
        let max = super::pointer_pixels([
            position[0] + (index + 1) as f32 * width,
            position[1] + size[1],
        ]);
        for value in [
            origin[0] + min[0],
            origin[1] + min[1],
            max[0] - min[0],
            max[1] - min[1],
        ] {
            command.arg(value.to_string());
        }
    }
    let output = command
        .output()
        .expect("AT-SPI client requires Python GI and a session accessibility bus");
    assert!(
        output.status.success(),
        "AT-SPI client failed: {}",
        String::from_utf8_lossy(&output.stderr)
    );
}
