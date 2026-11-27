// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::ribbon_appearance::icon_quads::icon_quads_inside;
use super::*;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

const LABELS: [&str; 4] = ["Join", "Cut", "Intersect", "NewSolid"];

#[given(
    "a fresh copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open in a solid extrusion with its circular profile selected"
)]
fn start_with_icons(world: &mut ApplicationWorld) {
    published_circle_part(world);
    start_circle_extrude(world);
    for label in LABELS {
        let [x, y] = world.run_in_window("Extrude", label).center();
        let icons = icon_quads_inside(([x - 14., y - 28.], [x + 14., y - 4.]));
        assert!(
            icons
                .iter()
                .any(|(low, high)| ((low[0] + high[0]) / 2. - x).abs() < 2.)
        );
    }
    click_panel(world, "Join");
    world.pump(Vec::new());
    observe_join_preview(world);
    world.save_capture("extrude-boolean-icons");
}

#[when("the user changes the Boolean selector style from Icons to Dropdown")]
fn switch_to_dropdown(world: &mut ApplicationWorld) {
    click_panel(world, "Operation");
    assert!(world.checked_run(popup_option(world, "Icons")));
    let point = diagnostics::pointer_pixels(popup_option(world, "Dropdown").center());
    super::super::viewport_interaction::click_at(world, point);
    world.pump(Vec::new());
    assert!(!world.window_text("Extrude").contains("Intersect"));
    observe_join_preview(world);
}

#[when("the user opens the Boolean dropdown")]
fn open_dropdown(world: &mut ApplicationWorld) {
    click_panel(world, "Join");
}

#[then("it offers Join, Cut, Intersect and New Solid")]
fn named_options(world: &mut ApplicationWorld) {
    for label in LABELS {
        assert!(world.run_is_enabled(label));
        popup_option(world, label);
    }
    world.save_capture("extrude-boolean-dropdown");
}

#[then("switching the selector style has not changed the currently selected operation")]
fn still_join(world: &mut ApplicationWorld) {
    observe_join_preview(world);
}

fn popup_option<'w>(world: &'w ApplicationWorld, label: &str) -> &'w diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .unwrap_or_else(|| panic!("expected {label:?} in the operation popup"))
}

fn observe_join_preview(world: &mut ApplicationWorld) {
    let state = world.state();
    let draft = state.extrude_draft_preview.expect("certified Join preview");
    assert!(state.extrude_preview_active);
    assert_eq!(draft.operation, Some(PrismBoolean::Join));
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Positive
    );
    assert_circular_cap(world);
}
