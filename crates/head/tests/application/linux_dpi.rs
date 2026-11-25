// SPDX-License-Identifier: GPL-2.0-only
//! Exercise live X11 display scaling through the native desktop and detached drawing.
use super::{ApplicationWorld, diagnostics, x11_input};
use crate::HeadWork;
use x11rb::connection::Connection;
use x11rb::protocol::xproto::{
    AtomEnum, ConnectionExt as _, CreateWindowAux, PropMode, WindowClass,
};
use x11rb::wrapper::ConnectionExt;

#[test]
#[ignore = "changes X11 screen resources on an isolated display"]
fn x11_display_dpi_scales_main_and_detached_windows_live() {
    let (connection, _) = x11rb::connect(None).unwrap();
    let root = connection.setup().roots[0].root;
    let selection = setting_atom(&connection, b"_XSETTINGS_S0");
    let property = setting_atom(&connection, b"_XSETTINGS_SETTINGS");
    let manager = connection.generate_id().unwrap();
    connection
        .create_window(
            x11rb::COPY_DEPTH_FROM_PARENT,
            manager,
            root,
            0,
            0,
            1,
            1,
            0,
            WindowClass::INPUT_ONLY,
            0,
            &CreateWindowAux::new(),
        )
        .unwrap()
        .check()
        .unwrap();
    connection
        .set_selection_owner(manager, selection, x11rb::CURRENT_TIME)
        .unwrap()
        .check()
        .unwrap();
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

    write_xsettings_dpi(&connection, manager, property, 144);
    let mut world = ApplicationWorld::new();
    world.add_part();
    expect_main_scale(&mut world, 1.5, 1.25);
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
    expect_detached_scale(&mut world, drawing, 1.5, 1.25);

    let explicit = oblikovati_model::ui_scale::UiScaleOptions::new(1., 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(explicit).unwrap();
    world.pump(Vec::new());
    expect_main_scale(&mut world, 1.5, 1.5);
    expect_detached_scale(&mut world, drawing, 1.5, 1.5);
    world
        .app()
        .owner
        .set_ui_scale_from_gui(explicit.with_automatic_font())
        .unwrap();

    write_xsettings_dpi(&connection, manager, property, 192);
    world.pump(Vec::new());
    expect_main_scale(&mut world, 2., 1.5);
    expect_detached_scale(&mut world, drawing, 2., 1.5);

    for (dpi, backing, layout) in [(144, 1.5, 1.25), (192, 2., 1.5), (96, 1., 1.)] {
        set_dpi(dpi);
        world.pump(Vec::new());
        expect_main_scale(&mut world, backing, layout);
        expect_detached_scale(&mut world, drawing, backing, layout);
    }
    connection
        .delete_property(root, AtomEnum::RESOURCE_MANAGER.into())
        .unwrap()
        .check()
        .unwrap();
    world.pump(Vec::new());
    expect_main_scale(&mut world, 2., 1.5);
    expect_detached_scale(&mut world, drawing, 2., 1.5);

    connection.destroy_window(manager).unwrap().check().unwrap();
    world.pump(Vec::new());
    expect_main_scale(&mut world, 1., 1.);
    expect_detached_scale(&mut world, drawing, 1., 1.);
    world.close();
}

fn setting_atom(connection: &impl Connection, name: &[u8]) -> u32 {
    connection
        .intern_atom(false, name)
        .unwrap()
        .reply()
        .unwrap()
        .atom
}

fn write_xsettings_dpi(connection: &impl Connection, manager: u32, property: u32, dpi: u32) {
    let mut value = b"\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x07\x00Xft/DPI\x00\x00\x00\x00\x00".to_vec();
    value.extend_from_slice(&(dpi * 1024).to_le_bytes());
    connection
        .change_property8(PropMode::REPLACE, manager, property, property, &value)
        .unwrap()
        .check()
        .unwrap();
}

fn expect_main_scale(world: &mut ApplicationWorld, backing: f32, layout: f32) {
    assert_eq!(world.app().window.native.view.backing_scale(), backing);
    let [left, top, right, bottom] =
        diagnostics::window_pixel_bounds("##application-window-controls").unwrap();
    assert_eq!(bottom - top, 32. * layout);
    assert!((right - left - 3. * 46. * layout).abs() <= layout);
}

fn expect_detached_scale(world: &mut ApplicationWorld, document: u64, backing: f32, layout: f32) {
    world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(document, |view| {
            assert_eq!(view.backing_scale(), backing);
            // SAFETY: inspect retains the active detached drawing UI context.
            let io = unsafe { imgui::sys::igGetIO().as_ref() }.unwrap();
            assert_eq!(io.DisplayFramebufferScale.x, layout);
            assert_eq!(io.DisplayFramebufferScale.y, layout);
        })
        .expect("expected a detached drawing window");
}
