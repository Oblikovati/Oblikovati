// SPDX-License-Identifier: GPL-2.0-only
//! Exercise live X11 display scaling through the native desktop and detached drawing.
use super::{ApplicationWorld, diagnostics, x11_input};
use x11rb::connection::Connection;
use x11rb::protocol::xproto::{AtomEnum, PropMode};
use x11rb::wrapper::ConnectionExt;

#[test]
#[ignore = "changes X11 screen resources on an isolated display"]
fn x11_display_dpi_scales_main_and_detached_windows_live() {
    let (connection, _) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[0].root;
    let set_dpi = |dpi| {
        let resources = format!("Xft.dpi:\t{dpi}\n");
        connection
            .change_property8(
                PropMode::REPLACE,
                root,
                AtomEnum::RESOURCE_MANAGER,
                AtomEnum::STRING,
                resources.as_bytes(),
            )
            .unwrap()
            .check()
            .unwrap();
    };

    set_dpi(144);
    let mut world = ApplicationWorld::new();
    world.add_part();
    expect_main_scale(&mut world, 1.5);
    let file = diagnostics::pointer_pixels(world.run("File").center());
    x11_input::click(file);
    world.pump(Vec::new());
    world.expect_text("Open...");
    world.escape();

    world.add_template(oblikovati_gui::GuiTemplateKind::Drawing);
    let drawing = world.active_id();
    let name = world.document_name(drawing);
    let tab = diagnostics::pointer_pixels(world.expect_tab(&name).center());
    let [left, top, right, bottom] = world.viewport_bounds();
    world.drag(
        tab,
        [((left + right) / 2) as i32, ((top + bottom) / 2) as i32],
    );
    world.pump(Vec::new());
    expect_detached_scale(&mut world, drawing, 1.5);

    for (dpi, scale) in [(192, 2.), (96, 1.)] {
        set_dpi(dpi);
        world.pump(Vec::new());
        expect_main_scale(&mut world, scale);
        expect_detached_scale(&mut world, drawing, scale);
    }
    world.close();
}

fn expect_main_scale(world: &mut ApplicationWorld, scale: f32) {
    assert_eq!(world.app().window.native.view.backing_scale(), scale);
    let [left, top, right, bottom] =
        diagnostics::window_pixel_bounds("##application-window-controls").unwrap();
    assert_eq!(bottom - top, 32. * scale);
    assert!((right - left - 3. * 46. * scale).abs() <= scale);
}

fn expect_detached_scale(world: &mut ApplicationWorld, document: u64, scale: f32) {
    world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(document, |view| {
            assert_eq!(view.backing_scale(), scale);
            // SAFETY: inspect retains the active detached drawing UI context.
            let io = unsafe { imgui::sys::igGetIO().as_ref() }.unwrap();
            assert_eq!(io.DisplayFramebufferScale.x, scale);
            assert_eq!(io.DisplayFramebufferScale.y, scale);
        })
        .expect("expected a detached drawing window");
}
