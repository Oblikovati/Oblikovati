// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::theme::ThemeLibrary;
#[test]
fn owner_theme_selection_refreshes_gui_and_queued_api_without_a_document() {
    let (owner, awake) = owner();
    let mut owner = owner.with_themes(ThemeLibrary::new(Vec::new(), "Dark").unwrap());
    let before = owner.gui_command_state().unwrap();
    assert!(before.theme.is_some());
    assert!(owner.select_theme_from_gui("Light").unwrap());
    assert!(!owner.select_theme_from_gui("Light").unwrap());
    let active = invoke(&mut owner, &awake, |client| {
        client.theme().active().unwrap()
    });
    assert_eq!(active.name, "Light");
    assert_ne!(before.theme, owner.gui_command_state().unwrap().theme);
    assert!(owner.documents().workspace().documents().is_empty());
    assert!(owner.select_theme_from_gui("missing").is_err());
    assert_eq!(
        Client::new(owner.document_host())
            .theme()
            .active()
            .unwrap()
            .name,
        "Light"
    );
}
#[test]
fn theme_selection_preserves_part_focus_and_undo_history() {
    let (owner, awake) = owner();
    let mut owner = owner.with_themes(ThemeLibrary::new(Vec::new(), "Dark").unwrap());
    let document = create(&mut owner, &awake);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .undo_label()
        .map(str::to_owned);
    owner.select_theme_from_gui("Light").unwrap();
    assert_eq!(
        owner.documents().workspace().active_document(),
        Some(document)
    );
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .undo_label(),
        before.as_deref()
    );
}
#[test]
fn missing_application_library_refuses_theme_selection() {
    let (mut owner, _) = owner();
    assert!(
        owner
            .select_theme_from_gui("Dark")
            .unwrap_err()
            .contains("injected application theme library")
    );
}

struct RecordedThemeStore {
    saved: std::rc::Rc<std::cell::RefCell<Vec<String>>>,
    refuse: bool,
}
impl oblikovati_model::theme::ThemeSelectionStore for RecordedThemeStore {
    fn save_selection(
        &mut self,
        theme: &oblikovati_model::theme::ThemeDefinition,
    ) -> Result<(), String> {
        if self.refuse {
            return Err(format!(
                "theme {:?}: expected a writable selection store",
                theme.name()
            ));
        }
        self.saved.borrow_mut().push(theme.name().into());
        Ok(())
    }
}
#[test]
fn application_theme_selection_publishes_only_after_a_successful_save() {
    for refuse in [false, true] {
        let (owner, _) = owner();
        let saved = std::rc::Rc::new(std::cell::RefCell::new(Vec::new()));
        let store = RecordedThemeStore {
            saved: saved.clone(),
            refuse,
        };
        let mut owner = owner.with_theme_store(
            ThemeLibrary::new(Vec::new(), "Dark").unwrap(),
            Box::new(store),
        );
        let before = owner.gui_command_state().unwrap().theme;
        assert_eq!(owner.select_theme_from_gui("Light").is_err(), refuse);
        let expected = if refuse { "Dark" } else { "Light" };
        assert_eq!(
            Client::new(owner.document_host())
                .theme()
                .active()
                .unwrap()
                .name,
            expected
        );
        assert_eq!(before == owner.gui_command_state().unwrap().theme, refuse);
        assert_eq!(saved.borrow().len(), usize::from(!refuse));
        assert!(!owner.select_theme_from_gui(expected).unwrap());
        assert_eq!(saved.borrow().len(), usize::from(!refuse));
    }
}

#[test]
fn custom_file_declines_reach_the_desktop_without_losing_valid_themes() {
    let (owner, _) = owner();
    let library = ThemeLibrary::new(Vec::new(), "Light").unwrap();
    let decline = oblikovati_persistence::theme_files::ThemeFileDecline {
        path: "config/themes/broken.xml".into(),
        cause: "expected valid XML".into(),
    };
    let mut owner = owner
        .with_themes(library)
        .with_theme_file_declines(vec![decline]);
    let notice = owner.take_command_notice().unwrap();
    assert!(notice.message().contains("broken.xml"));
    assert!(notice.message().contains("expected valid XML"));
    assert_eq!(
        Client::new(owner.document_host())
            .theme()
            .active()
            .unwrap()
            .name,
        "Light"
    );
    let mut owner = owner.with_theme_file_declines(Vec::new());
    assert!(owner.take_command_notice().is_none());
}
