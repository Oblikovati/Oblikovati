// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn queued_palette_selection_is_independent_of_chrome_theme_and_document_history() {
    let (owner, awake) = owner();
    let mut owner =
        owner.with_themes(oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Dark").unwrap());
    let palette = invoke(&mut owner, &awake, |client| {
        client
            .color_schemes()
            .set_active(&SetColorSchemeArgs {
                name: "High Contrast".into(),
            })
            .unwrap()
    });
    assert_eq!(palette.name, "High Contrast");
    assert_eq!(
        palette.background_type,
        oblikovati_api::types::ONE_COLOR_BACKGROUND
    );
    assert!(owner.select_theme_from_gui("Light").unwrap());
    let active = Client::new(owner.document_host())
        .color_schemes()
        .active()
        .unwrap();
    assert_eq!(active, palette);
    assert!(owner.documents().workspace().documents().is_empty());
}
