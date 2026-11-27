// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn feature_edit_action(
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
) -> GuiShellAction {
    if let Some(edit) = loft_edit_action(feature) {
        return edit;
    }
    if browser.sweep_edits.contains_key(&feature.id) {
        return GuiShellAction::Sweep(crate::GuiSweepAction::OpenEdit(feature.id));
    }
    if browser.revolve_edits.contains_key(&feature.id) {
        return GuiShellAction::Revolve(crate::GuiRevolveAction::OpenEdit(feature.id));
    }
    if browser.coil_edits.contains_key(&feature.id) {
        return GuiShellAction::Coil(crate::GuiCoilAction::OpenEdit(feature.id));
    }
    if browser.emboss_edits.contains_key(&feature.id) {
        return GuiShellAction::Emboss(crate::GuiEmbossAction::OpenEdit(feature.id));
    }
    if browser.derive_edits.contains_key(&feature.id) {
        return GuiShellAction::Derive(crate::GuiDeriveAction::OpenEdit(feature.id));
    }
    if browser.rib_edits.contains_key(&feature.id) {
        return GuiShellAction::Rib(crate::GuiRibAction::OpenEdit(feature.id));
    }
    if browser.decal_edits.contains_key(&feature.id) {
        return GuiShellAction::Decal(crate::GuiDecalAction::OpenEdit(feature.id));
    }
    if browser.import_edits.contains_key(&feature.id) {
        return GuiShellAction::Import(crate::GuiImportAction::OpenEdit(feature.id));
    }
    if browser.chamfer_edits.contains_key(&feature.id) {
        return GuiShellAction::OpenChamferEdit(feature.id);
    }
    if browser.fillet_edits.contains_key(&feature.id) {
        return GuiShellAction::OpenFilletEdit(feature.id);
    }
    GuiShellAction::OpenExtrudeEdit(feature.id)
}

pub(super) fn feature_double_click_action(
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
) -> GuiShellAction {
    if browser.derive_edits.contains_key(&feature.id) {
        return GuiShellAction::Derive(crate::GuiDeriveAction::OpenSource(feature.id));
    }
    feature_edit_action(browser, feature)
}

pub(super) fn feature_edit_label(
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
) -> &'static str {
    if browser.derive_edits.contains_key(&feature.id) {
        "Edit Derived Part"
    } else {
        "Edit Feature"
    }
}

pub(super) fn loft_edit_action(feature: &GuiBrowserFeature) -> Option<GuiShellAction> {
    (feature.kind == "Loft").then_some(GuiShellAction::Loft(crate::GuiLoftAction::OpenEdit(
        feature.id,
    )))
}
