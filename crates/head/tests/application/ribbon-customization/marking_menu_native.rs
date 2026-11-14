// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for marking menu customization.
use super::*;

#[test]
fn api_marking_menu_overflow_runs_its_sketch_command() {
    use oblikovati_api::{
        client::Client,
        types,
        wire::{MarkingMenuView, SetMarkingMenuArgs},
    };
    let mut world = ApplicationWorld::new();
    world.add_part();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: None,
        overflow: Some(vec!["Sketch.Rectangle".into()]),
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in menu overflow: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn marking_menu_editor_applies_and_restores_a_slot() {
    use oblikovati_api::{client::Client, types, wire::GetMarkingMenuArgs};
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "MarkingMenu");
    let north = world.run_in_window("Customize", "North").clone();
    world.click([(north.minimum[0] - 70.) as i32, north.center()[1] as i32]);
    world.pump(Vec::new());
    world.click_text("Rectangle");
    world.pump(Vec::new());
    click_in_customize(&mut world, "Apply");
    let menu = Client::new(world.app().owner.document_host())
        .ui()
        .marking_menu(&GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        })
        .unwrap();
    assert!(menu.quadrants.unwrap().iter().any(
        |item| item.quadrant == types::QUADRANT_NORTH && item.command_id == "Sketch.Rectangle"
    ));
    click_in_customize(&mut world, "Close");
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in canvas menu: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}

#[test]
fn saved_classic_menu_keeps_the_sketch_context_menu_linear() {
    let mut world = ApplicationWorld::new();
    std::fs::write(
        world.preferences_directory().join("markingmenu.yaml"),
        "classic: true\n",
    )
    .unwrap();
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    assert!(world.state().sketch_classic_menu);
    world.right_click([900, 450]);
    world.expect_text("Measure");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
    );
    world.close();
}

#[test]
fn api_marking_menu_drives_the_sketch_canvas() {
    use oblikovati_api::{
        client::Client,
        types,
        wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
    };
    let mut world = ApplicationWorld::new();
    world.add_part();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Rectangle".into(),
        }]),
        overflow: None,
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.reopen_application();
    world.add_part();
    world.start_sketch();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Rectangle" && run.center()[0] > 700. && run.center()[1] > 250.)
        .unwrap_or_else(|| panic!("expected Rectangle in canvas menu: {:?}", world.text()))
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.click([760, 380]);
    world.click([1060, 560]);
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.close();
}
