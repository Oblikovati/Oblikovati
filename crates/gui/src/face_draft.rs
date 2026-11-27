// SPDX-License-Identifier: GPL-2.0-only
//! Shared preview lifecycle for face-based feature drafts (#4106).
use crate::GuiShellAction;

pub(crate) fn refresh_preview<T: Clone + PartialEq>(
    attempted: &mut Option<T>,
    request: Option<T>,
    actions: &mut Vec<GuiShellAction>,
    make_action: impl Fn(Option<T>) -> GuiShellAction,
) {
    if request == *attempted {
        return;
    }
    if attempted.is_some() {
        actions.push(make_action(None));
    }
    *attempted = request.clone();
    if let Some(request) = request {
        actions.push(make_action(Some(request)));
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn changed_request_clears_old_preview_before_requesting_the_new_one() {
        let mut attempted = Some(1_u32);
        let mut calls = Vec::new();
        refresh_preview(&mut attempted, Some(2), &mut calls, |value| {
            GuiShellAction::PreviewShellFeature(value.map(|id| crate::GuiShellFeatureRequest {
                document: u64::from(id),
                generation: 1,
                faces: vec!["face".into()],
                thickness: "1 mm".into(),
            }))
        });
        assert_eq!(calls.len(), 2);
        assert_eq!(calls[0], GuiShellAction::PreviewShellFeature(None));
        assert_eq!(attempted, Some(2));
    }
}
