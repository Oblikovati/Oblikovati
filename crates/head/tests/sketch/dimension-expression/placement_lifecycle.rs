// SPDX-License-Identifier: GPL-2.0-only
//! Native General Dimension placement, editing, cancellation and history.
use super::super::super::{ApplicationWorld, diagnostics};
use oblikovati_viewport::ViewportPhysicalKey;

fn draw_line(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_line_tool();
    world.sketch_click([650, 450]);
    world.sketch_click([950, 500]);
    world.escape();
}
fn place_line_dimension(world: &mut ApplicationWorld) {
    draw_line(world);
    place_dimension(world);
}
fn place_dimension(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    world.sketch_click([800, 475]);
    world.sketch_click([800, 560]);
    world.pump(Vec::new());
}
fn dimension_count(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().dimensional_constraints().count()
}
#[test]
fn cancel_value_retains_placed_dimension_and_its_history() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    assert_eq!(dimension_count(&mut world), 1);
    assert!(diagnostics::window_geometry("Edit Dimension").is_some());
    world.type_text("20");
    world.escape();
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert_eq!(dimension_count(&mut world), 1);
    assert!(world.state().dimension_tool_active);
    world.escape();
    world.undo();
    assert_eq!(dimension_count(&mut world), 0);
    world.redo();
    assert_eq!(dimension_count(&mut world), 1);
    world.close();
}
fn dimension_value(world: &mut ApplicationWorld) -> f64 {
    use oblikovati_api::{client::Client, wire::SketchArgs};
    Client::new(world.app().owner.document_host())
        .sketch()
        .dimensions(&SketchArgs::default())
        .unwrap()
        .dimensions
        .unwrap()[0]
        .value
}
#[test]
fn accepted_value_and_placement_undo_as_separate_steps() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    let measured = dimension_value(&mut world);
    world.type_text("20");
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert!(world.state().dimension_tool_active);
    let edited = dimension_value(&mut world);
    assert_ne!(edited, measured);
    world.escape();
    world.undo();
    assert_eq!(dimension_count(&mut world), 1);
    assert!((dimension_value(&mut world) - measured).abs() < 1e-8);
    world.undo();
    assert_eq!(dimension_count(&mut world), 0);
    world.redo();
    assert_eq!(dimension_count(&mut world), 1);
    assert!((dimension_value(&mut world) - measured).abs() < 1e-8);
    world.redo();
    assert!((dimension_value(&mut world) - edited).abs() < 1e-8);
    world.close();
}

#[test]
fn invalid_value_keeps_the_placed_dimension_for_correction() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    let measured = dimension_value(&mut world);
    world.type_text("1+");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_text("The expression cannot be evaluated.");
    assert_eq!(dimension_count(&mut world), 1);
    assert_eq!(dimension_value(&mut world), measured);
    world.click_text("OK");
    world.key(ViewportPhysicalKey::End, false);
    world.type_text("2");
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert_eq!(dimension_count(&mut world), 1);
    assert_ne!(dimension_value(&mut world), measured);
    world.close();
}
#[test]
fn refused_driving_dimension_can_be_placed_as_reference() {
    use oblikovati_api::{client::Client, wire::AddConstraintArgs};
    let mut world = ApplicationWorld::new();
    draw_line(&mut world);
    let geometry = world.edited_geometry();
    let endpoints = geometry.lines()[0].endpoints;
    for point in endpoints {
        Client::new(world.app().owner.document_host())
            .constrain()
            .add(&AddConstraintArgs {
                kind: "fix".into(),
                entities: Some(vec![point.0]),
                ..Default::default()
            })
            .unwrap();
    }
    world.pump(Vec::new());
    place_dimension(&mut world);
    let draft = world.state().dimension_draft.unwrap();
    assert!(draft.committed.is_none());
    assert!(draft.refusal.is_some());
    assert!(
        world
            .window_text("Edit Dimension")
            .contains("dimensioncreation")
    );
    assert_eq!(dimension_count(&mut world), 0);
    world.click_text("Reference (driven)");
    assert!(world.checked("Reference (driven)"));
    world.click_text(&draft.expression);
    world.key(ViewportPhysicalKey::Enter, false);
    let rows = Client::new(world.app().owner.document_host())
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap();
    assert_eq!(
        rows.len(),
        1,
        "draft {:?}; status {:?}; editor {:?}",
        world.state().dimension_draft,
        world.status_text(),
        world.window_text("Edit Dimension")
    );
    assert!(rows[0].driven);
    assert!(world.state().dimension_tool_active);
    world.close();
}
#[test]
fn cancelled_value_and_placed_dimension_survive_save_and_reopen() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    let measured = dimension_value(&mut world);
    world.type_text("20");
    world.escape();
    world.escape();
    world.finish_sketch();
    world.press_save_shortcut();
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected the saved sketch in the browser")
        .center();
    world.double_click(row.map(|coordinate| coordinate as i32));
    world.pump(Vec::new());
    assert_eq!(dimension_count(&mut world), 1);
    assert!((dimension_value(&mut world) - measured).abs() < 1e-8);
    world.close();
}
#[test]
fn editing_an_existing_label_keeps_general_dimension_active() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    world.escape();
    let label = crate::HeadWork::sketch_dimension_annotations(&world.app().owner)
        .unwrap()
        .labels[0]
        .text
        .clone();
    world.click(world.exact_run_center(&label));
    assert!(diagnostics::window_geometry("Edit Dimension").is_some());
    world.type_text("18");
    world.key(ViewportPhysicalKey::Enter, false);
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert!(world.state().dimension_tool_active);
    world.close();
}
#[test]
fn cancel_button_keeps_the_placed_dimension_and_command_active() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    let measured = dimension_value(&mut world);
    world.type_text("20");
    world.click_text("Cancel");
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert_eq!(dimension_count(&mut world), 1);
    assert_eq!(dimension_value(&mut world), measured);
    assert!(world.state().dimension_tool_active);
    world.close();
}
#[test]
fn ok_button_accepts_a_value_without_leaving_general_dimension() {
    let mut world = ApplicationWorld::new();
    place_line_dimension(&mut world);
    let measured = dimension_value(&mut world);
    world.type_text("20");
    world.click_text("OK");
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert_ne!(dimension_value(&mut world), measured);
    assert!(world.state().dimension_tool_active);
    world.close();
}
