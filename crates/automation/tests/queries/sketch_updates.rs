// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{parameter_value::numeric, parameters::parameter_workspace};
use oblikovati_automation::SessionHost;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::RecipeReservations,
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchPlane},
};
pub(super) fn create_circle(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<DocumentSketchId, String> {
    let sketch = ids.allocate_sketch().map_err(|cause| cause.to_string())?;
    let center = ids
        .allocate_sketch_entity()
        .map_err(|cause| cause.to_string())?;
    let circle = ids
        .allocate_sketch_entity()
        .map_err(|cause| cause.to_string())?;
    let constraints = std::array::from_fn(|_| ids.allocate_sketch_constraint(sketch).unwrap());
    let geometry = dimensioned_circle(center, circle, constraints);
    snapshot
        .sketches
        .add(sketch, "Driven circle", SketchPlane::xy(), geometry)
        .map_err(|cause| cause.to_string())?;
    Ok(sketch)
}
fn dimensioned_circle(
    center: SketchEntityId,
    circle: SketchEntityId,
    constraints: [SketchConstraintId; 2],
) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(center, [0.0, 0.0]).unwrap();
    geometry.add_circle(circle, center, 1.0).unwrap();
    for (constraint, relation) in constraints
        .into_iter()
        .zip(circle_relations(center, circle))
    {
        geometry.add_constraint(constraint, relation).unwrap();
    }
    geometry
}
fn circle_relations(center: SketchEntityId, circle: SketchEntityId) -> [PlanarConstraint; 2] {
    [
        PlanarConstraint::Fix {
            point: center,
            position: [0.0, 0.0],
        },
        PlanarConstraint::Radius {
            curve: circle,
            parameter: 1,
        },
    ]
}
pub(super) fn circle_radius(workspace: &Workspace, id: u64, sketch: DocumentSketchId) -> f64 {
    let snapshot = workspace.session(id).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert!(
        placed
            .require_fully_constrained(&snapshot.parameters)
            .is_ok()
    );
    placed.geometry().circles()[0].radius
}
fn api_width(
    workspace: &mut Workspace,
    expression: &str,
) -> Result<ParameterInfo, oblikovati_api::Error> {
    let client = Client::new(SessionHost::new(workspace));
    client.parameters().set(&numeric("width", expression))
}
pub(crate) fn circle_workspace() -> (Workspace, u64, DocumentSketchId) {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    let sketch = workspace
        .commands(id)
        .unwrap()
        .run("Sketch", create_circle)
        .unwrap();
    (workspace, id, sketch)
}
pub(crate) fn veto_workspace(snapshot: DocumentSnapshot) -> (Workspace, u64) {
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    super::parameters::parameter_workspace_sketches(
        snapshot.parameters,
        snapshot.sketches,
        observer,
    )
}
#[test]
fn typed_parameter_api_updates_geometry_in_the_same_undo_step() {
    let (mut workspace, id, sketch) = circle_workspace();
    api_width(&mut workspace, "7 cm").unwrap();
    assert!((circle_radius(&workspace, id, sketch) - 7.0).abs() < 1e-8);
    workspace.commands(id).unwrap().undo().unwrap();
    assert!((circle_radius(&workspace, id, sketch) - 1.0).abs() < 1e-8);
    workspace.commands(id).unwrap().redo().unwrap();
    assert!((circle_radius(&workspace, id, sketch) - 7.0).abs() < 1e-8);
}
#[test]
fn observer_veto_discards_both_the_parameter_edit_and_automatic_geometry_candidate() {
    let (original, id, sketch) = circle_workspace();
    let snapshot = original.session(id).unwrap().snapshot().clone();
    let (mut vetoed, id) = veto_workspace(snapshot);
    let error = api_width(&mut vetoed, "8 cm").unwrap_err();
    assert!(error.to_string().contains("settings are locked"));
    assert!((circle_radius(&vetoed, id, sketch) - 1.0).abs() < 1e-8);
    assert!(vetoed.session(id).unwrap().undo_labels().is_empty());
}
