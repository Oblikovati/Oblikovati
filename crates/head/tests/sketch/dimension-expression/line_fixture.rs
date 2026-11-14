// SPDX-License-Identifier: GPL-2.0-only
//! A dimensioned oblique line for the recorded dependency scenarios.
use super::*;
use oblikovati_api::wire::AddDimensionArgs;

pub(super) fn create(world: &mut ApplicationWorld) {
    create_with_reference_at(world, [1., 5.]);
}
pub(super) fn create_overlapping(world: &mut ApplicationWorld) {
    create_with_reference_at(world, [5., 4.5]);
}
fn create_with_reference_at(world: &mut ApplicationWorld, reference_at: [f64; 2]) {
    if world.state().authoring != oblikovati_gui::GuiAuthoringContext::Sketch {
        world.start_sketch_on("YZ Plane");
    }
    world.select_line_tool();
    for point in [[2., 2.], [4., 7.]] {
        let pixel = world.sketch_pixel_of(point);
        world.sketch_click(pixel);
    }
    world.escape();
    let line = world.edited_geometry().lines()[0];
    let entities = line.endpoints.map(|id| id.0).to_vec();
    add(
        world,
        &entities,
        "horizontal",
        "20 mm",
        [3., 1.],
        Some("d63"),
        false,
    );
    add(
        world,
        &entities,
        "vertical",
        "50 mm",
        [5., 4.5],
        Some("d64"),
        false,
    );
    add(
        world,
        &entities,
        "aligned",
        "1 mm",
        reference_at,
        Some("d78"),
        true,
    );
    world.pump(Vec::new());
}
fn add(
    world: &mut ApplicationWorld,
    entities: &[u64],
    orientation: &str,
    expression: &str,
    text_point: [f64; 2],
    name: Option<&str>,
    driven: bool,
) {
    let client = Client::new(world.app().owner.document_host());
    let added = client
        .dimension()
        .add(&AddDimensionArgs {
            kind: "distance".into(),
            entities: Some(entities.to_vec()),
            orientation: orientation.into(),
            expression: expression.into(),
            text_point: Some(text_point.to_vec()),
            driven,
            ..Default::default()
        })
        .unwrap();
    if let Some(name) = name {
        client
            .parameters()
            .rename(&ParameterRenameArgs {
                name: added.parameter,
                new_name: name.into(),
            })
            .unwrap();
    }
}

#[test]
fn native_oblique_fixture_displays_components_and_reference_length() {
    let mut world = ApplicationWorld::new();
    create(&mut world);
    world.expect_annotation("20");
    world.expect_annotation("50");
    world.expect_annotation("(53.852)");
    world.close();
}

#[test]
fn native_click_on_dimension_line_opens_its_editor() {
    let mut world = ApplicationWorld::new();
    create(&mut world);
    world.key(ViewportPhysicalKey::D, false);
    let point = world.sketch_pixel_of([5., 2.5]);
    world.sketch_click(point);
    world.expect_text("d64");
    assert!(diagnostics::window_geometry("Edit Dimension").is_some());
    world.close();
}
