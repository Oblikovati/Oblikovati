// SPDX-License-Identifier: GPL-2.0-only
//! Check member-scoped extrusion in the native rectangle-to-cube journey.
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};

pub(super) fn prepare(owner: &mut NativeDesktopOwner) -> u64 {
    let client = Client::new(owner.document_host());
    for name in ["Capture State", "Copy State"] {
        let member = client
            .model_states()
            .create(&CreateModelStateArgs {
                name: name.into(),
                ..Default::default()
            })
            .unwrap()
            .model_state;
        if name == "Copy State" {
            client
                .model_states()
                .activate(&RepRef { id: member.id })
                .unwrap();
            return member.id;
        }
    }
    panic!("expected Copy State to be created")
}

pub(super) fn check_switches(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    document: u64,
    variant: u64,
) {
    let branch = application_bdd::rendered_label_start("Model State");
    click_shell_position(window, owner, branch);
    activate_from_browser(window, owner, "Primary");
    assert_eq!(window.native.scene_status().unwrap().visible_bodies, 0);
    let state = owner.gui_command_state().unwrap();
    assert!(state.browser.unwrap().features[0].suppressed);
    activate_from_browser(window, owner, "Capture State");
    assert_eq!(window.native.scene_status().unwrap().visible_bodies, 0);
    assert!(owner.gui_command_state().unwrap().browser.unwrap().features[0].suppressed);
    activate_from_browser(window, owner, "Copy State");
    assert_eq!(window.native.scene_status().unwrap().visible_bodies, 1);
    let state = owner.gui_command_state().unwrap();
    assert!(!state.browser.unwrap().features[0].suppressed);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .model_states
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .id,
        variant
    );
    cube::verify(owner, document, 1.);
}
fn activate_from_browser(
    window: &mut ScriptedGuiWindow,
    owner: &mut NativeDesktopOwner,
    name: &str,
) {
    let position = application_bdd::rendered_label_center(name);
    drive_pointer_clicks(window, owner, position, 2);
}
