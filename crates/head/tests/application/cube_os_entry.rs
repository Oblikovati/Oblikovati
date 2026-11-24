// SPDX-License-Identifier: GPL-2.0-only
//! Native input for the parameter-driven cube journey (#3857).
use super::{ApplicationWorld, diagnostics, x11_input};

fn click_label(world: &mut ApplicationWorld, label: &str) {
    let center = super::quick_access_icons::standard_icon_center(label)
        .unwrap_or_else(|| world.run(label).center());
    let point = diagnostics::pointer_pixels(center);
    x11_input::click(point);
    world.pump(Vec::new());
}

#[test]
fn native_input_drives_parameter_cube_through_reopen() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    for label in ["New", "Metric", "Standard metric part", "Create"] {
        click_label(&mut world, label);
    }
    assert_eq!(world.state().documents.len(), 1);
    create_size_parameter(&mut world);
    let edit = draw_parameter_rectangle(&mut world);
    let document = extrude_parameter_rectangle(&mut world);
    show_cube(&mut world);
    let keys = super::super::cube::topology_keys(&mut world.app().owner);
    edit_cube_and_history(&mut world, edit, document, &keys);
    save_reopen_cube(&mut world, &keys);
    world.close();
}

fn draw_parameter_rectangle(world: &mut ApplicationWorld) -> crate::HeadSketchEdit {
    click_label(world, "Start 2D Sketch");
    click_label(world, "XY Plane");
    let edit = world
        .app()
        .owner
        .active_sketch_edit()
        .expect("expected XY sketch edit");
    assert_eq!(edit.plane, oblikovati_model::sketch::SketchPlane::xy());
    click_label(world, "Rectangle");
    place_rectangle_with_size(world);
    anchor_rectangle_at_origin(world);
    super::super::parameters::verify_rectangle(&world.app().owner, edit, 1.);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    click_label(world, "Finish Sketch");
    assert!(world.app().owner.active_sketch_edit().is_none());
    assert_eq!(world.state().browser.unwrap().sketches.len(), 1);
    edit
}

fn place_rectangle_with_size(world: &mut ApplicationWorld) {
    let point = world.sketch_pixel_of([3., 3.]);
    x11_input::click(point);
    world.pump(Vec::new());
    x11_input::move_pointer(world.sketch_pixel_of([4., 4.]));
    world.pump(Vec::new());
    for key in [0xff09, 0xff0d] {
        type_text(world, "size");
        x11_input::press_keysym(key);
        world.pump(Vec::new());
    }
}

fn anchor_rectangle_at_origin(world: &mut ApplicationWorld) {
    // The default current-command inference scope excludes the projected origin.
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    if !world.shows("Coincident") {
        x11_input::click(super::ribbon_display::compact_panel_point(
            world,
            "Constrain",
        ));
        world.pump(Vec::new());
    }
    click_label(world, "Coincident");
    for point in [world.sketch_pixel_of([0., 0.]), world.line_point(0, 0.)] {
        x11_input::move_pointer(point);
        world.pump(Vec::new());
        x11_input::click(point);
        world.pump(Vec::new());
    }
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
}

fn extrude_parameter_rectangle(world: &mut ApplicationWorld) -> u64 {
    click_label(world, "Extrude");
    let distance = world.run_in_window("Extrude", "DistanceA").center();
    let field = diagnostics::pointer_pixels([distance[0] - 100., distance[1]]);
    x11_input::click(field);
    world.pump(Vec::new());
    type_text(world, "size");
    assert!(world.state().extrude_preview_active);
    click_label(world, "OK");
    let document = active_document(world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    super::super::cube::verify(&mut world.app().owner, document, 1.);
    document
}

fn active_document(world: &mut ApplicationWorld) -> u64 {
    world
        .app()
        .owner
        .documents()
        .workspace()
        .active_document()
        .unwrap()
}

fn edit_cube_and_history(
    world: &mut ApplicationWorld,
    edit: crate::HeadSketchEdit,
    document: u64,
    keys: &std::collections::BTreeSet<String>,
) {
    edit_size_parameter(world);
    super::super::parameters::verify_rectangle(&world.app().owner, edit, 2.);
    super::super::cube::verify(&mut world.app().owner, document, 2.);
    for (label, side) in [("Undo", 1.), ("Redo", 2.)] {
        click_label(world, label);
        super::super::cube::verify(&mut world.app().owner, document, side);
        assert_eq!(
            &super::super::cube::topology_keys(&mut world.app().owner),
            keys
        );
    }
}

fn save_reopen_cube(world: &mut ApplicationWorld, keys: &std::collections::BTreeSet<String>) {
    let directory = tempfile::Builder::new()
        .prefix(&format!("obk-cube-{}-", std::process::id()))
        .rand_bytes(0)
        .tempdir()
        .unwrap();
    let path = directory.path().join("cube.opd");
    save_cube_file(world, &path);
    let reopened = reopen_cube_file(world, &path);
    super::super::cube::verify(&mut world.app().owner, reopened, 2.);
    assert_eq!(
        &super::super::cube::topology_keys(&mut world.app().owner),
        keys
    );
    super::super::cube::resolve_public_face(&mut world.app().owner);
}

fn save_cube_file(world: &mut ApplicationWorld, path: &std::path::Path) {
    click_label(world, "Save");
    type_text(world, path.to_str().unwrap());
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    assert!(path.is_file());
}

fn reopen_cube_file(world: &mut ApplicationWorld, path: &std::path::Path) -> u64 {
    click_label(world, "File");
    click_label(world, "Close");
    assert!(
        world
            .app()
            .owner
            .documents()
            .workspace()
            .active_document()
            .is_none()
    );
    x11_input::press_control_o();
    world.pump(Vec::new());
    type_text(world, path.to_str().unwrap());
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    active_document(world)
}

fn create_size_parameter(world: &mut ApplicationWorld) {
    click_label(world, "Parameters");
    let (position, _) = diagnostics::window_geometry("Parameters").unwrap();
    for (offset, value) in [(235., "size"), (380., "10 mm"), (475., "")] {
        let point = diagnostics::pointer_pixels([position[0] + offset, position[1] + 140.]);
        x11_input::click(point);
        world.pump(Vec::new());
        type_text(world, value);
    }
    click_label(world, "Done");
}

fn type_text(world: &mut ApplicationWorld, text: &str) {
    for character in text.chars() {
        x11_input::press_keysym(u32::from(character));
        world.pump(Vec::new());
    }
}

fn edit_size_parameter(world: &mut ApplicationWorld) {
    click_label(world, "Parameters");
    let (position, size) = diagnostics::window_geometry("Parameters").unwrap();
    let name = super::rendered_label_in_rect(
        "size",
        position,
        [position[0] + 200., position[1] + size[1]],
    );
    let x = diagnostics::pointer_pixels([position[0] + 290., position[1]])[0];
    x11_input::click([x, name[1]]);
    world.pump(Vec::new());
    x11_input::press_keysym(0xff57);
    for _ in 0..5 {
        x11_input::press_keysym(0xff08);
        world.pump(Vec::new());
    }
    type_text(world, "20 mm");
    click_label(world, "Done");
}

fn show_cube(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry("##orientation-control").unwrap();
    let home = oblikovati_gui::GuiOrientationLayout::for_side(size[0]).home;
    x11_input::click(diagnostics::pointer_pixels([
        position[0] + home[0],
        position[1] + home[1],
    ]));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    assert!(super::super::scene_tests::solid_pixels(&capture) > 0);
    if let Ok(path) = std::env::var("OBLIKOVATI_OS_CUBE_CAPTURE") {
        use crate::WindowCaptureFiles;
        crate::NativeWindowCaptureFiles
            .write_capture(std::path::Path::new(&path), &capture)
            .unwrap();
    }
}
