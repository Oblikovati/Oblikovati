// SPDX-License-Identifier: GPL-2.0-only
//! Route cancellation to the visible dialog.
use super::*;

#[when("the user clicks \"Cancel\"")]
#[when("the user clicks Cancel")]
pub(super) fn cancel_customization(world: &mut ApplicationWorld) {
    if world.state().sketch_text.is_some() {
        let at = world.run_in_window("Format Text", "Cancel").center();
        super::super::x11_input::click(diagnostics::pointer_pixels(at));
        world.pump(Vec::new());
        return;
    }
    for title in [
        "Import Customization Settings",
        "Export Customization Settings",
        "Geometry Properties",
        "Feature Properties",
        "Incremental View Rotate",
        "Navigation Wheel Options",
        "Save Search",
        "Open Search",
    ] {
        if let Some((position, size)) = diagnostics::window_geometry(title) {
            let center = world
                .frame
                .runs
                .iter()
                .rev()
                .find(|run| {
                    let point = run.center();
                    run.text == "Cancel"
                        && (0..2).all(|axis| {
                            point[axis] >= position[axis]
                                && point[axis] <= position[axis] + size[axis]
                        })
                })
                .expect("expected the dialog's Cancel button")
                .center();
            world.click(diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
            return;
        }
    }
    click_in_customize(world, "Cancel");
}

#[test]
fn cancel_keeps_customization_after_closing_import_picker() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Import");
    assert!(diagnostics::window_geometry("Import Customization Settings").is_some());
    cancel_customization(&mut world);
    assert!(diagnostics::window_geometry("Import Customization Settings").is_none());
    assert!(world.window_text("Customize").contains("Part|Tools*"));
    cancel_customization(&mut world);
    assert!(diagnostics::window_geometry("Customize").is_none());
    world.expect_no_text("User Commands");
    world.close();
}
