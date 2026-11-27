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
fn scheme_names_and_modes_keep_the_go_contract_with_updated_studio_colors() {
    let expected: Vec<GoScheme> =
        serde_json::from_str(include_str!("color_scheme/go-baseline.json")).unwrap();
    let registry = ColorSchemeRegistry::default();
    assert_eq!(registry.schemes().len(), expected.len());
    for (actual, expected) in registry.schemes().iter().zip(expected) {
        let contract: &dyn ColorSchemeContract = actual;
        assert_eq!(contract.name(), expected.name);
        assert_eq!(actual.background_type, expected.background_type);
        let channels = |color: Color| [color.r, color.g, color.b];
        let expected_colors = match expected.name.as_str() {
            "Default" => [
                [20, 22, 27],
                [20, 22, 27],
                [30, 33, 42],
                [245, 158, 11],
                [20, 184, 166],
                [45, 212, 191],
            ],
            "Presentation" => [
                [244, 246, 249],
                [244, 246, 249],
                [228, 232, 240],
                [217, 119, 6],
                [13, 148, 136],
                [20, 184, 166],
            ],
            "High Contrast" => [
                [0; 3],
                [0; 3],
                [0; 3],
                [251, 191, 36],
                [45, 212, 191],
                [94, 234, 212],
            ],
            _ => [
                expected.screen,
                expected.top,
                expected.bottom,
                expected.highlight,
                expected.primary_select,
                expected.secondary_select,
            ]
            .map(channels),
        };
        let actual_colors = [
            contract.screen_color(),
            contract.top_screen_color(),
            contract.bottom_screen_color(),
            contract.highlight_color(),
            contract.primary_select_color(),
            contract.secondary_select_color(),
        ];
        for color in &actual_colors {
            assert_eq!(color.source, OVERRIDE_COLOR_SOURCE);
            assert_eq!(color.opacity, 1.);
        }
        assert_eq!(actual_colors.map(channels), expected_colors);
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
