// SPDX-License-Identifier: GPL-2.0-only
//! Keep exclusive part feature drafts consistent with the dock lifecycle.
use super::*;

impl GuiFrameEngine {
    /// Attach Sweep presets to the active desktop preference profile.
    /// Example: `engine.set_sweep_preset_store(store)?`.
    pub fn set_sweep_preset_store(
        &mut self,
        store: Box<dyn crate::GuiSweepPresetStore>,
    ) -> Result<(), String> {
        self.create_dialogs.sweep.set_preset_store(store)
    }

    /// Open a revolution draft for the active part. Example: `engine.request_revolve()`.
    pub fn request_revolve(&mut self) {
        self.create_dialogs.revolve.request_open();
    }
}

type DrawModifyDialogs<'a> = (
    &'a mut crate::blend_dialog::BlendDialog,
    &'a mut crate::hole_dialog::HoleDialog,
    &'a mut crate::slot_dialog::SlotDialog,
    &'a mut crate::combine_dialog::CombineDialog,
    &'a mut crate::draft_dialog::DraftDialog,
    &'a mut crate::delete_face_dialog::DeleteFaceDialog,
    &'a mut crate::thicken_offset_dialog::ThickenOffsetDialog,
    &'a mut crate::direct_edit_dialog::DirectEditDialog,
    &'a mut crate::split_dialog::SplitDialog,
    &'a mut crate::shell_feature_dialog::ShellFeatureDialog,
    &'a mut crate::move_body_dialog::MoveBodyDialog,
    &'a mut crate::offset_plane::OffsetPlaneDialog,
);

pub(super) fn draw_feature_dialogs(
    (ui, commands, area, icons, parameters): (
        &imgui::Ui,
        &GuiCommandState,
        crate::application::document_creation::workspaces::WorkspaceArea,
        &crate::icons::GuiRibbonIcons,
        &mut crate::parameters::GuiParameterEditor,
    ),
    (extrude, plane_selection, popup_was_open, browser_plane): (
        &mut crate::extrude_dialog::ExtrudeDialog,
        bool,
        bool,
        Option<String>,
    ),
    (create, picked_axis): (
        &mut super::create_dialogs::CreateFeatureDialogs,
        Option<oblikovati_model::feature::work_axis::WorkAxisReference>,
    ),
    (
        blend,
        hole,
        slot,
        combine,
        draft,
        delete_face,
        thicken,
        direct_edit,
        split,
        shell,
        motion,
        offset,
    ): DrawModifyDialogs<'_>,
    selected: &mut Vec<GuiShellAction>,
) {
    let hole_plane = browser_plane.clone();
    extrude.panel(
        ui,
        commands,
        plane_selection,
        selected,
        crate::extrude_dialog::ExtrudePanelControls {
            icons,
            parameters,
            popup_was_open,
            browser_plane,
            area,
        },
    );
    create.draw(
        ui,
        commands,
        (area, icons, parameters, picked_axis),
        selected,
    );
    blend.panel(ui, commands, selected, area);
    hole.panel(ui, commands, area, icons, hole_plane.as_deref(), selected);
    slot.panel(ui, commands, area, icons, hole_plane.as_deref(), selected);
    combine.panel(ui, commands, area, icons, selected);
    draft.panel(ui, commands, area, icons, hole_plane.as_deref(), selected);
    delete_face.panel(ui, commands, area, icons, selected);
    thicken.panel(ui, commands, area, icons, selected);
    direct_edit.panel(ui, commands, area, selected);
    split.panel(ui, commands, area, icons, hole_plane.as_deref(), selected);
    shell.panel(ui, commands, area, selected);
    motion.panel(ui, commands, area, selected);
    offset.panel(ui, commands, area, selected);
    let browser_picks = super::work_plane::parse_browser_picks(hole_plane.as_deref(), picked_axis);
    create
        .work_plane
        .sync_and_draw(ui, commands, area, browser_picks, selected);
}

pub(super) fn resolve_menu_extrude(commands: &GuiCommandState, actions: &mut [GuiShellAction]) {
    for action in actions {
        if *action == GuiShellAction::OpenExtrude {
            *action = crate::extrude_dialog::start_action(commands);
        }
    }
}

pub(super) fn finish_frame_capture(
    capture: &mut crate::GuiInputCapture,
    ribbon: &crate::ribbon::DocumentRibbon,
    events: &[crate::GuiInputEvent],
    (popup_open, dock_escape): (bool, bool),
    pointer_flags: (bool, bool, bool),
) {
    let has_text = events
        .iter()
        .any(|event| matches!(event, crate::GuiInputEvent::Text(_)));
    capture.text |= ribbon.tips.consumed || (ribbon.tips.active() && has_text);
    capture.popup = popup_open
        || dock_escape
        || ribbon.tips.consumed
        || ribbon.customization.escape_consumed
        || ribbon.display.temporary_escape_consumed;
    capture.pointer |= pointer_flags.0 || pointer_flags.1 || pointer_flags.2;
}

#[derive(Clone, Copy, PartialEq, Eq)]
enum Draft {
    Extrude,
    Revolve,
    Coil,
    Emboss,
    Derive,
    Rib,
    Decal,
    Import,
    Unwrap,
    Sweep,
    Loft,
    Blend,
    Shell,
    Hole,
    Slot,
    Combine,
    Taper,
    Thread,
    DeleteFace,
    ReplaceFace,
    ThickenOffset,
    DirectEdit,
    Split,
    Mark,
    Finish,
    ExtendSurface,
    BoundaryPatch,
    RuledSurface,
    MoveBody,
}

fn requested_draft(action: &GuiShellAction) -> Option<Draft> {
    Some(match action {
        GuiShellAction::Revolve(
            crate::GuiRevolveAction::Open | crate::GuiRevolveAction::OpenEdit(_),
        ) => Draft::Revolve,
        GuiShellAction::Coil(crate::GuiCoilAction::Open | crate::GuiCoilAction::OpenEdit(_)) => {
            Draft::Coil
        }
        GuiShellAction::Emboss(
            crate::GuiEmbossAction::Open | crate::GuiEmbossAction::OpenEdit(_),
        ) => Draft::Emboss,
        GuiShellAction::Derive(
            crate::GuiDeriveAction::Open | crate::GuiDeriveAction::OpenEdit(_),
        ) => Draft::Derive,
        GuiShellAction::Rib(crate::GuiRibAction::Open | crate::GuiRibAction::OpenEdit(_)) => {
            Draft::Rib
        }
        GuiShellAction::Decal(crate::GuiDecalAction::Open | crate::GuiDecalAction::OpenEdit(_)) => {
            Draft::Decal
        }
        GuiShellAction::Import(
            crate::GuiImportAction::Open | crate::GuiImportAction::OpenEdit(_),
        ) => Draft::Import,
        GuiShellAction::Unwrap(
            crate::GuiUnwrapAction::Open | crate::GuiUnwrapAction::OpenEdit(_),
        ) => Draft::Unwrap,
        GuiShellAction::Sweep(crate::GuiSweepAction::Open | crate::GuiSweepAction::OpenEdit(_)) => {
            Draft::Sweep
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open | crate::GuiLoftAction::OpenEdit(_)) => {
            Draft::Loft
        }
        GuiShellAction::OpenExtrude | GuiShellAction::OpenExtrudeEdit(_) => Draft::Extrude,
        GuiShellAction::OpenFillet
        | GuiShellAction::OpenChamfer
        | GuiShellAction::OpenFilletEdit(_)
        | GuiShellAction::OpenChamferEdit(_) => Draft::Blend,
        GuiShellAction::OpenShellFeature | GuiShellAction::OpenShellEdit(_) => Draft::Shell,
        GuiShellAction::Hole(crate::GuiHoleAction::Open | crate::GuiHoleAction::OpenEdit(_)) => {
            Draft::Hole
        }
        GuiShellAction::Slot(crate::GuiSlotAction::Open | crate::GuiSlotAction::OpenEdit(_)) => {
            Draft::Slot
        }
        GuiShellAction::Combine(
            crate::GuiCombineAction::Open | crate::GuiCombineAction::OpenEdit(_),
        ) => Draft::Combine,
        GuiShellAction::Draft(crate::GuiDraftAction::Open | crate::GuiDraftAction::OpenEdit(_)) => {
            Draft::Taper
        }
        GuiShellAction::Thread(
            crate::GuiThreadAction::Open | crate::GuiThreadAction::OpenEdit(_),
        ) => Draft::Thread,
        GuiShellAction::DeleteFace(
            crate::GuiDeleteFaceAction::Open | crate::GuiDeleteFaceAction::OpenEdit(_),
        ) => Draft::DeleteFace,
        GuiShellAction::ReplaceFace(
            crate::GuiReplaceFaceAction::Open | crate::GuiReplaceFaceAction::OpenEdit(_),
        ) => Draft::ReplaceFace,
        GuiShellAction::ThickenOffset(
            crate::GuiThickenOffsetAction::Open | crate::GuiThickenOffsetAction::OpenEdit(_),
        ) => Draft::ThickenOffset,
        GuiShellAction::DirectEdit(
            crate::GuiDirectEditAction::Open | crate::GuiDirectEditAction::OpenEdit { .. },
        ) => Draft::DirectEdit,
        GuiShellAction::Split(crate::GuiSplitAction::Open | crate::GuiSplitAction::OpenEdit(_)) => {
            Draft::Split
        }
        GuiShellAction::Mark(crate::GuiMarkAction::Open | crate::GuiMarkAction::OpenEdit(_)) => {
            Draft::Mark
        }
        GuiShellAction::Finish(
            crate::GuiFinishAction::Open | crate::GuiFinishAction::OpenEdit(_),
        ) => Draft::Finish,
        GuiShellAction::ExtendSurface(
            crate::GuiExtendSurfaceAction::Open | crate::GuiExtendSurfaceAction::OpenEdit(_),
        ) => Draft::ExtendSurface,
        GuiShellAction::BoundaryPatch(
            crate::GuiBoundaryPatchAction::Open | crate::GuiBoundaryPatchAction::OpenEdit(_),
        ) => Draft::BoundaryPatch,
        GuiShellAction::RuledSurface(
            crate::GuiRuledSurfaceAction::Open | crate::GuiRuledSurfaceAction::OpenEdit(_),
        ) => Draft::RuledSurface,
        GuiShellAction::OpenMoveBody => Draft::MoveBody,
        _ => return None,
    })
}

pub(super) fn open_drafts(
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    (shell, hole, slot, combine, face_draft, delete_face, thicken, direct_edit, split): (
        &mut crate::shell_feature_dialog::ShellFeatureDialog,
        &mut crate::hole_dialog::HoleDialog,
        &mut crate::slot_dialog::SlotDialog,
        &mut crate::combine_dialog::CombineDialog,
        &mut crate::draft_dialog::DraftDialog,
        &mut crate::delete_face_dialog::DeleteFaceDialog,
        &mut crate::thicken_offset_dialog::ThickenOffsetDialog,
        &mut crate::direct_edit_dialog::DirectEditDialog,
        &mut crate::split_dialog::SplitDialog,
    ),
    (motion, blend): (
        &mut crate::move_body_dialog::MoveBodyDialog,
        &mut crate::blend_dialog::BlendDialog,
    ),
    create: &mut super::create_dialogs::CreateFeatureDialogs,
    extrude: &mut crate::extrude_dialog::ExtrudeDialog,
) {
    let requested = actions
        .iter()
        .rev()
        .find(|action| requested_draft(action).is_some());
    let last = requested
        .cloned()
        .or_else(|| extrude.requested.then_some(GuiShellAction::OpenExtrude));
    let Some(action) = last else { return };
    let Some(draft) = requested_draft(&action) else {
        return;
    };
    actions.retain(|action| {
        requested_draft(action).is_none() || *action == GuiShellAction::OpenExtrude
    });
    if draft != Draft::Extrude || matches!(action, GuiShellAction::OpenExtrudeEdit(_)) {
        extrude.close(actions);
    }
    create.close_all(actions);
    if draft != Draft::Shell {
        shell.close(actions);
    }
    if draft != Draft::Hole {
        hole.close(actions);
    }
    if draft != Draft::Slot {
        slot.close(actions);
    }
    if draft != Draft::Combine {
        combine.close(actions);
    }
    if draft != Draft::Taper {
        face_draft.close(actions);
    }
    if draft != Draft::DeleteFace {
        delete_face.close(actions);
    }
    if draft != Draft::ThickenOffset {
        thicken.close(actions);
    }
    if draft != Draft::DirectEdit {
        direct_edit.close(actions);
    }
    if draft != Draft::Split {
        split.close(actions);
    }
    if draft != Draft::MoveBody {
        motion.close(actions);
    }
    if blend.active() {
        blend.close(actions);
    }
    match action {
        GuiShellAction::Revolve(crate::GuiRevolveAction::Open) => create.revolve.open(state, None),
        GuiShellAction::Revolve(crate::GuiRevolveAction::OpenEdit(feature)) => {
            create.revolve.open(state, Some(feature))
        }
        GuiShellAction::Coil(crate::GuiCoilAction::Open) => create.coil.open(state, None),
        GuiShellAction::Coil(crate::GuiCoilAction::OpenEdit(feature)) => {
            create.coil.open(state, Some(feature))
        }
        GuiShellAction::Emboss(crate::GuiEmbossAction::Open) => create.emboss.open(state, None),
        GuiShellAction::Emboss(crate::GuiEmbossAction::OpenEdit(feature)) => {
            create.emboss.open(state, Some(feature))
        }
        GuiShellAction::Derive(crate::GuiDeriveAction::Open) => {
            create.derive.open(state, None, actions)
        }
        GuiShellAction::Derive(crate::GuiDeriveAction::OpenEdit(feature)) => {
            create.derive.open(state, Some(feature), actions)
        }
        GuiShellAction::Rib(crate::GuiRibAction::Open) => create.rib.open(state, None),
        GuiShellAction::Rib(crate::GuiRibAction::OpenEdit(feature)) => {
            create.rib.open(state, Some(feature))
        }
        GuiShellAction::Decal(crate::GuiDecalAction::Open) => create.decal.open(state, None),
        GuiShellAction::Decal(crate::GuiDecalAction::OpenEdit(feature)) => {
            create.decal.open(state, Some(feature))
        }
        GuiShellAction::Import(crate::GuiImportAction::Open) => {
            create.import.open(state, None, actions)
        }
        GuiShellAction::Import(crate::GuiImportAction::OpenEdit(feature)) => {
            create.import.open(state, Some(feature), actions)
        }
        GuiShellAction::Unwrap(crate::GuiUnwrapAction::Open) => create.unwrap.open(state, None),
        GuiShellAction::Unwrap(crate::GuiUnwrapAction::OpenEdit(feature)) => {
            create.unwrap.open(state, Some(feature))
        }
        GuiShellAction::Sweep(crate::GuiSweepAction::Open) => create.sweep.open(state, None),
        GuiShellAction::Sweep(crate::GuiSweepAction::OpenEdit(feature)) => {
            create.sweep.open(state, Some(feature))
        }
        GuiShellAction::Loft(crate::GuiLoftAction::Open) => create.loft.open(state, None, actions),
        GuiShellAction::Loft(crate::GuiLoftAction::OpenEdit(feature)) => {
            create.loft.open(state, Some(feature), actions)
        }
        GuiShellAction::OpenExtrude if state.authoring == crate::GuiAuthoringContext::Sketch => {
            extrude.close(actions);
            extrude.wait_for_sketch(state.document);
        }
        GuiShellAction::OpenExtrude => extrude.requested = true,
        GuiShellAction::OpenExtrudeEdit(feature) => extrude.open_edit(state, feature),
        GuiShellAction::OpenFillet => blend.open(state),
        GuiShellAction::OpenChamfer => blend.open_chamfer(state),
        GuiShellAction::OpenFilletEdit(feature) => blend.open_edit(state, feature),
        GuiShellAction::OpenChamferEdit(feature) => blend.open_chamfer_edit(state, feature),
        GuiShellAction::OpenShellFeature => shell.open(state),
        GuiShellAction::OpenShellEdit(feature) => shell.open_edit(state, feature),
        GuiShellAction::Hole(crate::GuiHoleAction::Open) => hole.open(state, None),
        GuiShellAction::Hole(crate::GuiHoleAction::OpenEdit(feature)) => {
            hole.open(state, Some(feature))
        }
        GuiShellAction::Slot(crate::GuiSlotAction::Open) => slot.open(state, None),
        GuiShellAction::Slot(crate::GuiSlotAction::OpenEdit(feature)) => {
            slot.open(state, Some(feature))
        }
        GuiShellAction::Combine(crate::GuiCombineAction::Open) => combine.open(state, None),
        GuiShellAction::Combine(crate::GuiCombineAction::OpenEdit(feature)) => {
            combine.open(state, Some(feature))
        }
        GuiShellAction::Draft(crate::GuiDraftAction::Open) => face_draft.open(state, None),
        GuiShellAction::Draft(crate::GuiDraftAction::OpenEdit(feature)) => {
            face_draft.open(state, Some(feature))
        }
        GuiShellAction::Thread(crate::GuiThreadAction::Open) => {
            create.thread.open(state, None, actions)
        }
        GuiShellAction::Thread(crate::GuiThreadAction::OpenEdit(feature)) => {
            create.thread.open(state, Some(feature), actions)
        }
        GuiShellAction::DeleteFace(crate::GuiDeleteFaceAction::Open) => {
            delete_face.open(state, None)
        }
        GuiShellAction::DeleteFace(crate::GuiDeleteFaceAction::OpenEdit(feature)) => {
            delete_face.open(state, Some(feature))
        }
        GuiShellAction::ThickenOffset(crate::GuiThickenOffsetAction::Open) => {
            thicken.open(state, None, actions)
        }
        GuiShellAction::ThickenOffset(crate::GuiThickenOffsetAction::OpenEdit(feature)) => {
            thicken.open(state, Some(feature), actions)
        }
        GuiShellAction::DirectEdit(crate::GuiDirectEditAction::Open) => {
            direct_edit.open(state, None, actions)
        }
        GuiShellAction::DirectEdit(crate::GuiDirectEditAction::OpenEdit { feature, operation }) => {
            direct_edit.open(state, Some((feature, operation)), actions)
        }
        GuiShellAction::Split(crate::GuiSplitAction::Open) => split.open(state, None),
        GuiShellAction::Split(crate::GuiSplitAction::OpenEdit(feature)) => {
            split.open(state, Some(feature))
        }
        GuiShellAction::Mark(crate::GuiMarkAction::Open) => create.mark.open(state, None, actions),
        GuiShellAction::Mark(crate::GuiMarkAction::OpenEdit(feature)) => {
            create.mark.open(state, Some(feature), actions)
        }
        GuiShellAction::Finish(crate::GuiFinishAction::Open) => {
            create.finish.open(state, None, actions)
        }
        GuiShellAction::Finish(crate::GuiFinishAction::OpenEdit(feature)) => {
            create.finish.open(state, Some(feature), actions)
        }
        GuiShellAction::ExtendSurface(action) => create.open_extend_surface(state, action, actions),
        GuiShellAction::BoundaryPatch(crate::GuiBoundaryPatchAction::Open) => {
            create.boundary_patch.open(state, None, actions)
        }
        GuiShellAction::BoundaryPatch(crate::GuiBoundaryPatchAction::OpenEdit(feature)) => {
            create.boundary_patch.open(state, Some(feature), actions)
        }
        GuiShellAction::RuledSurface(crate::GuiRuledSurfaceAction::Open) => {
            create.ruled_surface.open(state, None, actions)
        }
        GuiShellAction::RuledSurface(crate::GuiRuledSurfaceAction::OpenEdit(feature)) => {
            create.ruled_surface.open(state, Some(feature), actions)
        }
        GuiShellAction::ReplaceFace(crate::GuiReplaceFaceAction::Open) => {
            create.replace_face.open(state, None, actions)
        }
        GuiShellAction::ReplaceFace(crate::GuiReplaceFaceAction::OpenEdit(feature)) => {
            create.replace_face.open(state, Some(feature), actions)
        }
        GuiShellAction::OpenMoveBody => motion.open(state),
        _ => {}
    }
}
