// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn prompt_choice_preserves_history_on_cancel_and_creates_driven_on_accept() {
    for accept in [false, true] {
        let (mut owner, document, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
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
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        settings.over_constrained_behavior = oblikovati_api::types::OVER_CONSTRAINED_PROMPT;
        Client::new(owner.document_host())
            .documents()
            .set_sketch_settings(&oblikovati_api::wire::SetSketchSettingsArgs {
                document,
                settings,
            })
            .unwrap();
        let before = owner.gui_command_state().unwrap().undo_label;
        point_draft(&mut owner, [50, 100]);
        let pending = draft(&mut owner);
        assert!(pending.choice_required);
        assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        let action = if accept {
            GuiDimensionAction::Apply {
                id: pending.id,
                expression: pending.expression,
                driven: true,
            }
        } else {
            GuiDimensionAction::Cancel { id: pending.id }
        };
        owner.dimension_tool_from_gui(&action).unwrap();
        let result = dimensions(&mut owner);
        assert_eq!(result.len(), usize::from(accept));
        if accept {
            assert!(result[0].driven);
            assert_dimension_undo_redo(&mut owner, 1);
        } else {
            assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        }
    }
}

#[test]
fn locked_placement_choice_replays_atomically_and_can_be_cancelled() {
    use oblikovati_model::history::SketchPlacementFieldLock;
    for accept in [false, true] {
        let (mut owner, document, sketch) = geometric_owner(&[], false);
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        settings.infer_constraints = false;
        settings.over_constrained_behavior = oblikovati_api::types::OVER_CONSTRAINED_PROMPT;
        Client::new(owner.document_host())
            .documents()
            .set_sketch_settings(&oblikovati_api::wire::SetSketchSettingsArgs {
                document,
                settings,
            })
            .unwrap();
        owner.start_slot_placement(document, sketch, 2.0).unwrap();
        let before = owner.gui_command_state().unwrap().undo_label;
        let mut recipe = SketchPlacementRecipe::straight_slot([[0., 0.], [10., 0.]], 2.).unwrap();
        recipe.fields.push(recipe.fields[0].clone());
        let lock = |expression: &str| {
            Some(SketchPlacementFieldLock {
                expression: expression.into(),
                text_point: None,
                reference: false,
            })
        };
        let locks = vec![lock("10 cm"), None, None, lock("20 cm")];
        let Some(crate::sketch_tool::SketchToolState::Placement(mut tool)) =
            owner.sketch_tool.take()
        else {
            panic!("expected placement");
        };
        owner
            .commit_pending_recipe(&mut tool, recipe, locks)
            .unwrap();
        owner.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(tool));
        let pending = draft(&mut owner);
        assert!(pending.choice_required);
        assert!(dimensions(&mut owner).is_empty());
        assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        let action = if accept {
            GuiDimensionAction::Apply {
                id: pending.id,
                expression: pending.expression,
                driven: true,
            }
        } else {
            GuiDimensionAction::Cancel { id: pending.id }
        };
        owner.dimension_tool_from_gui(&action).unwrap();
        let rows = dimensions(&mut owner);
        assert_eq!(rows.len(), if accept { 2 } else { 0 });
        if accept {
            assert!(rows[1].driven);
            assert_dimension_undo_redo(&mut owner, 2);
        } else {
            assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        }
    }
}

#[test]
fn line_locked_length_prompt_accepts_without_silent_reference() {
    use oblikovati_model::history::{SketchLineChainCreation, SketchPlacementFieldLock};
    for accept in [false, true] {
        let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 40.]], false);
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
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        settings.over_constrained_behavior = oblikovati_api::types::OVER_CONSTRAINED_PROMPT;
        Client::new(owner.document_host())
            .documents()
            .set_sketch_settings(&oblikovati_api::wire::SetSketchSettingsArgs {
                document,
                settings,
            })
            .unwrap();
        owner
            .start_sketch_placement(document, sketch, crate::SketchPlacementTool::Line)
            .unwrap();
        let before = owner.gui_command_state().unwrap().undo_label;
        let input = SketchLineChainCreation {
            points: vec![[0., 0.], [100., 40.]],
            closed: false,
            construction: false,
            centerline: false,
            joins: [
                Some(oblikovati_model::sketch::SketchEntityId(1)),
                Some(oblikovati_model::sketch::SketchEntityId(2)),
            ],
            locks: [
                Some(SketchPlacementFieldLock {
                    expression: "107.70329614269 cm".into(),
                    text_point: None,
                    reference: false,
                }),
                None,
            ],
        };
        let Some(crate::sketch_tool::SketchToolState::Placement(mut tool)) =
            owner.sketch_tool.take()
        else {
            panic!("expected placement");
        };
        owner.commit_line_input(&mut tool, &input, None).unwrap();
        owner.sketch_tool = Some(crate::sketch_tool::SketchToolState::Placement(tool));
        let pending = draft(&mut owner);
        assert!(pending.choice_required);
        assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        let action = if accept {
            GuiDimensionAction::Apply {
                id: pending.id,
                expression: pending.expression,
                driven: true,
            }
        } else {
            GuiDimensionAction::Cancel { id: pending.id }
        };
        owner.dimension_tool_from_gui(&action).unwrap();
        let rows = dimensions(&mut owner);
        assert_eq!(rows.len(), usize::from(accept));
        if accept {
            assert!(rows[0].driven);
            assert_dimension_undo_redo(&mut owner, 1);
        } else {
            assert_eq!(owner.gui_command_state().unwrap().undo_label, before);
        }
    }
}

fn assert_dimension_undo_redo(owner: &mut TestOwner, expected: usize) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(dimensions(owner).is_empty());
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(dimensions(owner).len(), expected);
}
