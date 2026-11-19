// SPDX-License-Identifier: GPL-2.0-only
//! Placed dimension draft editing through the ribbon.
use super::*;

#[test]
fn placed_dimension_uses_value_editor_and_cancel_retains_owner_draft() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    request.extent = [1200, 800];
    enter_part(&mut request);
    request.commands.authoring = GuiAuthoringContext::Sketch;
    request.commands.dimension_tool_active = true;
    let edit = oblikovati_gui::GuiDimensionEdit {
        sketch: 1,
        constraint: 2,
        name: "d0".into(),
        expression: "10 mm".into(),
        driven: false,
    };
    request.commands.dimension_draft = Some(GuiDimensionDraft {
        choice_required: false,
        id: 42,
        kind: "distance".into(),
        orientation: "horizontal".into(),
        expression: "10 mm".into(),
        committed: Some(edit.clone()),
        refusal: None,
    });
    settle_ribbon(&mut engine, &mut request);
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Enter),
        [GuiShellAction::EditDimensionValue(
            oblikovati_gui::GuiDimensionValue {
                sketch: edit.sketch,
                constraint: edit.constraint,
                expression: edit.expression,
            }
        )]
    );
    request.commands.dimension_draft.as_mut().unwrap().id = 43;
    settle_ribbon(&mut engine, &mut request);
    assert_eq!(
        dimension_editor_key(&mut engine, &mut request, GuiKey::Escape),
        [GuiShellAction::Dimension(GuiDimensionAction::Cancel {
            id: 43
        })]
    );
}

#[test]
fn overconstraint_prompt_accepts_driven_or_cancels() {
    for (key, expected) in [
        (
            GuiKey::Enter,
            GuiDimensionAction::Apply {
                id: 42,
                expression: "10 mm".into(),
                driven: true,
            },
        ),
        (GuiKey::Escape, GuiDimensionAction::Cancel { id: 42 }),
    ] {
        let mut engine = GuiFrameEngine::new().unwrap();
        let mut request = ribbon_request();
        request.extent = [1200, 800];
        enter_part(&mut request);
        request.commands.authoring = GuiAuthoringContext::Sketch;
        request.commands.dimension_tool_active = true;
        request.commands.dimension_draft = Some(GuiDimensionDraft {
            id: 42,
            kind: "distance".into(),
            orientation: "horizontal".into(),
            expression: "10 mm".into(),
            committed: None,
            refusal: Some("This dimension would over-constrain the sketch.".into()),
            choice_required: true,
        });
        settle_ribbon(&mut engine, &mut request);
        assert_eq!(
            dimension_editor_key(&mut engine, &mut request, key),
            [GuiShellAction::Dimension(expected)]
        );
    }
}
