// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    geometric_tools::{geometric_owner, inspection},
    slot_input::deliver,
};
use crate::placement_test_inputs::*;
use oblikovati_gui::GuiSketchConstraint::Fix;

#[test]
fn fix_point_and_line_pin_exact_positions_in_one_undo_step() {
    for connected in [false, true] {
        let (mut owner, document, sketch) = geometric_owner(&[[1., 2.], [101., 42.]], connected);
        let before = owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .points()
            .to_vec();
        let history = owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .undo_labels()
            .len();
        owner.constrain_sketch_from_gui(Fix).unwrap();
        let pixel = if connected { [51, 22] } else { [1, 2] };
        deliver(&mut owner, &[button(pixel, true)]);
        assert!(owner.sketch_tool.is_none());
        assert_eq!(inspection(&mut owner).dof, if connected { 0 } else { 2 });
        assert_eq!(
            owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .undo_labels()
                .len(),
            history + 1
        );
        assert_eq!(
            owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .snapshot()
                .sketches
                .get(sketch)
                .unwrap()
                .geometry()
                .points(),
            before
        );
        Client::new(owner.document_host())
            .transactions()
            .undo()
            .unwrap();
        assert_eq!(inspection(&mut owner).dof, 4);
        Client::new(owner.document_host())
            .transactions()
            .redo()
            .unwrap();
        assert_eq!(inspection(&mut owner).dof, if connected { 0 } else { 2 });
    }
}

#[test]
fn partial_fix_failure_restores_geometry_and_keeps_the_tool_usable() {
    let (mut owner, document, sketch) = geometric_owner(&[[1., 2.], [101., 42.]], true);
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let point = before.get(sketch).unwrap().geometry().points()[0].id;
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    owner.constrain_sketch_from_gui(Fix).unwrap();
    let Some(crate::sketch_tool::SketchToolState::Geometric(mut tool)) = owner.sketch_tool.take()
    else {
        panic!("expected Fix tool")
    };
    let requests: Vec<AddConstraintArgs> = [point.0, 999]
        .into_iter()
        .map(|id| AddConstraintArgs {
            kind: oblikovati_api::types::GEO_CONSTRAINT_FIX.0.to_string(),
            entities: Some(vec![id]),
            ..Default::default()
        })
        .collect();
    assert!(
        owner
            .apply_geometric_requests(&mut tool, &requests)
            .unwrap_err()
            .contains("999")
    );
    let session = owner.files.workspace().session(document).unwrap();
    assert!(!session.in_transaction());
    assert!(session.snapshot().sketches.same_recipe(&before));
    assert_eq!(session.undo_labels().len(), history);
    assert_eq!(tool.generation, session.generation());
    owner.sketch_tool = Some(crate::sketch_tool::SketchToolState::Geometric(tool));
    deliver(&mut owner, &[button([51, 22], true)]);
    assert!(owner.sketch_tool.is_none());
    assert_eq!(inspection(&mut owner).dof, 0);
}
#[test]
fn fix_cancellation_and_empty_clicks_leave_history_unchanged() {
    let (mut owner, document, _) = geometric_owner(&[[1., 2.]], false);
    let history = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner.constrain_sketch_from_gui(Fix).unwrap();
    deliver(&mut owner, &[button([999, 999], true)]);
    assert!(owner.sketch_tool.is_some());
    deliver(
        &mut owner,
        &[key(oblikovati_viewport::ViewportKey::Cancel, true)],
    );
    assert!(owner.sketch_tool.is_none());
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        history
    );
}
