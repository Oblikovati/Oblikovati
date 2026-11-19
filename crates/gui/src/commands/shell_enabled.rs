// SPDX-License-Identifier: GPL-2.0-only
//! Admission for an exact generation-scoped Shell face selection (#4106).
use super::*;

pub(super) fn open(state: &GuiCommandState) -> bool {
    state.certified_part_feature_available
        && state.document.is_some()
        && state.authoring == GuiAuthoringContext::Part
        && !state.transaction_open
}
pub(super) fn request(state: &GuiCommandState, request: &GuiShellFeatureRequest) -> bool {
    let selected: Vec<_> = state
        .selected_face_appearances
        .iter()
        .map(|(key, _)| key)
        .collect();
    open(state)
        && state.document == Some(request.document)
        && state.document_generation == Some(request.generation)
        && !selected.is_empty()
        && selected == request.faces.iter().collect::<Vec<_>>()
        && !request.thickness.trim().is_empty()
        && !request.thickness.contains('\0')
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn shell_requests_require_the_current_selection_and_generation() {
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: GuiAuthoringContext::Part,
            certified_part_feature_available: true,
            document_generation: Some(3),
            ..Default::default()
        };
        state.selected_face_appearances.push(("face".into(), false));
        let input = GuiShellFeatureRequest {
            document: 7,
            generation: 3,
            faces: vec!["face".into()],
            thickness: "1 mm".into(),
        };
        assert!(request(&state, &input));
        assert!(!state.fillet_available);
        assert!(state.fillet_selection.is_none());
        assert!(GuiShellAction::OpenShellFeature.enabled(&state));
        state.document_generation = Some(4);
        assert!(!request(&state, &input));
    }
}
