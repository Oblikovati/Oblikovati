// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::ui_scale::{UiScaleOptions, UiScalePreferenceAccess};
use std::{cell::RefCell, rc::Rc};
type RecordedScaleWrites = Rc<RefCell<Vec<UiScaleOptions>>>;
struct RecordedUiScaleStore {
    current: UiScaleOptions,
    writes: RecordedScaleWrites,
    refuse: bool,
}
impl UiScalePreferenceAccess for RecordedUiScaleStore {
    fn ui_scale(&self) -> UiScaleOptions {
        self.current
    }
    fn set_ui_scale(&mut self, options: UiScaleOptions) -> Result<(), String> {
        if self.refuse {
            return Err(format!(
                "UI scale {options:?}: expected writable options.yaml"
            ));
        }
        self.writes.borrow_mut().push(options);
        self.current = options;
        Ok(())
    }
}
fn scale_owner(refuse: bool) -> (TestOwner, RecordedScaleWrites) {
    let (owner, _) = owner();
    let writes = RecordedScaleWrites::default();
    let store = RecordedUiScaleStore {
        current: UiScaleOptions::default(),
        writes: writes.clone(),
        refuse,
    };
    (owner.with_ui_scale_preferences(Box::new(store)), writes)
}
#[test]
fn scale_owner_persists_before_publication_without_document_history() {
    let (mut owner, writes) = scale_owner(false);
    let initial = UiScaleOptions::default();
    assert!(!owner.set_ui_scale_from_gui(initial).unwrap());
    let options = UiScaleOptions::new(2., 1.5).unwrap();
    assert!(owner.set_ui_scale_from_gui(options).unwrap());
    assert_eq!(owner.gui_command_state().unwrap().ui_scale, Some(options));
    assert_eq!(*writes.borrow(), [options]);
    assert!(owner.documents().workspace().documents().is_empty());
    owner.create_part_document().unwrap();
    assert!(owner.gui_command_state().unwrap().undo_label.is_none());
    assert_eq!(owner.gui_command_state().unwrap().ui_scale, Some(options));
}
#[test]
fn failed_scale_save_preserves_the_visible_values() {
    let (mut owner, _) = scale_owner(true);
    let options = UiScaleOptions::new(2., 2.).unwrap();
    let error = owner.set_ui_scale_from_gui(options).unwrap_err();
    assert!(error.contains("options.yaml"));
    let published = owner.gui_command_state().unwrap().ui_scale;
    assert_eq!(published, Some(UiScaleOptions::default()));
}
#[test]
fn missing_scale_owner_returns_a_named_decline_and_disables_the_action() {
    let (mut owner, _) = owner();
    let state = owner.gui_command_state().unwrap();
    let action = oblikovati_gui::GuiShellAction::SetUiScale(UiScaleOptions::default());
    assert!(state.ui_scale.is_none() && !action.enabled(&state));
    assert_scale_owner_decline(&mut owner);
    assert_scale_owner_decline(&mut crate::work::IdleHeadWork);
}
fn assert_scale_owner_decline(owner: &mut dyn HeadWork) {
    let error = owner
        .set_ui_scale_from_gui(UiScaleOptions::default())
        .unwrap_err();
    assert!(error.contains("expected an application preference owner"));
}
