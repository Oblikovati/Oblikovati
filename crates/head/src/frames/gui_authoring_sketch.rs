// SPDX-License-Identifier: GPL-2.0-only
//! Dispatch sketch and feature actions to their owner.
use super::{GuiShellAction, HeadWork, unsupported};

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::SelectBrowserSketch {
            document,
            sketch,
            toggle,
        } => owner.select_browser_sketch_from_gui(*document, *sketch, *toggle),
        GuiShellAction::SelectBrowserBlock {
            document,
            sketch,
            instance,
            toggle,
        } => owner.select_browser_block_from_gui(*document, *sketch, *instance, *toggle),
        GuiShellAction::SelectBrowserFeature { document, feature } => {
            owner.select_browser_feature_from_gui(*document, *feature)
        }
        GuiShellAction::EditFeature { document, edit } => {
            owner.edit_feature_from_gui(*document, edit)
        }
        GuiShellAction::SetSketchLineStyle {
            document,
            sketch,
            style,
        } => owner.set_sketch_line_style_from_gui(*document, *sketch, style),
        GuiShellAction::BrowserChoice { document, choice } => {
            owner.browser_choice_from_gui(*document, choice)
        }
        GuiShellAction::KeepDocumentPicture { document, picture } => {
            owner.keep_document_picture_from_gui(*document, picture.as_ref())
        }
        GuiShellAction::InspectFile(path) => owner.inspect_file_from_gui(path),
        GuiShellAction::UpdateBodyProperties { document, body } => {
            owner.update_body_properties_from_gui(*document, *body)
        }
        GuiShellAction::AssignPartAsset { document, kind, id } => {
            owner.assign_part_asset_from_gui(*document, *kind, id)
        }
        GuiShellAction::EditDocumentAppearance(edit) => {
            owner.edit_document_appearance_from_gui(edit)
        }
        GuiShellAction::OpenClearBodyAppearance { .. } => Ok(false),
        GuiShellAction::ClearSelectedFaceAppearance { document, keys } => {
            owner.clear_selected_face_appearances_from_gui(*document, keys)
        }
        GuiShellAction::ClearSelectedMixedAppearance {
            document,
            body_keys,
            face_keys,
        } => owner.clear_selected_mixed_appearance_from_gui(*document, body_keys, face_keys),
        GuiShellAction::ClearSelectedBodyAppearance { document, keys } => {
            owner.clear_selected_body_appearances_from_gui(*document, keys)
        }
        GuiShellAction::SetBodyAppearance {
            document,
            key,
            appearance,
        } => owner.set_body_appearance_from_gui(*document, key, appearance),
        GuiShellAction::SetSketchVisibility {
            document,
            sketch,
            visible,
        } => owner.set_sketch_visibility_from_gui(*document, *sketch, *visible),
        GuiShellAction::EditSketch { document, sketch } => {
            owner.edit_existing_sketch_from_gui(*document, *sketch)
        }
        // Only the frame's plane display reacts; the sketch starts when a plane is chosen.
        GuiShellAction::ChooseSketchPlane
        | GuiShellAction::ChooseSketchPlaneForExtrude
        | GuiShellAction::ShowHome => Ok(false),
        GuiShellAction::Recent(action) => owner.recent_from_gui(action),
        GuiShellAction::ActivateWindow(id) => owner.document_window_from_gui(*id, false),
        GuiShellAction::CloseDocumentWindow(id) => owner.document_window_from_gui(*id, true),
        GuiShellAction::NewDocumentWindow => owner.new_document_window_from_gui(),
        GuiShellAction::StartSketchOnSelectedFace { document, key } => {
            owner.start_sketch_on_selected_face_from_gui(*document, key)
        }
        GuiShellAction::StartSketchOn(plane) => owner.start_sketch_on_plane_from_gui(*plane),
        GuiShellAction::StartSketchOnWorkPlane(plane) => {
            owner.start_sketch_on_work_plane_from_gui(*plane)
        }
        GuiShellAction::FinishSketch => owner.edit_sketch_from_gui(false),
        GuiShellAction::OpenExtrude => Ok(false),
        GuiShellAction::Loft(action) => owner.loft_from_gui(*action),
        GuiShellAction::Revolve(action) => owner.revolve_from_gui(action.clone()),
        GuiShellAction::Coil(action) => owner.coil_from_gui(action.clone()),
        GuiShellAction::Emboss(action) => owner.emboss_from_gui(action.clone()),
        GuiShellAction::Derive(action) => owner.derive_from_gui(action.clone()),
        GuiShellAction::Rib(action) => owner.rib_from_gui(action.clone()),
        GuiShellAction::Decal(action) => owner.decal_from_gui(action.clone()),
        GuiShellAction::Import(action) => owner.import_from_gui(action.clone()),
        GuiShellAction::Unwrap(action) => owner.unwrap_from_gui(action.clone()),
        GuiShellAction::Sweep(action) => owner.sweep_from_gui(action.clone()),
        GuiShellAction::OpenMoveBody => Ok(false),
        GuiShellAction::PreviewMoveBody(request) => {
            owner.preview_move_body_from_gui(request.as_ref())
        }
        GuiShellAction::CreateMoveBody(request) => owner.commit_move_body_from_gui(request),
        GuiShellAction::OpenShellFeature => Ok(false),
        GuiShellAction::PreviewShellFeature(request) => {
            owner.preview_shell_from_gui(request.as_ref())
        }
        GuiShellAction::CreateShellFeature(request) => owner.commit_shell_from_gui(request),
        GuiShellAction::Hole(action) => owner.hole_from_gui(action.clone()),
        GuiShellAction::Slot(action) => owner.slot_from_gui(action.clone()),
        GuiShellAction::Combine(action) => owner.combine_from_gui(action.clone()),
        GuiShellAction::FilletV2(action) => owner.fillet_v2_from_gui(action.clone()),
        GuiShellAction::ChamferV2(action) => owner.chamfer_v2_from_gui(action.clone()),
        GuiShellAction::OpenFillet => Ok(false),
        GuiShellAction::OpenChamfer => Ok(false),
        GuiShellAction::OpenFilletEdit(_) => Ok(false),
        GuiShellAction::OpenChamferEdit(_) => Ok(false),
        GuiShellAction::PreviewFilletEdit(request) => owner.preview_fillet_edit(request),
        GuiShellAction::EditFilletRadius(request) => owner.commit_fillet_edit(request),
        GuiShellAction::PreviewChamferEdit(request) => owner.preview_chamfer_edit(request),
        GuiShellAction::EditChamferDistance(request) => owner.commit_chamfer_edit(request),
        GuiShellAction::LookAtSelection => Ok(false),
        GuiShellAction::PreviewFillet(request) => owner.preview_fillet_from_gui(request.as_ref()),
        GuiShellAction::CreateFillet(request) => owner.commit_fillet_from_gui(request),
        GuiShellAction::PreviewChamfer(request) => owner.preview_chamfer_from_gui(request.as_ref()),
        GuiShellAction::CreateChamfer(request) => owner.commit_chamfer_from_gui(request),
        GuiShellAction::OpenOffsetPlane => Ok(false),
        GuiShellAction::ValidateOffsetPlane { document, offset } => {
            owner.validate_offset_plane_from_gui(*document, offset)
        }
        GuiShellAction::CreateOffsetPlane {
            document,
            base,
            offset,
        } => owner.create_offset_plane_from_gui(*document, *base, offset),
        GuiShellAction::OpenExtrudeEdit(_) => Ok(false),
        GuiShellAction::PreviewExtrudeEdit(request) => owner.preview_extrusion_edit(request),
        GuiShellAction::EditExtrudeDistance(request) => owner.commit_extrusion_edit(request),
        GuiShellAction::PreviewExtrude(request) => {
            owner.extrusion_preview_from_gui(request.as_ref())
        }
        GuiShellAction::CommitExtrudeDraft(draft, body_name, keep_sketch) => {
            owner.commit_extrusion_draft(draft, body_name.as_deref(), *keep_sketch)
        }
        GuiShellAction::ValidateExtrudeDistance {
            document,
            distance,
            second_distance,
        } => owner.validate_extrusion_distance(*document, distance, second_distance.as_deref()),
        GuiShellAction::Extrude {
            document,
            sketch,
            profile,
            distance,
            direction,
            second_distance,
        } => owner.extrude_from_gui(
            *document,
            *sketch,
            *profile,
            distance,
            *direction,
            second_distance.as_deref(),
        ),
        GuiShellAction::ToggleSketchFormat(format) => owner.toggle_sketch_format_from_gui(*format),
        GuiShellAction::DeleteSketchSelection => owner.delete_sketch_selection_from_gui(),
        GuiShellAction::Dimension(action) => owner.dimension_tool_from_gui(action),
        GuiShellAction::PreviewSketchPattern(request) => {
            owner.preview_sketch_pattern_from_gui(request.clone())
        }
        GuiShellAction::CreateSketchPattern(request) => {
            owner.create_sketch_pattern_from_gui(request.clone())
        }
        GuiShellAction::PreviewSketchPatternEdit { pattern, request } => {
            owner.preview_sketch_pattern_edit_from_gui(*pattern, request.clone())
        }
        GuiShellAction::EditSketchPattern { pattern, request } => {
            owner.edit_sketch_pattern_from_gui(*pattern, request.clone())
        }
        GuiShellAction::SketchMove(action) => owner.sketch_move_from_gui(*action),
        GuiShellAction::SketchText(action) => owner.sketch_text_from_gui(action.clone()),
        GuiShellAction::ConstrainSketch(choice) => owner.constrain_sketch_from_gui(*choice),
        GuiShellAction::PlaceSketchTool(tool) => owner.place_sketch_tool_from_gui(*tool),
        _ => unsupported(action),
    }
}
