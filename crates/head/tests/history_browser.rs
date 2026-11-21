// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#[path = "owner/files.rs"]
mod files;
use files::{MemoryDocumentFiles, OwnerThreadGuids};
use oblikovati_api::{
    client::Client,
    types::{Variant, VariantValue},
    wire::SetPropertyArgs,
};
use oblikovati_automation::dispatch::DispatchWakeup;
use oblikovati_head::{DesktopDispatchOptions, DesktopObservations, DesktopOwner, HeadWork};
use oblikovati_session::files::FileWorkspace;

#[derive(Clone)]
struct HistoryTestWake;
impl DispatchWakeup for HistoryTestWake {
    fn wake(&self) -> Result<(), String> {
        Ok(())
    }
}
#[test]
fn history_browser_moves_the_document_cursor_and_refuses_an_invalid_position() {
    let observations = DesktopObservations::default();
    let workspace = FileWorkspace::new(observations.clone(), MemoryDocumentFiles::default());
    let mut owner = DesktopOwner::new(
        workspace,
        OwnerThreadGuids::default(),
        observations,
        "test".into(),
        HistoryTestWake,
        DesktopDispatchOptions::default(),
    );
    owner.create_part_document().unwrap();
    let document = owner.gui_command_state().unwrap().document.unwrap();
    Client::new(owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: oblikovati_model::property::USER_DEFINED.into(),
            name: "history-test".into(),
            value: Variant {
                value: VariantValue::String("edited".into()),
                ..Default::default()
            },
        })
        .unwrap();
    let before = owner.gui_command_state().unwrap();
    assert_eq!(before.histories[0].position, 1);
    assert_eq!(before.histories[0].entries.as_ref().unwrap().len(), 1);
    assert!(owner.jump_history_from_gui(document, 0).unwrap());
    let undone = owner.gui_command_state().unwrap();
    assert_eq!(undone.histories[0].position, 0);
    assert!(undone.undo_label.is_none() && undone.redo_label.is_some());
    assert!(owner.jump_history_from_gui(document, 2).is_err());
    assert_eq!(owner.gui_command_state().unwrap().histories[0].position, 0);
    assert!(owner.jump_history_from_gui(document, 1).unwrap());
    assert_eq!(owner.gui_command_state().unwrap().histories[0].position, 1);
}

struct HistoryActionWindow {
    document: u64,
    delivered: bool,
}
impl oblikovati_head::HeadWindow for HistoryActionWindow {
    fn gui_frame(
        &mut self,
        _: Option<&oblikovati_viewport::ViewportEvent>,
        _: &oblikovati_gui::GuiCommandState,
    ) -> Result<oblikovati_head::HeadGuiUpdate, String> {
        let actions = if std::mem::replace(&mut self.delivered, true) {
            Vec::new()
        } else {
            vec![oblikovati_gui::GuiShellAction::JumpHistoryTo {
                document: self.document,
                position: 0,
            }]
        };
        Ok(oblikovati_head::HeadGuiUpdate {
            actions,
            ..Default::default()
        })
    }
    fn submit_scene(&mut self, _: oblikovati_head::HeadSceneUpdate) -> Result<(), String> {
        Ok(())
    }
    fn events(&mut self, _: usize) -> Result<Vec<oblikovati_viewport::ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<oblikovati_head::HeadPresentation, String> {
        Ok(oblikovati_head::HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}

#[test]
fn history_browser_action_reaches_the_document_owner() {
    let observations = DesktopObservations::default();
    let workspace = FileWorkspace::new(observations.clone(), MemoryDocumentFiles::default());
    let mut owner = DesktopOwner::new(
        workspace,
        OwnerThreadGuids::default(),
        observations,
        "test".into(),
        HistoryTestWake,
        DesktopDispatchOptions::default(),
    );
    owner.create_part_document().unwrap();
    let document = owner.gui_command_state().unwrap().document.unwrap();
    Client::new(owner.document_host())
        .documents()
        .set_property(&SetPropertyArgs {
            document,
            set: oblikovati_model::property::USER_DEFINED.into(),
            name: "history-frame".into(),
            value: Variant {
                value: VariantValue::String("edited".into()),
                ..Default::default()
            },
        })
        .unwrap();
    let mut window = HistoryActionWindow {
        document,
        delivered: false,
    };
    oblikovati_head::run_window_with_owner(
        &mut window,
        oblikovati_head::HeadOptions { frames: Some(1) },
        &mut owner,
    )
    .unwrap();
    assert_eq!(owner.gui_command_state().unwrap().histories[0].position, 0);
}
