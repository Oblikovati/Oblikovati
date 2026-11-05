// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::theme::ThemeLibrary;
#[test]
fn file_host_theme_queries_use_the_injected_application_library() {
    let (mut files, id, mut source) = created_files(false);
    let library = ThemeLibrary::new(Vec::new(), "Light").unwrap();
    let client = Client::new(FileHost::new(&mut files, &mut source, "test").with_themes(&library));
    assert_eq!(client.theme().active().unwrap().name, "Light");
    assert_eq!(client.theme().list().unwrap().themes.unwrap().len(), 2);
    assert_eq!(files.workspace().active_document(), Some(id));
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_label()
            .is_none()
    );
}
#[test]
fn missing_theme_owner_is_a_named_decline_without_document_changes() {
    let (mut files, id, mut source) = created_files(false);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let error = client.theme().active().unwrap_err();
    assert!(
        error
            .to_string()
            .contains("injected application theme library")
    );
    assert_eq!(files.workspace().active_document(), Some(id));
    assert!(
        files
            .workspace()
            .session(id)
            .unwrap()
            .undo_label()
            .is_none()
    );
}
