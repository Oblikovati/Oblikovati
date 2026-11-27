// SPDX-License-Identifier: GPL-2.0-only
//! Keep inference dialog state and desktop actions together (#4107).
use super::*;
pub(super) fn draw(
    dialog: &mut crate::sketch::inference_scope::InferenceScopeDialog,
    ui: &imgui::Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    open: bool,
) {
    if open {
        append_inference_actions(actions, dialog.begin(state));
    }
    append_inference_actions(actions, dialog.synchronize(state));
    let selected = state
        .sketch_selection
        .entities
        .iter()
        .filter(|(sketch, _)| Some(*sketch) == state.active_sketch)
        .map(|(_, entity)| entity.0)
        .collect::<Vec<_>>();
    append_inference_actions(
        actions,
        dialog.panel(ui, &selected, state.sketch_inference_scope_error.as_deref()),
    );
}

fn append_inference_actions(
    actions: &mut Vec<GuiShellAction>,
    events: Vec<crate::sketch::inference_scope::InferenceScopeEvent>,
) {
    use crate::sketch::inference_scope::InferenceScopeEvent;
    actions.extend(events.into_iter().map(|event| match event {
        InferenceScopeEvent::BeginSelection(ids) => {
            GuiShellAction::SelectInferenceGeometry(Some(ids))
        }
        InferenceScopeEvent::EndSelection => GuiShellAction::SelectInferenceGeometry(None),
        InferenceScopeEvent::Apply(scope) => GuiShellAction::SetInferenceScope(scope),
    }));
}
