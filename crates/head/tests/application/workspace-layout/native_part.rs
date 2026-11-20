// SPDX-License-Identifier: GPL-2.0-only
//! Load the named Part and point fixtures through the desktop Open dialog (#3897).
use super::{ApplicationWorld, browser_feature};
use cucumber::{given, then};
use oblikovati_gui::GuiModelBrowser;

const WORKSPACE_PART: &str = "fixtures/cad/workspace-layout-native.opd";
const POINTS_PART: &str = "fixtures/cad/workspace-layout-points.opd";
pub(super) const INSPECTION_POINTS: [&str; 3] =
    ["InspectionPointA", "InspectionPointB", "InspectionPointC"];

fn open_workspace_part(world: &mut ApplicationWorld, relative: &str) -> GuiModelBrowser {
    let path = super::super::scenario_root().join(relative);
    world.open_fixture(&path);
    let document = world.active_id();
    let state = world.state();
    let storage = &state.document_storage[&document];
    assert_eq!(
        storage.path.as_deref().map(std::path::Path::new),
        Some(path.as_path())
    );
    world.previous_document = Some(document);
    let browser = state.browser.expect("saved Part: expected the browser");
    assert_eq!(browser.kind, oblikovati_gui::GuiBrowserKind::Part);
    assert!(browser.bodies.iter().any(|body| body.visible));
    browser
}

#[given("`fixtures/cad/workspace-layout-native.opd` is open in the MCAD Software")]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is open in the MCAD Software with nested model browser branches"
)]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is active in the MCAD Software with a 20 mm new-solid extrusion"
)]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` is active in the MCAD Software with an extrusion and its underlying dimensioned sketch"
)]
pub(super) fn open_named_extrusion(world: &mut ApplicationWorld) {
    let browser = open_workspace_part(world, WORKSPACE_PART);
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Extrusion1")
    );
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1")
    );
}

#[given("`fixtures/cad/workspace-layout-points.opd` is open in the MCAD Software")]
#[given(
    "`fixtures/cad/workspace-layout-points.opd` is open in the MCAD Software with multiple work points and other features"
)]
fn open_named_points(world: &mut ApplicationWorld) {
    let browser = open_workspace_part(world, POINTS_PART);
    for name in INSPECTION_POINTS {
        assert!(
            browser.features.iter().any(|feature| feature.name == name),
            "{name}: expected a saved point row"
        );
    }
}

#[given("Sketch1 is being edited")]
fn edit_named_sketch(world: &mut ApplicationWorld) {
    browser_feature::right_click_row(world, "Sketch1");
    browser_feature::choose_menu_row(world, "Edit Sketch");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert_eq!(world.state().document, world.previous_document);
    world.expect_text("Finish Sketch");
}

#[then("InspectionPointA, InspectionPointB and InspectionPointC are shown in the browser")]
fn filtered_point_rows(world: &mut ApplicationWorld) {
    for name in INSPECTION_POINTS {
        world.expect_text(name);
    }
}
