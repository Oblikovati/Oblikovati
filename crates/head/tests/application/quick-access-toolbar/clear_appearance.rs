// SPDX-License-Identifier: GPL-2.0-only
//! Clear an exact selected body override through the native quick access command.
use super::super::workspace_layout::browser_feature::{
    click_branch_arrow, open_visible_extruded_part,
};
use super::*;
use crate::HeadWork;
use oblikovati_api::{
    client::Client,
    wire::{AssignAppearanceArgs, AssignMaterialArgs},
};

fn body_with_override(world: &mut ApplicationWorld) -> String {
    open_visible_extruded_part(world);
    assign_part_assets(world);
    let body = world.state().browser.unwrap().bodies[0].key.clone();
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "body".into(),
            key: body.clone(),
            appearance_id: "abs-black".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    click_branch_arrow(world, "Solid Bodies");
    world.click_text("Solid1");
    world.pump(Vec::new());
    body
}
fn selected_appearance(world: &mut ApplicationWorld, key: &str) -> String {
    world
        .state()
        .browser
        .unwrap()
        .bodies
        .into_iter()
        .find(|body| body.key == key)
        .unwrap()
        .appearance
}
pub(super) fn enable_clear(world: &mut ApplicationWorld) {
    open_customization(world);
    world.click_text("Clear");
    world.pump(Vec::new());
    assert!(on_row(world, "Clear"));
}
#[test]
fn clear_body_override_requires_ok_and_supports_undo_redo() {
    let mut world = ApplicationWorld::new();
    let key = body_with_override(&mut world);
    enable_clear(&mut world);
    assert_eq!(selected_appearance(&mut world, &key), "abs-black");
    let unchanged = part_assets(&mut world);
    verify_cancel(&mut world, &key);
    world.click_text("Extrude1");
    world.click(row_center(&world, "Clear"));
    assert!(window_geometry("Clear Appearance").is_some());
    assert!(!world.run_is_enabled("OK"));
    world.click_text("Solid1");
    world.pump(Vec::new());
    assert!(world.run_is_enabled("OK"));
    world.click_text("OK");
    assert_eq!(selected_appearance(&mut world, &key), "");
    assert_eq!(part_assets(&mut world), unchanged);
    assert_eq!(
        world.state().selected_body_key.as_deref(),
        Some(key.as_str())
    );
    verify_clear_history(&mut world, &key);
    world.close();
}

fn verify_clear_history(world: &mut ApplicationWorld, key: &str) {
    world.click(row_center(world, "Undo"));
    assert_eq!(selected_appearance(world, key), "abs-black");
    world.click(row_center(world, "Redo"));
    assert_eq!(selected_appearance(world, key), "");
}

fn part_assets(world: &mut ApplicationWorld) -> (Option<String>, Option<String>) {
    let browser = world.state().browser.unwrap();
    (browser.part_material, browser.part_appearance)
}

#[test]
fn clear_cannot_apply_after_the_selection_changes() {
    let mut world = ApplicationWorld::new();
    let key = body_with_override(&mut world);
    enable_clear(&mut world);
    world.click(row_center(&world, "Clear"));
    world.click_text("Extrude1");
    world.pump(Vec::new());
    assert!(!world.run_is_enabled("OK"));
    let document = world.active_id();
    assert!(
        world
            .app()
            .owner
            .clear_selected_body_appearances_from_gui(document, std::slice::from_ref(&key))
            .is_err()
    );
    world.click_text("Cancel");
    assert_eq!(selected_appearance(&mut world, &key), "abs-black");
    world.close();
}

fn verify_cancel(world: &mut ApplicationWorld, key: &str) {
    let history = world.state().undo_labels;
    world.click(row_center(world, "Clear"));
    assert!(window_geometry("Clear Appearance").is_some());
    world.click_text("Cancel");
    assert_eq!(world.state().undo_labels, history);
    assert_eq!(selected_appearance(world, key), "abs-black");
}

fn assign_part_assets(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let material = browser.materials.first().unwrap().id.clone();
    let appearance = browser
        .appearances
        .iter()
        .find(|item| item.id != "abs-black")
        .unwrap()
        .id
        .clone();
    assign_part_material(world, &material);
    assign_part_appearance(world, &appearance);
    world.pump(Vec::new());
}
fn assign_part_material(world: &mut ApplicationWorld, id: &str) {
    Client::new(world.app().owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: String::new(),
            material_id: id.into(),
        })
        .unwrap();
}
fn assign_part_appearance(world: &mut ApplicationWorld, id: &str) {
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "part".into(),
            key: String::new(),
            appearance_id: id.into(),
        })
        .unwrap();
}

#[test]
fn ctrl_select_two_bodies_and_clear_in_one_undo() {
    let mut world = ApplicationWorld::new();
    let keys = three_bodies_with_overrides(&mut world);
    let part_intent = part_assets(&mut world);
    world.click_text("Solid1");
    ctrl_click(&mut world, "Solid2");
    assert_eq!(world.state().selected_body_keys, keys[..2]);
    assert!(world.row_accented_beside("Solid1", "Solid3"));
    assert!(world.row_accented_beside("Solid2", "Solid3"));
    enable_clear(&mut world);
    verify_cancel(&mut world, &keys[0]);
    assert_body_appearances(&mut world, &keys, ["abs-black"; 3]);
    world.click(row_center(&world, "Clear"));
    assert!(world.run_is_enabled("OK"));
    world.save_capture("clear-two-selected-bodies");
    world.click_text("OK");
    assert_body_appearances(&mut world, &keys, ["", "", "abs-black"]);
    verify_multi_body_history(&mut world, &keys);
    verify_multi_body_reopen(&mut world, &keys);
    assert_eq!(part_assets(&mut world), part_intent);
    world.close();
}
fn verify_multi_body_history(world: &mut ApplicationWorld, keys: &[String]) {
    world.click(row_center(world, "Undo"));
    assert_body_appearances(world, keys, ["abs-black"; 3]);
    world.click(row_center(world, "Redo"));
    assert_body_appearances(world, keys, ["", "", "abs-black"]);
}
fn assert_body_appearances(world: &mut ApplicationWorld, keys: &[String], expected: [&str; 3]) {
    for (key, appearance) in keys.iter().zip(expected) {
        assert_eq!(selected_appearance(world, key), appearance);
    }
}
fn verify_multi_body_reopen(world: &mut ApplicationWorld, keys: &[String]) {
    let document = world.active_id();
    let path = world.save_part_to_file(document, "clear-multi-reopen.opd");
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .close(&oblikovati_api::wire::CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_body_appearances(world, keys, ["", "", "abs-black"]);
}
fn three_bodies_with_overrides(world: &mut ApplicationWorld) -> Vec<String> {
    let path = world.scratch_folder().join("three-bodies.opd");
    std::fs::write(&path, three_body_package()).unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
    let bodies = world.state().browser.unwrap().bodies;
    assert_eq!(bodies.len(), 3, "expected three solids in the test fixture");
    let keys: Vec<_> = bodies.into_iter().map(|body| body.key).collect();
    assign_part_assets(world);
    for key in &keys {
        assign_body_override(world, key);
    }
    world.pump(Vec::new());
    click_branch_arrow(world, "Solid Bodies");
    keys
}
fn assign_body_override(world: &mut ApplicationWorld, key: &str) {
    Client::new(world.app().owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "body".into(),
            key: key.into(),
            appearance_id: "abs-black".into(),
        })
        .unwrap();
}
fn three_body_package() -> Vec<u8> {
    use oblikovati_model::{
        feature::{AuthoredFeature, FeatureId},
        units::UnitsOfMeasure,
    };
    use oblikovati_persistence::package::DocumentPackage;
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    for (id, name) in [(2, "Second"), (3, "Third")] {
        let feature = AuthoredFeature::sketch_prism(
            FeatureId(id),
            name,
            crate::scene_fixture::selected_prism_definition(),
            &snapshot,
        )
        .unwrap();
        snapshot.work_geometry.append(feature).unwrap();
    }
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 3)
        .unwrap()
        .encode()
        .unwrap()
}
pub(super) fn ctrl_click(world: &mut ApplicationWorld, text: &str) {
    let position = world.run(text).center().map(|value| value as i32);
    modified_viewport_click(world, position, true, false);
}

#[test]
fn viewport_two_bodies_clear_with_modifiers() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    let keys = open_separate_bodies(&mut world);
    fit_separate_bodies(&mut world);
    world.click(row_center(&world, "Select"));
    world.click_text("Components");
    assert_eq!(
        world.state().selection_mode,
        oblikovati_gui::GuiSelectionMode::Components
    );
    enable_clear(&mut world);
    world.click(row_center(&world, "Clear"));
    assert!(!world.run_is_enabled("OK"));
    select_separate_bodies(&mut world, &keys);
    verify_viewport_clear(&mut world, &keys);
    world.close();
}

fn fit_separate_bodies(world: &mut ApplicationWorld) {
    use crate::HeadWindow;
    world
        .app()
        .window
        .navigate_view(oblikovati_viewport::ViewportNavigation::Fit { margin: 0.1 })
        .unwrap();
    world.pump(Vec::new());
}

fn select_separate_bodies(world: &mut ApplicationWorld, keys: &[String]) {
    use crate::HeadWindow;
    let first = world.app().window.model_pixel([2., 1.5, 2.]).unwrap();
    let second = world.app().window.model_pixel([8., 1.5, 2.]).unwrap();
    world.click(first);
    assert_eq!(world.state().selected_body_keys, [keys[0].clone()]);
    modified_viewport_click(world, second, true, false);
    assert_eq!(world.state().selected_body_keys, keys);
    modified_viewport_click(world, second, false, true);
    assert_eq!(world.state().selected_body_keys, [keys[0].clone()]);
    modified_viewport_click(world, second, false, true);
    assert_eq!(world.state().selected_body_keys, keys);
    assert!(world.run_is_enabled("OK"));
}

fn verify_viewport_clear(world: &mut ApplicationWorld, keys: &[String]) {
    world.click_text("Cancel");
    assert_pair_appearances(world, keys, "abs-black");
    world.click(row_center(world, "Clear"));
    world.click_text("OK");
    assert_pair_appearances(world, keys, "");
    world.click(row_center(world, "Undo"));
    assert_pair_appearances(world, keys, "abs-black");
}

fn assert_pair_appearances(world: &mut ApplicationWorld, keys: &[String], expected: &str) {
    assert_eq!(selected_appearance(world, &keys[0]), expected);
    assert_eq!(selected_appearance(world, &keys[1]), expected);
}

fn open_separate_bodies(world: &mut ApplicationWorld) -> Vec<String> {
    let path = world.scratch_folder().join("separate-bodies.opd");
    std::fs::write(&path, separate_body_package()).unwrap();
    open_body_fixture(world, &path);
    let keys: Vec<_> = world
        .state()
        .browser
        .unwrap()
        .bodies
        .into_iter()
        .map(|body| body.key)
        .collect();
    assert_eq!(keys.len(), 2);
    for key in &keys {
        assign_body_override(world, key);
    }
    world.pump(Vec::new());
    keys
}

fn open_body_fixture(world: &mut ApplicationWorld, path: &std::path::Path) {
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

fn separate_body_package() -> Vec<u8> {
    use oblikovati_model::units::UnitsOfMeasure;
    use oblikovati_persistence::package::DocumentPackage;
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    add_separate_sketch(&mut snapshot);
    add_separate_extrusion(&mut snapshot);
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, UnitsOfMeasure::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 2, 204)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 2)
        .unwrap()
        .encode()
        .unwrap()
}

fn add_separate_sketch(snapshot: &mut oblikovati_model::history::DocumentSnapshot) {
    use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};
    let plane =
        SketchPlane::new([6., 0., 0.], [1., 0., 0.], [0., 1., 0.], Default::default()).unwrap();
    snapshot
        .sketches
        .add(
            DocumentSketchId(2),
            "Second profile",
            plane,
            separate_rectangle(),
        )
        .unwrap();
    snapshot
        .sketches
        .set_visible(DocumentSketchId(2), false)
        .unwrap();
    snapshot
        .sketches
        .set_visible(crate::scene_fixture::SKETCH, false)
        .unwrap();
    crate::scene_fixture::solve_selected_sketches(snapshot);
}

fn add_separate_extrusion(snapshot: &mut oblikovati_model::history::DocumentSnapshot) {
    use oblikovati_model::{
        document_sketch::DocumentSketchId,
        feature::{AuthoredFeature, FeatureId, sketch_prism::SketchPrismDefinition},
    };
    let mut recipe = crate::scene_fixture::selected_prism_definition().recipe();
    recipe.sketch = DocumentSketchId(2);
    recipe.selection.outer = crate::scene_fixture::selected_sketch_line_ring(201);
    let definition = SketchPrismDefinition::from_recipe(&recipe).unwrap();
    let feature =
        AuthoredFeature::sketch_prism(FeatureId(2), "Second", definition, snapshot).unwrap();
    snapshot.work_geometry.append(feature).unwrap();
}

fn separate_rectangle() -> oblikovati_model::sketch::Sketch2D {
    use oblikovati_model::sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
    };
    let mut geometry = Sketch2D::default();
    let points = [[0., 0.], [4., 0.], [4., 3.], [0., 3.]];
    for (index, position) in points.into_iter().enumerate() {
        let point = SketchEntityId(11 + index as u64);
        geometry.add_point(point, position).unwrap();
        geometry
            .add_constraint(
                SketchConstraintId(21 + index as u64),
                PlanarConstraint::Fix { point, position },
            )
            .unwrap();
    }
    for index in 0..4 {
        let ends = [11 + index, 11 + (index + 1) % 4].map(SketchEntityId);
        geometry
            .add_line(SketchEntityId(201 + index), ends)
            .unwrap();
    }
    geometry
}

pub(super) fn modified_viewport_click(
    world: &mut ApplicationWorld,
    position: [i32; 2],
    control: bool,
    shift: bool,
) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportModifiers};
    let modifiers = ViewportModifiers {
        control,
        shift,
        ..Default::default()
    };
    let mut events = vec![ViewportEvent::Input(ViewportInput::Motion {
        position,
        modifiers,
    })];
    events.extend([true, false].map(|pressed| viewport_button(position, modifiers, pressed)));
    world.pump(events);
    world.pump(Vec::new());
}

fn viewport_button(
    position: [i32; 2],
    modifiers: oblikovati_viewport::ViewportModifiers,
    pressed: bool,
) -> oblikovati_viewport::ViewportEvent {
    use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};
    ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed,
        position,
        modifiers,
    })
}
