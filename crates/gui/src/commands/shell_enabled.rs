// SPDX-License-Identifier: GPL-2.0-only
//! Admission for an exact generation-scoped Shell face selection (#4106).
use super::*;

pub(super) fn request(state: &GuiCommandState, request: &GuiShellFeatureRequest) -> bool {
    let selected: Vec<_> = state
        .selected_face_appearances
        .iter()
        .map(|(key, _)| key)
        .collect();
    hole_enabled::open(state)
        && state.document == Some(request.document)
        && state
            .fillet_selection
            .as_ref()
            .is_some_and(|(generation, _)| *generation == request.generation)
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
            fillet_available: true,
            fillet_selection: Some((3, vec![])),
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
        state.fillet_selection = Some((4, vec![]));
        assert!(!request(&state, &input));
    }
}
