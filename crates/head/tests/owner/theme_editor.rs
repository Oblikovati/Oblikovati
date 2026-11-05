// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Rgba, TOKEN_CHROME_ACCENT, ThemeToken};
use oblikovati_model::theme::{ThemeDefinition, ThemeLibrary, ThemeSelectionStore};
use std::{cell::RefCell, rc::Rc};
type CustomWrites = Rc<RefCell<Vec<ThemeDefinition>>>;
struct RecordedCustomThemeStore {
    writes: CustomWrites,
    refuse: bool,
}
impl ThemeSelectionStore for RecordedCustomThemeStore {
    fn save_selection(&mut self, _: &ThemeDefinition) -> Result<(), String> {
        Ok(())
    }
    fn can_save_custom(&self, theme: &ThemeDefinition) -> bool {
        theme.name() == "Workshop"
    }
    fn save_custom_theme(&mut self, theme: &ThemeDefinition) -> Result<(), String> {
        if self.refuse {
            return Err(format!("theme {:?}: expected writable XML", theme.name()));
        }
        self.writes.borrow_mut().push(theme.clone());
        Ok(())
    }
}
fn custom_library() -> ThemeLibrary {
    let mut library = ThemeLibrary::new(Vec::new(), "Light").unwrap();
    library.duplicate("Light", "Workshop").unwrap();
    library
}
fn editor_owner(refuse: bool) -> (TestOwner, CustomWrites) {
    let (owner, _) = owner();
    let writes = CustomWrites::default();
    let store = RecordedCustomThemeStore {
        writes: writes.clone(),
        refuse,
    };
    (
        owner.with_theme_store(custom_library(), Box::new(store)),
        writes,
    )
}
fn red() -> Rgba {
    Rgba {
        r: 1.,
        g: 0.,
        b: 0.,
        a: 1.,
    }
}
fn accent(owner: &mut TestOwner) -> String {
    Client::new(owner.document_host())
        .theme()
        .active()
        .unwrap()
        .colors
        .unwrap()["chrome.accent"]
        .clone()
}
#[test]
fn custom_edits_preview_through_api_and_gui_but_only_save_writes_storage() {
    let (mut owner, writes) = editor_owner(false);
    let before = owner.gui_command_state().unwrap().theme;
    assert!(
        owner
            .edit_theme_color_from_gui("Workshop", &TOKEN_CHROME_ACCENT, red())
            .unwrap()
    );
    assert_eq!(accent(&mut owner), "#ff0000ff");
    assert_ne!(before, owner.gui_command_state().unwrap().theme);
    assert!(writes.borrow().is_empty());
    let action = oblikovati_gui::GuiShellAction::SaveTheme {
        name: "Workshop".into(),
    };
    assert!(action.enabled(&owner.gui_command_state().unwrap()));
    assert!(owner.save_theme_from_gui("Workshop").unwrap());
    assert_eq!(writes.borrow().len(), 1);
    assert_eq!(writes.borrow()[0].color(&TOKEN_CHROME_ACCENT), red());
    assert_eq!(
        owner.take_command_notice().unwrap().message(),
        "Saved theme Workshop"
    );
    assert!(owner.documents().workspace().documents().is_empty());
}
#[test]
fn failed_saves_keep_live_preview_and_report_the_storage_failure() {
    let (mut owner, writes) = editor_owner(true);
    owner
        .edit_theme_color_from_gui("Workshop", &TOKEN_CHROME_ACCENT, red())
        .unwrap();
    assert!(
        owner
            .save_theme_from_gui("Workshop")
            .unwrap_err()
            .contains("writable XML")
    );
    assert_eq!(accent(&mut owner), "#ff0000ff");
    assert!(writes.borrow().is_empty());
    assert!(owner.take_command_notice().is_none());
}
#[test]
fn stale_names_builtin_edits_and_invalid_colours_cannot_mutate_the_active_theme() {
    let (mut owner, _) = editor_owner(false);
    let before = owner.gui_command_state().unwrap();
    for name in ["Dark", "Missing"] {
        assert!(
            owner
                .edit_theme_color_from_gui(name, &TOKEN_CHROME_ACCENT, red())
                .is_err()
        );
        assert!(owner.save_theme_from_gui(name).is_err());
    }
    assert!(
        owner
            .edit_theme_color_from_gui("Workshop", &ThemeToken("unknown".into()), red())
            .is_err()
    );
    let invalid = Rgba {
        r: f32::NAN,
        ..red()
    };
    assert!(
        owner
            .edit_theme_color_from_gui("Workshop", &TOKEN_CHROME_ACCENT, invalid)
            .is_err()
    );
    assert_eq!(owner.gui_command_state().unwrap(), before);
    owner.select_theme_from_gui("Dark").unwrap();
    assert!(
        owner
            .edit_theme_color_from_gui("Dark", &TOKEN_CHROME_ACCENT, red())
            .is_err()
    );
    assert!(
        !oblikovati_gui::GuiShellAction::SaveTheme {
            name: "Dark".into()
        }
        .enabled(&owner.gui_command_state().unwrap())
    );
}
#[test]
fn memory_only_customs_preview_but_do_not_claim_writable_source_files() {
    let (owner, _) = owner();
    let mut owner = owner.with_themes(custom_library());
    assert!(
        owner
            .edit_theme_color_from_gui("Workshop", &TOKEN_CHROME_ACCENT, red())
            .unwrap()
    );
    assert!(
        owner
            .save_theme_from_gui("Workshop")
            .unwrap_err()
            .contains("writable custom XML source")
    );
    let (mut empty, _) = super::owner();
    assert!(
        empty
            .edit_theme_color_from_gui("Workshop", &TOKEN_CHROME_ACCENT, red())
            .is_err()
    );
}
