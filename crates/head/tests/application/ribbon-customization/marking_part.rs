// SPDX-License-Identifier: GPL-2.0-only
//! Check the published Part marking-menu profile through the visible editor.
use super::*;
use oblikovati_api::{client::Client, wire::ParameterSetArgs};
use oblikovati_gui::GuiAuthoringContext;
use oblikovati_model::marking_menu::MarkingMenus;

#[given("`fixtures/cad/blank-metric-part.opd` is open with no active sketch in the MCAD Software")]
fn part_without_sketch(world: &mut ApplicationWorld) {
    super::super::super::ribbon_panel_expansion::open_blank_metric_part(world);
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    world.closed_document = world.previous_document;
}

#[then("the environment selector shows \"Part\"")]
fn part_environment(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "Part");
}

#[then(
    "the menu-location area shows Fillet above the center, Measure and Undo on the left, Extrude and Revolve on the right, Work Plane and Hole below, and New Sketch at the bottom"
)]
fn part_slots(world: &mut ApplicationWorld) {
    for (location, command) in [
        ("North", "Fillet"),
        ("NorthEast", "Extrude"),
        ("East", "Revolve"),
        ("SouthEast", "Hole"),
        ("South", "NewSketch"),
        ("SouthWest", "WorkPlane"),
        ("West", "Undo"),
        ("NorthWest", "Measure"),
    ] {
        let label = world.run_in_window("Customize", location);
        assert!(
            world.frame.runs.iter().any(|run| {
                run.list == label.list
                    && run.text == command
                    && run.maximum[0] < label.minimum[0]
                    && (run.center()[1] - label.center()[1]).abs() < 3.
            }),
            "expected {command} at {location}"
        );
    }
}

#[then("the customization dialog closes without applying changes")]
fn part_profile_unchanged(world: &mut ApplicationWorld) {
    customization_closed(world);
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert_eq!(world.state().marking_menus, Some(MarkingMenus::default()));
    assert!(
        !world
            .preferences_directory()
            .join("markingmenu.yaml")
            .exists()
    );
}

#[test]
fn measure_and_undo_run_from_part_marking_menu_slots() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    Client::new(world.app().owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "marking_width".into(),
            expression: "10 mm".into(),
            value_type: "numeric".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    assert!(has_marking_width(&mut world));

    world.right_click([950, 420]);
    click_part_menu(&mut world, "Measure");
    world.run_in_window("Measure", "Selectasolidfaceoredgetomeasure");
    world.click_text("Done");
    assert!(world.state().measure.is_none());

    world.right_click([950, 420]);
    click_part_menu(&mut world, "Undo");
    assert_eq!(world.state().document, Some(part));
    assert!(!has_marking_width(&mut world));
    world.close();
}

fn has_marking_width(world: &mut ApplicationWorld) -> bool {
    world
        .state()
        .parameters
        .iter()
        .any(|parameter| parameter.name == "marking_width")
}

fn click_part_menu(world: &mut ApplicationWorld, label: &str) {
    let center = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .unwrap_or_else(|| panic!("expected {label} in the open Part menu"))
        .center();
    world.click(diagnostics::pointer_pixels(center));
}
