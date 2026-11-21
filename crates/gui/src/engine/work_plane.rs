// SPDX-License-Identifier: GPL-2.0-only
//! Extract the selected work plane from browser state.
pub(super) fn selected_work_plane(
    state: &crate::GuiCommandState,
) -> Option<oblikovati_model::feature::FeatureId> {
    let browser = state.browser.as_ref()?;
    let selected = browser.selected_feature?;
    browser
        .features
        .iter()
        .find(|feature| feature.id == selected && feature.kind == "Work Plane")
        .map(|feature| feature.id)
}
