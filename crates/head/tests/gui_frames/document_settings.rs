// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn document_settings_acknowledges_noop_and_owner_failure() {
    for rejected in [false, true] {
        let settings = oblikovati_model::sketch_settings::default_sketch_settings();
        let action = GuiShellAction::SetDocumentSketchSettings {
            request_id: 11,
            document: 7,
            expected: settings.clone(),
            settings,
        };
        let mut window = FakeGuiWindow::default();
        window.packets.push_back(HeadGuiUpdate {
            actions: vec![action.clone()],
            ..Default::default()
        });
        let mut owner = FakeGuiOwner {
            settings_document: Some(7),
            settings_rejected: rejected,
            ..Default::default()
        };
        assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
        let result = if rejected {
            Err("settings rejected".into())
        } else {
            Ok(())
        };
        assert_eq!(window.constraint_results, [(action, Some(7), result)]);
    }
}

#[test]
fn changed_parent_draft_is_rejected_before_document_owner_is_called() {
    let settings = oblikovati_model::sketch_settings::default_sketch_settings();
    let action = GuiShellAction::SetDocumentSketchSettings {
        request_id: 12,
        document: 7,
        expected: settings.clone(),
        settings,
    };
    let mut window = FakeGuiWindow {
        constraint_preflight_rejected: true,
        ..Default::default()
    };
    window.packets.push_back(HeadGuiUpdate {
        actions: vec![action.clone()],
        ..Default::default()
    });
    let mut owner = FakeGuiOwner {
        settings_document: Some(7),
        ..Default::default()
    };
    assert!(advance_gui(&mut window, &mut owner, None).unwrap().changed);
    assert_eq!(owner.settings_calls, 0);
    assert_eq!(
        window.constraint_results,
        [(action, Some(7), Err("parent draft changed".into()))]
    );
}

#[test]
fn unavailable_gui_state_is_reported_before_document_owner_is_called() {
    let settings = oblikovati_model::sketch_settings::default_sketch_settings();
    let action = GuiShellAction::SetDocumentSketchSettings {
        request_id: 13,
        document: 7,
        expected: settings.clone(),
        settings,
    };
    let mut window = FakeGuiWindow::default();
    let mut owner = FakeGuiOwner {
        settings_state_rejected: true,
        ..Default::default()
    };
    assert!(dispatch_gui_action(&mut window, &mut owner, &action).unwrap());
    assert_eq!(owner.settings_calls, 0);
    assert_eq!(
        window.constraint_results,
        [(action, Some(7), Err("state unavailable".into()))]
    );
}
