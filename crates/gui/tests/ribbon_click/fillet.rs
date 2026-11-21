// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn selected_edge_toolbar_opens_preview_and_escape_cancels() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.fillet_available = true;
    request.commands.selected_edge_count = 1;
    request.commands.fillet_selection = Some((3, vec![vec![1, 2, 3]]));
    request.commands.feature_edges = vec![[[500, 300], [550, 300]]];
    settle_ribbon(&mut engine, &mut request);
    let actions = click_ribbon(&mut engine, &mut request, [515., 270.]);
    assert!(
        matches!(actions.as_slice(), [GuiShellAction::PreviewFillet(Some(draft))] if draft.document == 7 && draft.radius == "2 mm" && draft.edges == vec![vec![1, 2, 3]]),
        "actions: {actions:?}"
    );
    let actions = dimension_editor_key(&mut engine, &mut request, GuiKey::Escape);
    assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
}

#[test]
fn selected_edge_toolbar_commits_with_enter() {
    let mut engine = GuiFrameEngine::new().unwrap();
    let mut request = ribbon_request();
    enter_part(&mut request);
    request.commands.fillet_available = true;
    request.commands.selected_edge_count = 1;
    request.commands.fillet_selection = Some((3, vec![vec![1, 2, 3]]));
    request.commands.feature_edges = vec![[[500, 300], [550, 300]]];
    settle_ribbon(&mut engine, &mut request);
    click_ribbon(&mut engine, &mut request, [515., 270.]);
    let actions = dimension_editor_key(&mut engine, &mut request, GuiKey::Enter);
    assert!(
        matches!(actions.as_slice(), [GuiShellAction::CreateFillet(draft)] if draft.generation == 3 && draft.radius == "2 mm")
    );
}
