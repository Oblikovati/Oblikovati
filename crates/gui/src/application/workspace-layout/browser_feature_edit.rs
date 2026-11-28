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
    if browser.unwrap_edits.contains_key(&feature.id) {
        return GuiShellAction::Unwrap(crate::GuiUnwrapAction::OpenEdit(feature.id));
    }
    if browser.hole_edits.contains_key(&feature.id) {
        return GuiShellAction::Hole(crate::GuiHoleAction::OpenEdit(feature.id));
    }
    if browser.slot_edits.contains_key(&feature.id) {
        return GuiShellAction::Slot(crate::GuiSlotAction::OpenEdit(feature.id));
    }
    if browser.combine_edits.contains_key(&feature.id) {
        return GuiShellAction::Combine(crate::GuiCombineAction::OpenEdit(feature.id));
    }
    if browser.draft_edits.contains_key(&feature.id) {
        return GuiShellAction::Draft(crate::GuiDraftAction::OpenEdit(feature.id));
    }
    if browser.delete_face_edits.contains_key(&feature.id) {
        return GuiShellAction::DeleteFace(crate::GuiDeleteFaceAction::OpenEdit(feature.id));
    }
    if browser.direct_edit_edits.contains_key(&feature.id) {
        return GuiShellAction::DirectEdit(crate::GuiDirectEditAction::OpenEdit {
            feature: feature.id,
            operation: None,
        });
    }
    if browser.split_edits.contains_key(&feature.id) {
        return GuiShellAction::Split(crate::GuiSplitAction::OpenEdit(feature.id));
    }
    if browser.finish_edits.contains_key(&feature.id) {
        return GuiShellAction::Finish(crate::GuiFinishAction::OpenEdit(feature.id));
    }
    if browser.extend_surface_edits.contains_key(&feature.id) {
        return GuiShellAction::ExtendSurface(crate::GuiExtendSurfaceAction::OpenEdit(feature.id));
    }
    if browser.shell_v2_edits.contains_key(&feature.id) {
        return GuiShellAction::OpenShellEdit(feature.id);
    }
    if browser.thread_edits.contains_key(&feature.id) {
        return GuiShellAction::Thread(crate::GuiThreadAction::OpenEdit(feature.id));
    }
    if browser.thicken_offset_edits.contains_key(&feature.id) {
        return GuiShellAction::ThickenOffset(crate::GuiThickenOffsetAction::OpenEdit(feature.id));
    }
    if browser.mark_edits.contains_key(&feature.id) {
        return GuiShellAction::Mark(crate::GuiMarkAction::OpenEdit(feature.id));
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

pub(super) fn datum_visibility_menu_item(
    ui: &imgui::Ui,
    state: &crate::GuiCommandState,
    browser: &GuiModelBrowser,
    feature: &GuiBrowserFeature,
    actions: &mut Vec<GuiShellAction>,
) {
    if !matches!(feature.kind, "Work Plane" | "Work Axis" | "Work Point") {
        return;
    }
    let visible = !browser.hidden_datums.contains(&feature.id);
    if ui.menu_item_config("Visibility").selected(visible).build() {
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: crate::GuiBrowserChoice::DatumVisible {
                feature: feature.id,
                visible: !visible,
            },
        });
    }
}
