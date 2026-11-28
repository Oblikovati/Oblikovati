// SPDX-License-Identifier: GPL-2.0-only
//! Sketch expansion fixtures and checks for the ribbon benchmark (#3856).
use super::super::workspace_layout::browser_feature::{
    SKETCH, open_visible_extruded_part, right_click_row,
};
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};
use cucumber::{given, then, when};

const CREATE: &str = "##ribbon-additional-Create";
const CONSTRAIN: &str = "##ribbon-additional-Constrain";

pub(super) fn check_create_flyout_rows(world: &mut ApplicationWorld) -> f32 {
    world.hover([850, 350]);
    world.save_capture("sketch-create-flyout-rows");
    let (position, size) = diagnostics::window_geometry(CREATE).unwrap();
    let rows = ["CreateBlock", "PreciseInput"].map(|label| world.run_in_window(CREATE, label));
    let pitch = rows[1].center()[1] - rows[0].center()[1];
    assert!((pitch - 34.).abs() < 2., "Create flyout row pitch {pitch}");
    for row in rows {
        let y = row.center()[1];
        let icons = super::super::ribbon_appearance::icon_quads::icon_quads_inside((
            [position[0], y - 12.],
            [row.minimum[0], y + 12.],
        ));
        assert_eq!(icons.len(), 1, "expected a leading icon for {:?}", row.text);
        assert_eq!(icons[0].1[0] - icons[0].0[0], 16.);
    }
    position[0] + size[0] - 18.
}

fn widen_sketch_ribbon(world: &mut ApplicationWorld) {
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
}
#[given(
    "a part containing an existing sketch is active and the sketch is being edited in the MCAD Software"
)]
fn existing_sketch_under_edit(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
    widen_sketch_ribbon(world);
    world.record_initial_sketch();
}
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open with an existing sketch in the MCAD Software"
)]
#[given("`fixtures/cad/workspace-layout-native.opd` contains an existing editable `Sketch1`")]
fn named_sketch_part(world: &mut ApplicationWorld) {
    super::super::workspace_layout::native_part::open_named_extrusion(world);
    widen_sketch_ribbon(world);
}

#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open with `Sketch1` being edited in the MCAD Software"
)]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open with an active editable `Sketch1` in the MCAD Software"
)]
fn named_sketch_edit(world: &mut ApplicationWorld) {
    named_sketch_part(world);
    super::super::workspace_layout::native_part::edit_named_sketch(world);
    world.record_initial_sketch();
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with its visible solid in the MCAD Software"
)]
#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with a solid face visible in the MCAD Software"
)]
fn named_visible_solid(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    super::super::viewport_orientation::open_saved_navigation_block(world, &path);
    world.previous_document = Some(world.active_id());
}
#[given("the Sketch ribbon is expanded and the Create panel is visible")]
#[given("the Sketch ribbon is expanded and the Constrain panel is visible")]
fn full_sketch(world: &mut ApplicationWorld) {
    world.expect_text("Create");
    world.expect_text("Constrain");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the Create panel's additional command area is closed")]
fn create_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_none());
}
#[given("the Constrain panel's additional command area is closed")]
fn constrain_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}
#[when("the user selects the expansion arrow on the Create panel title")]
fn open_create(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Create");
}
#[when("the user opens the Constrain panel expansion")]
fn open_constrain(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Constrain");
}
#[then("an additional command area opens below that panel over the workspace")]
fn create_below_ribbon(_: &mut ApplicationWorld) {
    let (panel, _) = diagnostics::window_geometry(CREATE).expect("Create expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
}
#[then("an additional command area opens below the panel over the workspace")]
fn constrain_below_ribbon(_: &mut ApplicationWorld) {
    let (panel, _) = diagnostics::window_geometry(CONSTRAIN).expect("Constrain expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
}
#[then("the Create Block command is visible")]
fn create_block_visible(world: &mut ApplicationWorld) {
    world.run_in_window(CREATE, "CreateBlock");
}
#[then("Precise Input is visible but unavailable in this captured state")]
fn precise_unavailable(world: &mut ApplicationWorld) {
    assert!(world.run_in_window(CREATE, "PreciseInput").alpha < u8::MAX);
}
#[then("the sketch and model browser remain visible")]
fn sketch_and_browser(world: &mut ApplicationWorld) {
    assert!(world.state().active_sketch.is_some());
    assert!(diagnostics::window_geometry("Model").is_some());
}
#[then("the Create additional command area remains open")]
fn create_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_some());
}
#[then("the Create additional command area closes")]
fn create_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_none());
}
#[then("the Constrain additional command area closes")]
fn constrain_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}
