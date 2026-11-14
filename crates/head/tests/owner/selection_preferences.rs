// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::selection::{AmbientSelectionFilter, SelectionKind};
use std::{cell::RefCell, rc::Rc};

struct FakeSelectionPreferences {
    current: AmbientSelectionFilter,
    writes: Rc<RefCell<Vec<AmbientSelectionFilter>>>,
    refuse: bool,
    mode: oblikovati_gui::GuiSelectionMode,
}
impl crate::SelectionPreferenceAccess for FakeSelectionPreferences {
    fn selection_filter(&self) -> &AmbientSelectionFilter {
        &self.current
    }
    fn selection_mode(&self) -> oblikovati_gui::GuiSelectionMode {
        self.mode
    }
    fn set_selection_filter(
        &mut self,
        filter: &AmbientSelectionFilter,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<(), String> {
        if self.refuse {
            return Err("selection preferences: expected writable file".into());
        }
        self.writes.borrow_mut().push(filter.clone());
        self.current = filter.clone();
        self.mode = mode;
        Ok(())
    }
}
#[test]
fn changes_and_reset_persist_before_publication_without_document_history() {
    let (owner, awake) = owner();
    let writes = Rc::new(RefCell::new(Vec::new()));
    let store = FakeSelectionPreferences {
        current: AmbientSelectionFilter::default(),
        writes: writes.clone(),
        refuse: false,
        mode: oblikovati_gui::GuiSelectionMode::All,
    };
    let mut owner = owner.with_selection_preferences(Box::new(store));
    let document = create(&mut owner, &awake);
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    owner.move_selection_kind_from_gui(7, 0).unwrap();
    owner
        .set_selection_enabled_from_gui(oblikovati_gui::GuiSelectionKind::Body, false)
        .unwrap();
    assert!(owner.reset_selection_filter_from_gui().unwrap());
    assert_eq!(
        writes.borrow().last(),
        Some(&AmbientSelectionFilter::default())
    );
    assert_eq!(owner.ambient_selection, AmbientSelectionFilter::default());
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .generation(),
        generation
    );
}
#[test]
fn failed_save_preserves_the_live_filter_and_selection_mode() {
    let (owner, awake) = owner();
    let store = FakeSelectionPreferences {
        current: AmbientSelectionFilter::default(),
        writes: Default::default(),
        refuse: true,
        mode: oblikovati_gui::GuiSelectionMode::All,
    };
    let mut owner = owner.with_selection_preferences(Box::new(store));
    create(&mut owner, &awake);
    assert!(owner.move_selection_kind_from_gui(7, 0).is_err());
    assert!(
        owner
            .set_selection_enabled_from_gui(oblikovati_gui::GuiSelectionKind::Face, false)
            .is_err()
    );
    assert_eq!(owner.ambient_selection, AmbientSelectionFilter::default());
    assert_eq!(owner.selection_mode, oblikovati_gui::GuiSelectionMode::All);
}
#[test]
fn restored_face_only_filter_restores_the_preset_mode() {
    let (owner, _) = owner();
    let mut saved = AmbientSelectionFilter::default();
    saved.apply_priority(oblikovati_session::selection::SelectionPriority::Face);
    saved.move_kind(saved.rank(SelectionKind::Face), 0);
    let store = FakeSelectionPreferences {
        current: saved.clone(),
        writes: Default::default(),
        refuse: false,
        mode: oblikovati_gui::GuiSelectionMode::Faces,
    };
    let owner = owner.with_selection_preferences(Box::new(store));
    assert_eq!(owner.ambient_selection, saved);
    assert_eq!(
        owner.selection_mode,
        oblikovati_gui::GuiSelectionMode::Faces
    );
}
#[test]
fn failed_reset_preserves_the_saved_filter_and_mode() {
    let (owner, awake) = owner();
    let mut saved = AmbientSelectionFilter::default();
    saved.set_enabled(SelectionKind::Body, false);
    let store = FakeSelectionPreferences {
        current: saved.clone(),
        writes: Default::default(),
        refuse: true,
        mode: oblikovati_gui::GuiSelectionMode::Custom,
    };
    let mut owner = owner.with_selection_preferences(Box::new(store));
    create(&mut owner, &awake);
    assert!(owner.reset_selection_filter_from_gui().is_err());
    assert_eq!(owner.ambient_selection, saved);
    assert_eq!(
        owner.selection_mode,
        oblikovati_gui::GuiSelectionMode::Custom
    );
}
