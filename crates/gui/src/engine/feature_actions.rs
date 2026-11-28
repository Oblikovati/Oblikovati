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
}

impl crate::GuiFrameEngine {
    /// Open a revolution draft for the active part. Example: `engine.request_revolve()`.
    pub fn request_revolve(&mut self) {
        self.create_dialogs.revolve.request_open();
    }
}

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
    (blend, hole, slot, combine, draft, delete_face, shell, motion, offset): (
        &mut crate::blend_dialog::BlendDialog,
        &mut crate::hole_dialog::HoleDialog,
        &mut crate::slot_dialog::SlotDialog,
        &mut crate::combine_dialog::CombineDialog,
        &mut crate::draft_dialog::DraftDialog,
        &mut crate::delete_face_dialog::DeleteFaceDialog,
        &mut crate::shell_feature_dialog::ShellFeatureDialog,
        &mut crate::move_body_dialog::MoveBodyDialog,
        &mut crate::offset_plane::OffsetPlaneDialog,
    ),
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
    shell.panel(ui, commands, area, selected);
    motion.panel(ui, commands, area, selected);
    offset.panel(ui, commands, area, selected);
}

pub(super) fn sync_offset_plane(
    commands: &GuiCommandState,
    workspace: &mut crate::application::workspace_layout::WorkspaceLayout,
    offset_plane: &mut crate::offset_plane::OffsetPlaneDialog,
    selected: &mut Vec<GuiShellAction>,
) {
    let open_offset = selected.contains(&GuiShellAction::OpenOffsetPlane);
    selected.retain(|action| !matches!(action, GuiShellAction::OpenOffsetPlane));
    if !open_offset && !offset_plane.active() {
        return;
    }
    let base = workspace
        .take_selected_origin_plane()
        .map(crate::GuiOffsetPlaneBase::Origin)
        .or_else(|| selected_work_plane(commands).map(crate::GuiOffsetPlaneBase::WorkPlane))
        .or_else(|| {
            commands
                .model_face_chosen
                .then_some(crate::GuiOffsetPlaneBase::SelectedFace)
        });
    if open_offset && let Some(document) = commands.document {
        offset_plane.open(document, base);
    } else if let Some(base) = base {
        offset_plane.choose(base);
    }
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
    DeleteFace,
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
        GuiShellAction::DeleteFace(
            crate::GuiDeleteFaceAction::Open | crate::GuiDeleteFaceAction::OpenEdit(_),
        ) => Draft::DeleteFace,
        GuiShellAction::OpenMoveBody => Draft::MoveBody,
        _ => return None,
    })
}

pub(super) fn open_drafts(
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    (shell, hole, slot, combine, face_draft, delete_face): (
        &mut crate::shell_feature_dialog::ShellFeatureDialog,
        &mut crate::hole_dialog::HoleDialog,
        &mut crate::slot_dialog::SlotDialog,
        &mut crate::combine_dialog::CombineDialog,
        &mut crate::draft_dialog::DraftDialog,
        &mut crate::delete_face_dialog::DeleteFaceDialog,
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
    if draft != Draft::Revolve {
        create.revolve.close(actions);
    }
    if draft != Draft::Coil {
        create.coil.close(actions);
    }
    if draft != Draft::Emboss {
        create.emboss.close(actions);
    }
    if draft != Draft::Derive {
        create.derive.close(actions);
    }
    if draft != Draft::Rib {
        create.rib.close(actions);
    }
    if draft != Draft::Decal {
        create.decal.close(actions);
    }
    if draft != Draft::Import {
        create.import.close(actions);
    }
    if draft != Draft::Unwrap {
        create.unwrap.close(actions);
    }
    if draft != Draft::Sweep {
        create.sweep.close(actions);
    }
    if draft != Draft::Loft {
        create.loft.close(actions);
    }
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
        GuiShellAction::DeleteFace(crate::GuiDeleteFaceAction::Open) => {
            delete_face.open(state, None)
        }
        GuiShellAction::DeleteFace(crate::GuiDeleteFaceAction::OpenEdit(feature)) => {
            delete_face.open(state, Some(feature))
        }
        GuiShellAction::OpenMoveBody => motion.open(state),
        _ => {}
    }
}
