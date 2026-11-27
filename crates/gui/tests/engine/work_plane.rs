// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::FeatureId;

#[test]
fn support_preselection_accepts_only_the_selected_work_plane() {
    let mut state = crate::GuiCommandState::default();
    assert_eq!(selected_work_plane(&state), None);
    let browser = crate::GuiModelBrowser {
        features: vec![
            crate::GuiBrowserFeature {
                id: FeatureId(1),
                name: "Extrude1".into(),
                suppressed: false,
                extended: None,
                kind: "Extrusion",
            },
            crate::GuiBrowserFeature {
                id: FeatureId(2),
                name: "Work Plane1".into(),
                suppressed: false,
                extended: None,
                kind: "Work Plane",
            },
        ],
        selected_feature: Some(FeatureId(1)),
        ..Default::default()
    };
    state.browser = Some(browser);
    assert_eq!(selected_work_plane(&state), None);
    state.browser.as_mut().unwrap().selected_feature = Some(FeatureId(2));
    assert_eq!(selected_work_plane(&state), Some(FeatureId(2)));
    state.browser.as_mut().unwrap().selected_feature = Some(FeatureId(3));
    assert_eq!(selected_work_plane(&state), None);
}
