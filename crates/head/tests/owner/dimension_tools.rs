// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    geometric_tools::{geometric_owner, inspection},
    slot_input::deliver,
};
use crate::placement_test_inputs::*;
use oblikovati_gui::{GuiDimensionAction, GuiDimensionDraft};
use oblikovati_model::sketch::SketchPlacementRecipe;
use oblikovati_viewport::ViewportKey;

fn start(owner: &mut TestOwner) {
    owner
        .dimension_tool_from_gui(&GuiDimensionAction::Start)
        .unwrap();
}
fn draft(owner: &TestOwner) -> GuiDimensionDraft {
    owner.gui_command_state().unwrap().dimension_draft.unwrap()
}
fn apply(owner: &mut TestOwner, expression: &str, driven: bool) -> Result<bool, String> {
    owner.dimension_tool_from_gui(&GuiDimensionAction::Apply {
        id: draft(owner).id,
        expression: expression.into(),
        driven,
    })
}
fn dimensions(owner: &mut TestOwner) -> Vec<DimensionInfo> {
    Client::new(owner.document_host())
        .sketch()
        .dimensions(&SketchArgs::default())
        .unwrap()
        .dimensions
        .unwrap_or_default()
}
fn point_draft(owner: &mut TestOwner, at: [i32; 2]) {
    start(owner);
    deliver(owner, &[button([0, 0], true), button([100, 40], true)]);
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(dimensions(owner).is_empty());
    deliver(owner, &[button(at, true)]);
}
#[test]
fn point_dimension_keeps_history_until_placement_and_selects_the_drag_orientation() {
    for (at, orientation, expression) in [
        ([50, 100], "horizontal", "10 cm"),
        ([150, 20], "vertical", "10 cm"),
        ([10, 120], "aligned", "10 cm"),
    ] {
        let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
        let before = owner.gui_command_state().unwrap().undo_label;
        point_draft(&mut owner, at);
        let state = draft(&owner);
        assert_eq!(state.kind, "distance");
        assert_eq!(state.orientation, orientation);
        assert!(!state.expression.is_empty());
        assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        apply(&mut owner, expression, false).unwrap();
        assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
        assert!(inspection(&mut owner).healthy);
        assert_eq!(inspection(&mut owner).dof, 3);
        let rows = dimensions(&mut owner);
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].text_point, Some(at.map(f64::from).to_vec()));
        assert!((rows[0].value - 10.).abs() < 1e-8);
    }
}
#[test]
fn a_second_line_remains_reachable_before_the_placement_click() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 0.], [120., 100.]], true);
    start(&mut owner);
    deliver(&mut owner, &[button([50, 0], true)]);
    assert!(dimensions(&mut owner).is_empty());
    deliver(
        &mut owner,
        &[button([110, 50], true), button([200, 200], true)],
    );
    assert_eq!(draft(&owner).kind, "angle");
    apply(&mut owner, "90 deg", false).unwrap();
    let rows = dimensions(&mut owner);
    assert_eq!(rows[0].kind, "angle");
    assert!((rows[0].value - std::f64::consts::FRAC_PI_2).abs() < 1e-8);
    assert!(inspection(&mut owner).healthy);
}
#[test]
fn a_single_line_measures_its_endpoints_with_a_measured_default() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], true);
    start(&mut owner);
    deliver(
        &mut owner,
        &[button([50, 20], true), button([50, 100], true)],
    );
    assert_eq!(draft(&owner).orientation, "horizontal");
    apply(&mut owner, "", false).unwrap();
    assert!((dimensions(&mut owner)[0].value - 100.).abs() < 1e-8);
    assert!(inspection(&mut owner).healthy);
}
#[test]
fn circular_outlines_create_radius_dimensions_and_reference_keeps_freedom() {
    for (recipe, pixel) in [
        (
            SketchPlacementRecipe::circle([0.; 2], 100.).unwrap(),
            [100, 0],
        ),
        (
            SketchPlacementRecipe::arc([0.; 2], [[100., 0.], [0., 100.]], true).unwrap(),
            [70, 70],
        ),
    ] {
        let (mut owner, document, sketch) = geometric_owner(&[], false);
        owner
            .commit_sketch_placement(document, sketch, &recipe, &[])
            .unwrap();
        let dof = inspection(&mut owner).dof;
        start(&mut owner);
        deliver(&mut owner, &[button(pixel, true), button([200, 200], true)]);
        assert_eq!(draft(&owner).kind, "radius");
        apply(&mut owner, "999 cm", true).unwrap();
        let rows = dimensions(&mut owner);
        assert!(rows[0].driven);
        assert!((rows[0].value - 100.).abs() < 1e-8);
        assert_eq!(inspection(&mut owner).dof, dof);
    }
}
#[test]
fn cancellation_replacement_and_stale_dialog_actions_never_create_dimensions() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    point_draft(&mut owner, [50, 100]);
    let old = draft(&owner).id;
    owner
        .dimension_tool_from_gui(&GuiDimensionAction::Cancel { id: old })
        .unwrap();
    assert!(dimensions(&mut owner).is_empty());
    point_draft(&mut owner, [50, 100]);
    assert_ne!(draft(&owner).id, old);
    assert!(
        owner
            .dimension_tool_from_gui(&GuiDimensionAction::Apply {
                id: old,
                expression: "10 cm".into(),
                driven: false
            })
            .is_err()
    );
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(dimensions(&mut owner).is_empty());
    point_draft(&mut owner, [50, 100]);
    owner
        .constrain_sketch_from_gui(oblikovati_gui::GuiSketchConstraint::Horizontal)
        .unwrap();
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(dimensions(&mut owner).is_empty());
}
#[test]
fn a_parameter_expression_drives_the_dimension_through_undo_and_redo() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    Client::new(owner.document_host())
        .parameters()
        .add(&oblikovati_api::wire::ParameterSetArgs {
            name: "size".into(),
            expression: "10 cm".into(),
            ..Default::default()
        })
        .unwrap();
    point_draft(&mut owner, [50, 100]);
    apply(&mut owner, "size", false).unwrap();
    let parameter = dimensions(&mut owner)[0].name.clone();
    Client::new(owner.document_host())
        .parameters()
        .set(&oblikovati_api::wire::ParameterSetArgs {
            name: "size".into(),
            expression: "20 cm".into(),
            ..Default::default()
        })
        .unwrap();
    assert!((dimensions(&mut owner)[0].value - 20.).abs() < 1e-8);
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!((dimensions(&mut owner)[0].value - 10.).abs() < 1e-8);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert!((dimensions(&mut owner)[0].value - 20.).abs() < 1e-8);
    assert_eq!(dimensions(&mut owner)[0].name, parameter);
    assert!(inspection(&mut owner).healthy);
}
#[test]
fn invalid_expression_keeps_the_same_draft_for_correction() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    point_draft(&mut owner, [50, 100]);
    let id = draft(&owner).id;
    assert!(apply(&mut owner, "1 +", false).is_err());
    assert_eq!(draft(&owner).id, id);
    assert!(dimensions(&mut owner).is_empty());
    apply(&mut owner, "10 cm", false).unwrap();
    assert_eq!(dimensions(&mut owner).len(), 1);
}

#[test]
fn incompatible_dimension_keeps_fixed_geometry_and_the_owner_draft() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    for point in [1, 2] {
        Client::new(owner.document_host())
            .constrain()
            .add(&oblikovati_api::wire::AddConstraintArgs {
                kind: "fix".into(),
                entities: Some(vec![point]),
                ..Default::default()
            })
            .unwrap();
    }
    assert!(inspection(&mut owner).healthy);
    assert_eq!(inspection(&mut owner).dof, 0);
    point_draft(&mut owner, [50, 100]);
    let id = draft(&owner).id;
    let result = apply(&mut owner, "10 cm", false);
    assert!(result.is_err(), "{result:?}: {:?}", inspection(&mut owner));
    assert_eq!(draft(&owner).id, id);
    assert!(dimensions(&mut owner).is_empty());
    assert!(inspection(&mut owner).healthy);
}

#[test]
fn cursor_preview_and_authored_labels_do_not_create_extra_history() {
    use oblikovati_gui::GuiDimensionLabelRole::{Driving, Preview};
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    let before = owner.gui_command_state().unwrap().undo_label;
    start(&mut owner);
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([100, 40], true),
            motion([50, 100]),
        ],
    );
    let pending = owner.sketch_dimension_annotations().unwrap();
    assert!(pending.declines.is_empty());
    assert_eq!(pending.labels.len(), 1);
    assert_eq!(pending.labels[0].role, Preview);
    assert_eq!(pending.labels[0].position, [50., 100.]);
    assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
    deliver(&mut owner, &[button([50, 100], true)]);
    apply(&mut owner, "10 cm", false).unwrap();
    let authored = owner.sketch_dimension_annotations().unwrap();
    assert_eq!(authored.labels.len(), 1);
    assert_eq!(authored.labels[0].role, Driving);
    assert_eq!(authored.labels[0].position, [50., 100.]);
    assert!(authored.labels[0].text.starts_with("d0 = "));
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        owner
            .sketch_dimension_annotations()
            .unwrap()
            .labels
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(owner.sketch_dimension_annotations().unwrap(), authored);
}
#[test]
fn history_changes_retire_placed_dimension_dialogs_and_selection() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    point_draft(&mut owner, [50, 100]);
    let old = draft(&owner).id;
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        owner
            .dimension_tool_from_gui(&GuiDimensionAction::Apply {
                id: old,
                expression: "10 cm".into(),
                driven: false
            })
            .is_err()
    );
    deliver(&mut owner, &[]);
    assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
    assert!(
        owner
            .sketch_dimension_annotations()
            .unwrap()
            .labels
            .is_empty()
    );
    assert!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .kinds
            .unwrap_or_default()
            .is_empty()
    );
}

#[test]
fn programmatic_unavailable_sources_keep_their_health_visible_in_annotations() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    Client::new(owner.document_host())
        .dimension()
        .add(&oblikovati_api::wire::AddDimensionArgs {
            kind: "distance".into(),
            entities: Some(vec![1, 2]),
            expression: "futureWidth".into(),
            text_point: Some(vec![50., 100.]),
            ..Default::default()
        })
        .unwrap();
    let error = Client::new(owner.document_host())
        .sketch()
        .get(&SketchArgs::default())
        .unwrap_err();
    assert!(error.to_string().contains("futureWidth"));
    let annotations = owner.sketch_dimension_annotations().unwrap();
    assert!(annotations.declines.is_empty());
    assert_eq!(annotations.labels.len(), 1);
    assert!(annotations.labels[0].text.contains("source unavailable"));
}

#[test]
fn redundant_dimension_keeps_the_editor_until_explicit_reference_correction() {
    let (mut owner, _, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    point_draft(&mut owner, [50, 100]);
    apply(&mut owner, "100 cm", false).unwrap();
    let before = dimensions(&mut owner);
    start(&mut owner);
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([100, 40], true),
            button([50, 100], true),
        ],
    );
    let id = draft(&owner).id;
    let cause = apply(&mut owner, "100 cm", false).unwrap_err();
    assert!(cause.contains("redundant"), "{cause}");
    assert_eq!(draft(&owner).id, id);
    assert_eq!(dimensions(&mut owner), before);
    assert!(inspection(&mut owner).healthy);
    apply(&mut owner, "100 cm", true).unwrap();
    assert!(dimensions(&mut owner)[1].driven);
    assert!(inspection(&mut owner).healthy);
}

#[test]
fn hidden_sketch_dimensions_leave_the_overlay_and_return_after_show() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 40.]], false);
    point_draft(&mut owner, [50, 100]);
    apply(&mut owner, "10 cm", false).unwrap();
    let authored = owner.sketch_dimension_annotations().unwrap();
    assert_eq!(authored.labels.len(), 1);
    owner
        .set_sketch_visibility_from_gui(document, sketch, false)
        .unwrap();
    assert!(
        owner
            .sketch_dimension_annotations()
            .unwrap()
            .labels
            .is_empty()
    );
    assert_eq!(dimensions(&mut owner).len(), 1);
    owner
        .set_sketch_visibility_from_gui(document, sketch, true)
        .unwrap();
    assert_eq!(owner.sketch_dimension_annotations().unwrap(), authored);
}
