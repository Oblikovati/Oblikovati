// SPDX-License-Identifier: GPL-2.0-only
use super::UiScaleOptions;

#[test]
fn defaults_and_independent_factors_match_go_preferences() {
    assert_eq!(
        UiScaleOptions::default(),
        UiScaleOptions::new(1., 1.).unwrap()
    );
    let options = UiScaleOptions::new(1.5, 2.).unwrap();
    assert_eq!(options.font(), 1.5);
    assert_eq!(options.icon(), 2.);
}
#[test]
fn edits_clamp_each_finite_factor_to_its_own_bounds() {
    for (font, icon) in [(-10., -5.), (0., 0.), (0.25, 0.25)] {
        let options = UiScaleOptions::new(font, icon).unwrap();
        assert_eq!((options.font(), options.icon()), (0.5, 0.5));
    }
    let options = UiScaleOptions::new(100., 100.).unwrap();
    assert_eq!((options.font(), options.icon()), (2., 2.5));
}
#[test]
fn absent_or_nonpositive_stored_factors_use_one_before_bounding() {
    for value in [0., -1., -100.] {
        assert_eq!(
            UiScaleOptions::from_stored(value, value).unwrap(),
            UiScaleOptions::default()
        );
    }
    let options = UiScaleOptions::from_stored(0.25, 3.).unwrap();
    assert_eq!((options.font(), options.icon()), (0.5, 2.5));
}
#[test]
fn nonfinite_edits_and_stored_values_are_named_declines() {
    for value in [f32::NAN, f32::INFINITY, f32::NEG_INFINITY] {
        for (font, icon) in [(value, 1.), (1., value)] {
            let error = UiScaleOptions::new(font, icon).unwrap_err();
            assert!(error.contains("font=") && error.contains("icon="));
            assert!(error.contains("expected finite"));
            // Negative infinity is corrupt, not the legacy finite negative-value default.
            assert!(UiScaleOptions::from_stored(font, icon).is_err());
        }
    }
}
