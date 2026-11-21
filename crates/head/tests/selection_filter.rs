// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
#[path = "owner/files.rs"]
mod files;
use files::{MemoryDocumentFiles, OwnerThreadGuids};
use oblikovati_automation::dispatch::DispatchWakeup;
use oblikovati_gui::{GuiSelectionKind, GuiSelectionMode};
use oblikovati_head::{DesktopDispatchOptions, DesktopObservations, DesktopOwner, HeadWork};
use oblikovati_session::files::FileWorkspace;

#[derive(Clone)]
struct TestWake;
impl DispatchWakeup for TestWake {
    fn wake(&self) -> Result<(), String> {
        Ok(())
    }
}

#[test]
fn owner_filter_changes_are_visible_ordered_and_refuse_invalid_requests() {
    let observations = DesktopObservations::default();
    let workspace = FileWorkspace::new(observations.clone(), MemoryDocumentFiles::default());
    let mut owner = DesktopOwner::new(
        workspace,
        OwnerThreadGuids::default(),
        observations,
        "test".into(),
        TestWake,
        DesktopDispatchOptions::default(),
    );
    assert!(
        owner
            .set_selection_enabled_from_gui(GuiSelectionKind::Face, false)
            .is_err()
    );
    owner.create_part_document().unwrap();
    assert!(
        owner
            .set_selection_enabled_from_gui(GuiSelectionKind::Face, false)
            .unwrap()
    );
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.selection_mode, GuiSelectionMode::Custom);
    let document = state.document.unwrap();
    assert!(
        !owner
            .click_model_from_gui(document, [[0., 0., 1.], [0., 0., -1.]])
            .unwrap()
    );
    let filter = state.selection_filter.unwrap();
    assert_eq!(filter.entries.len(), 12);
    assert!(
        !filter
            .entries
            .iter()
            .find(|entry| entry.kind == GuiSelectionKind::Face)
            .unwrap()
            .enabled
    );
    assert!(owner.move_selection_kind_from_gui(7, 0).unwrap());
    assert_eq!(
        owner
            .gui_command_state()
            .unwrap()
            .selection_filter
            .unwrap()
            .entries[0]
            .kind,
        GuiSelectionKind::Face
    );
    assert!(owner.move_selection_kind_from_gui(0, 12).is_err());
    assert!(
        owner
            .set_selection_mode_from_gui(GuiSelectionMode::All)
            .unwrap()
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .selection_filter
            .unwrap()
            .entries
            .iter()
            .all(|entry| entry.enabled)
    );
}

#[derive(Default)]
struct FilterActionWindow {
    delivered: bool,
}
impl oblikovati_head::HeadWindow for FilterActionWindow {
    fn gui_frame(
        &mut self,
        _: Option<&oblikovati_viewport::ViewportEvent>,
        _: &oblikovati_gui::GuiCommandState,
    ) -> Result<oblikovati_head::HeadGuiUpdate, String> {
        let actions = if std::mem::replace(&mut self.delivered, true) {
            Vec::new()
        } else {
            vec![oblikovati_gui::GuiShellAction::SetSelectionEnabled {
                kind: GuiSelectionKind::Face,
                enabled: false,
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
fn gui_filter_action_reaches_the_document_owner() {
    let observations = DesktopObservations::default();
    let workspace = FileWorkspace::new(observations.clone(), MemoryDocumentFiles::default());
    let mut owner = DesktopOwner::new(
        workspace,
        OwnerThreadGuids::default(),
        observations,
        "test".into(),
        TestWake,
        DesktopDispatchOptions::default(),
    );
    owner.create_part_document().unwrap();
    let mut window = FilterActionWindow::default();
    oblikovati_head::run_window_with_owner(
        &mut window,
        oblikovati_head::HeadOptions { frames: Some(1) },
        &mut owner,
    )
    .unwrap();
    let state = owner.gui_command_state().unwrap();
    assert_eq!(state.selection_mode, GuiSelectionMode::Custom);
    assert!(
        !state
            .selection_filter
            .unwrap()
            .entries
            .iter()
            .find(|entry| entry.kind == GuiSelectionKind::Face)
            .unwrap()
            .enabled
    );
}
