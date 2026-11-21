// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn creation_editor_preference_controls_draft_and_preserves_history() {
    for open_editor in [false, true] {
        let (mut owner, document, _) = geometric_owner(&[[0., 0.], [100., 40.]], false);
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        settings.edit_dimensions_when_created = open_editor;
        Client::new(owner.document_host())
            .documents()
            .set_sketch_settings(&oblikovati_api::wire::SetSketchSettingsArgs {
                document,
                settings,
            })
            .unwrap();
        start(&mut owner);
        deliver(
            &mut owner,
            &[
                button([0, 0], true),
                button([100, 40], true),
                button([50, 100], true),
            ],
        );
        let state = owner.gui_command_state().unwrap();
        assert!(state.dimension_tool_active);
        assert_eq!(state.dimension_draft.is_some(), open_editor);
        assert_eq!(dimensions(&mut owner).len(), 1);
        if let Some(draft) = state.dimension_draft {
            owner
                .dimension_tool_from_gui(&GuiDimensionAction::Cancel { id: draft.id })
                .unwrap();
        }
        assert!(owner.gui_command_state().unwrap().dimension_draft.is_none());
        Client::new(owner.document_host())
            .transactions()
            .undo()
            .unwrap();
        assert!(dimensions(&mut owner).is_empty());
        Client::new(owner.document_host())
            .transactions()
            .redo()
            .unwrap();
        assert_eq!(dimensions(&mut owner).len(), 1);
    }
}
