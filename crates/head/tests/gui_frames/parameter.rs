// SPDX-License-Identifier: GPL-2.0-only
//! GUI frame handling of parameter views and edits.
use super::*;

#[test]
fn parameter_owner_results_are_local_and_metadata_is_requested_only_when_open() {
    let action = oblikovati_gui::GuiParameterAction::Toggle {
        document: 7,
        parameter: 11,
        field: oblikovati_gui::GuiParameterToggleField::Key,
        value: true,
    };
    let mut window = FakeGuiWindow::default();
    let mut owner = FakeGuiOwner {
        parameter_failure: true,
        ..Default::default()
    };
    window.packets.push_back(HeadGuiUpdate {
        parameter_actions: vec![action.clone()],
        ..Default::default()
    });
    assert!(!advance_gui(&mut window, &mut owner, None).unwrap().changed);
    assert!(window.parameter_views.is_empty());
    assert!(window.parameter_results[0].is_err());
    window.parameters_open = true;
    owner.parameter_failure = false;
    window.packets.push_back(HeadGuiUpdate {
        parameter_actions: vec![action.clone()],
        ..Default::default()
    });
    assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    assert_eq!(window.parameter_views, vec![None]);
    assert_eq!(window.parameter_results[1], Ok(()));
    assert_eq!(owner.parameter_edits, vec![action.clone(), action]);
}

#[test]
fn failed_parameter_views_are_local_and_do_not_repeat_idle_redraws() {
    for producer_failure in [true, false] {
        let mut owner = FakeGuiOwner {
            parameter_view_failure: producer_failure,
            ..Default::default()
        };
        let mut window = FakeGuiWindow {
            parameters_open: true,
            parameter_view_decline: !producer_failure,
            ..Default::default()
        };
        assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
        assert_eq!(window.parameter_issues.len(), 1);
        assert!(!advance_gui(&mut window, &mut owner, None).unwrap().changed);
        assert_eq!(window.parameter_issues.len(), 2);
        assert!(window.states.len() == 2);
    }
}
