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
    if browser.chamfer_edits.contains_key(&feature.id) {
        return GuiShellAction::OpenChamferEdit(feature.id);
    }
    if browser.fillet_edits.contains_key(&feature.id) {
        return GuiShellAction::OpenFilletEdit(feature.id);
    }
    GuiShellAction::OpenExtrudeEdit(feature.id)
}

pub(super) fn loft_edit_action(feature: &GuiBrowserFeature) -> Option<GuiShellAction> {
    (feature.kind == "Loft").then_some(GuiShellAction::Loft(crate::GuiLoftAction::OpenEdit(
        feature.id,
    )))
}
