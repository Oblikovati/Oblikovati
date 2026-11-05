// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::color_scheme::ColorSchemeRegistry;
#[test]
fn typed_color_scheme_calls_share_the_registry_without_document_edits() {
    let (mut files, id, mut source) = created_files(false);
    let mut registry = ColorSchemeRegistry::default();
    {
        let client = Client::new(
            FileHost::new(&mut files, &mut source, "test").with_color_schemes(&mut registry),
        );
        assert_eq!(client.color_schemes().active().unwrap().name, "Default");
        let selected = client
            .color_schemes()
            .set_active(&SetColorSchemeArgs {
                name: "High Contrast".into(),
            })
            .unwrap();
        assert_eq!(selected.name, "High Contrast");
        assert_eq!(selected.background_type, ONE_COLOR_BACKGROUND);
        let schemes = client.color_schemes().list().unwrap().schemes.unwrap();
        assert_eq!(schemes.len(), 4);
        assert_eq!(schemes.iter().filter(|scheme| scheme.active).count(), 1);
        assert!(
            schemes
                .iter()
                .all(|scheme| scheme.background_type == ONE_COLOR_BACKGROUND)
        );
    }
    assert_eq!(registry.active().name, "High Contrast");
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
fn missing_owner_bad_json_and_unknown_scheme_are_named_declines() {
    let (mut files, _, mut source) = created_files(false);
    assert!(
        Client::new(FileHost::new(&mut files, &mut source, "test"))
            .color_schemes()
            .active()
            .unwrap_err()
            .to_string()
            .contains("injected application palette registry")
    );
    let mut registry = ColorSchemeRegistry::default();
    let before = registry.clone();
    let host = FileHost::new(&mut files, &mut source, "test").with_color_schemes(&mut registry);
    assert!(host.call(METHOD_COLOR_SCHEMES_SET_ACTIVE, b"{bad").is_err());
    assert!(
        Client::new(host)
            .color_schemes()
            .set_active(&SetColorSchemeArgs {
                name: "missing".into()
            })
            .unwrap_err()
            .to_string()
            .contains("missing")
    );
    assert_eq!(registry, before);
}
