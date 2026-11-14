// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{contract::ColorScheme as ColorSchemeContract, types::*};
use oblikovati_model::color_scheme::ColorSchemeRegistry;
use serde::Deserialize;
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct GoScheme {
    name: String,
    background_type: BackgroundTypeEnum,
    screen: Color,
    top: Color,
    bottom: Color,
    highlight: Color,
    primary_select: Color,
    #[serde(rename = "SecondSelect")]
    secondary_select: Color,
}
#[test]
fn every_builtin_scheme_property_matches_the_go_oracle_and_public_contract() {
    let expected: Vec<GoScheme> =
        serde_json::from_str(include_str!("color_scheme/go-baseline.json")).unwrap();
    let registry = ColorSchemeRegistry::default();
    assert_eq!(registry.schemes().len(), expected.len());
    for (actual, expected) in registry.schemes().iter().zip(expected) {
        let contract: &dyn ColorSchemeContract = actual;
        assert_eq!(contract.name(), expected.name);
        assert_eq!(actual.background_type, expected.background_type);
        assert_eq!(contract.screen_color(), expected.screen);
        assert_eq!(contract.top_screen_color(), expected.top);
        assert_eq!(contract.bottom_screen_color(), expected.bottom);
        assert_eq!(contract.highlight_color(), expected.highlight);
        assert_eq!(contract.primary_select_color(), expected.primary_select);
        assert_eq!(contract.secondary_select_color(), expected.secondary_select);
    }
}
#[test]
fn selection_restores_authored_background_even_when_the_name_is_unchanged() {
    let mut registry = ColorSchemeRegistry::default();
    assert_eq!(registry.active().name, "Default");
    assert_eq!(registry.background_type(), GRADIENT_BACKGROUND);
    registry.set_active("High Contrast").unwrap();
    assert_eq!(registry.background_type(), ONE_COLOR_BACKGROUND);
    registry.set_background_type(IMAGE_BACKGROUND).unwrap();
    registry.set_active("High Contrast").unwrap();
    assert_eq!(registry.background_type(), ONE_COLOR_BACKGROUND);
    registry.set_active("Sky").unwrap();
    assert_eq!(registry.background_type(), GRADIENT_BACKGROUND);
    assert_eq!(registry.active().name, "Sky");
}
#[test]
fn invalid_names_and_modes_leave_registry_state_unchanged() {
    let mut registry = ColorSchemeRegistry::default();
    let before = registry.clone();
    for name in ["missing", "", "sky"] {
        assert!(registry.set_active(name).unwrap_err().contains(name));
        assert_eq!(registry, before);
    }
    for value in [0, -1, i32::MAX] {
        assert!(
            registry
                .set_background_type(BackgroundTypeEnum(value))
                .unwrap_err()
                .contains(&value.to_string())
        );
        assert_eq!(registry, before);
    }
    for value in [ONE_COLOR_BACKGROUND, GRADIENT_BACKGROUND, IMAGE_BACKGROUND] {
        registry.set_background_type(value).unwrap();
        assert_eq!(registry.background_type(), value);
    }
}
