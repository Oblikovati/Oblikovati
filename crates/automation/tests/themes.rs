// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    client::{Caller, Client},
    types,
};
use oblikovati_automation::ThemeQueryHost;
use oblikovati_model::theme::ThemeLibrary;
#[test]
fn canonical_client_reads_complete_active_palette_and_ordered_gallery() {
    let library = ThemeLibrary::new(Vec::new(), "Light").unwrap();
    let client = Client::new(ThemeQueryHost::new(&library));
    let active = client.theme().active().unwrap();
    assert_eq!(active.name, "Light");
    assert_eq!(active.kind, "light");
    assert_eq!(active.colors.unwrap(), library.active().hex_colors());
    let gallery = client.theme().list().unwrap().themes.unwrap();
    assert_eq!(gallery.len(), 2);
    assert_eq!(gallery[0].name, "Dark");
    assert!(!gallery[0].active);
    assert_eq!(gallery[1].name, "Light");
    assert!(gallery[1].active);
}
#[test]
fn custom_edits_and_selection_are_visible_to_new_client_views() {
    let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    library.duplicate("Dark", "Workshop").unwrap();
    library
        .edit_active_color(
            &types::TOKEN_CHROME_TEXT,
            types::Rgba {
                r: 1.,
                g: 0.,
                b: 0.,
                a: 1.,
            },
        )
        .unwrap();
    let client = Client::new(ThemeQueryHost::new(&library));
    let active = client.theme().active().unwrap();
    assert_eq!(active.name, "Workshop");
    assert_eq!(active.kind, "custom");
    assert_eq!(active.colors.unwrap()["chrome.text"], "#ff0000ff");
    assert_eq!(
        client
            .theme()
            .list()
            .unwrap()
            .themes
            .unwrap()
            .iter()
            .filter(|row| row.active)
            .count(),
        1
    );
}
#[test]
fn unknown_theme_calls_are_refused_without_library_changes() {
    let library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
    let host = ThemeQueryHost::new(&library);
    let error = host.call("theme.unknown", b"{}").unwrap_err();
    assert!(error.to_string().contains("theme.unknown"));
    assert_eq!(library.active().name(), "Dark");
    assert_eq!(library.revision(), 1);
}
