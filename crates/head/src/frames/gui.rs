// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, synchronize_sketch_view};
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::{ViewportInput, ViewportKey};
use std::time::Duration;
#[path = "gui_authoring.rs"]
mod authoring;
#[path = "gui_dimension_labels.rs"]
mod dimension_labels;
/// GUI work joins the existing event and wake paths; lifecycle events remain owner-visible.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadGuiUpdate {
    pub changed: bool,
    pub consumed_input: bool,
    pub deferred: bool,
    pub actions: Vec<GuiShellAction>,
    pub parameter_actions: Vec<oblikovati_gui::GuiParameterAction>,
    pub constraint_hover: Option<Option<oblikovati_gui::GuiConstraintHover>>,
    pub wake_after: Option<Duration>,
}
pub(super) fn advance_gui(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    let metadata_changed = window.gui_parameters_open() && publish_parameter_view(window, owner)?;
    let mut commands = owner.gui_command_state()?;
    if let Some(cue) = commands.sketch_pick_cue.as_mut() {
        match window.sketch_pixel(cue.plane, cue.position) {
            Ok(pixel) => cue.pixel = pixel,
            Err(_) => commands.sketch_pick_cue = None,
        }
    }
    commands.dimension_labels = dimension_labels::projected_dimension_labels(window, owner)?;
    commands.origin_planes = dimension_labels::origin_plane_outlines(window, &commands);
    if commands.selected_edge_count > 0 {
        commands.selected_edge_outlines = dimension_labels::projected_selected_edges(window, owner);
        commands.feature_edges = commands
            .selected_edge_outlines
            .iter()
            .flatten()
            .copied()
            .collect();
    } else {
        commands.feature_edges = dimension_labels::projected_browser_outline(window, owner);
    }
    commands.origin_axes = dimension_labels::origin_axis_lines(window, &commands);
    follow_view_projection(window);
    commands.view_axes = window
        .view_camera()
        .ok()
        .map(|camera| crate::view_orientation::view_axes(&camera));
    commands.constraint_glyphs = dimension_labels::projected_constraint_glyphs(window, owner);
    dimension_labels::project_precise_triad(window, owner, &mut commands);
    let mut update = window.gui_frame(event, &commands)?;
    update.changed |= metadata_changed;
    for action in &update.parameter_actions {
        let result = owner.apply_gui_parameter(action);
        update.changed |= result.as_ref().copied().unwrap_or(false);
        window.acknowledge_gui_parameter(action, result.map(|_| ()))?;
    }
    for action in &update.actions {
        update.changed |= dispatch_gui_action(window, owner, action)?;
    }
    Ok(update)
}
fn orient_view(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    command: oblikovati_gui::GuiOrientCommand,
    fit: bool,
) -> Result<bool, String> {
    let current = window.view_camera()?;
    let (camera, selection_pivot) = orientation_camera(owner, &current, command)?;
    if fit && selection_pivot {
        return fit_selection_view(window, current, camera);
    }
    let changed = window.set_view_camera(camera)?;
    if fit {
        window.navigate_view(FIT_THE_MODEL)
    } else {
        Ok(changed)
    }
}
fn orientation_camera(
    owner: &mut dyn HeadWork,
    current: &oblikovati_viewport::ViewportCamera,
    command: oblikovati_gui::GuiOrientCommand,
) -> Result<(oblikovati_viewport::ViewportCamera, bool), String> {
    let locked = owner.gui_command_state()?.orientation_selection_locked;
    if !locked || matches!(command, oblikovati_gui::GuiOrientCommand::Roll { .. }) {
        return Ok((crate::view_orientation::oriented(current, command), false));
    }
    let pivot = owner
        .orientation_selection_pivot()?
        .ok_or("orientation selection lock: expected a current selected model face or body")?;
    Ok((
        crate::view_orientation::oriented_about(current, command, pivot)?,
        true,
    ))
}
fn fit_selection_view(
    window: &mut dyn HeadWindow,
    current: oblikovati_viewport::ViewportCamera,
    camera: oblikovati_viewport::ViewportCamera,
) -> Result<bool, String> {
    let frame = window.orientation_fit_frame()?;
    let fitted = crate::view_orientation::selection_fit::fit_about_selection(
        &camera,
        &[frame.bounds],
        frame.extent,
        frame.area,
    )?;
    window.set_view_camera(camera)?;
    let applied = apply_selection_fit(window, fitted);
    match applied {
        Ok(changed) => Ok(changed),
        Err(cause) => match window.set_view_camera(current) {
            Ok(_) => Err(cause),
            Err(restore) => Err(format!(
                "selection fit failed: {cause}; restoring the camera failed: {restore}"
            )),
        },
    }
}
fn apply_selection_fit(
    window: &mut dyn HeadWindow,
    fitted: oblikovati_viewport::ViewportCamera,
) -> Result<bool, String> {
    window.navigate_view(FIT_THE_MODEL)?;
    window.set_view_camera(fitted)
}
fn publish_parameter_view(
    window: &mut dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Result<bool, String> {
    match owner.gui_parameter_snapshot() {
        Ok(snapshot) => window
            .set_gui_parameter_snapshot(snapshot)
            .or_else(|issue| window.show_gui_parameter_issue(issue)),
        Err(issue) => window.show_gui_parameter_issue(issue),
    }
}
pub(super) fn dispatch_shortcut_action(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    action: &GuiShellAction,
) -> Result<bool, String> {
    if *action != GuiShellAction::OpenExtrude {
        return dispatch_gui_action(window, owner, action);
    }
    if !action.dispatchable(&owner.gui_command_state()?) {
        return Ok(false);
    }
    window.request_extrusion();
    Ok(true)
}
pub(super) fn dispatch_gui_action(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    action: &GuiShellAction,
) -> Result<bool, String> {
    // A GUI action can change tools or geometry without a viewport pointer event (#4043).
    if !matches!(action, GuiShellAction::KeepDocumentPicture { .. }) {
        window.set_tool_feedback(None)?;
    }
    match action {
        GuiShellAction::CaptureWindowPreview => return window.capture_window_preview(),
        GuiShellAction::CaptureIsometricPreview { document } => {
            super::publish_owner_scene(window, owner)?;
            return window.capture_isometric_preview(*document);
        }
        GuiShellAction::Exit => return window.request_close(),
        GuiShellAction::OrientView { command, fit } => {
            let result = orient_view(window, owner, *command, *fit);
            return report_authoring_result(window, result);
        }
        GuiShellAction::SetOrientationSelectionLock(locked) => {
            let result = owner.set_orientation_lock_from_gui(*locked);
            return report_authoring_result(window, result);
        }
        GuiShellAction::SetViewProjection(mode) => {
            window.set_view_projection(*mode);
            return Ok(true);
        }
        GuiShellAction::LookAtSelection => {
            let plane = owner
                .look_at_selection_plane()?
                .ok_or("look at: expected a selected planar reference")?;
            return window
                .set_view_camera_recorded(window.view_camera()?.facing_planar_reference(plane)?);
        }
        GuiShellAction::ViewCommand(command) => {
            return authoring::view_command(window, owner, command);
        }
        GuiShellAction::ClickModel {
            document,
            pixel,
            toggle,
        } => {
            let line = window.view_line(*pixel)?;
            let depth_window = match window.window_capture_extent()? {
                Some([_, height]) => Some(crate::part_scene::pick_candidates::depth_window(
                    window.view_camera()?.scale,
                    height,
                )?),
                None => None,
            };
            return owner.click_model_with_modifiers(*document, line, depth_window, *toggle);
        }
        GuiShellAction::StartSketchAt {
            document,
            pixel,
            origin_plane,
        } => {
            let result = owner
                .start_sketch_on_face_from_gui(*document, window.view_line(*pixel)?)
                .and_then(|found| match (found, origin_plane) {
                    (false, Some(plane)) => owner.start_sketch_on_plane_from_gui(*plane),
                    _ => Ok(found),
                });
            let changed = report_authoring_result(window, result)?;
            return Ok(synchronize_sketch_view(window, owner)? || changed);
        }
        _ => {}
    }
    if let Some(key) = action_key(action) {
        let event = ViewportEvent::Input(ViewportInput::Key { key, pressed: true });
        return dispatch_frame_input(window, owner, &event);
    }
    let result = authoring::apply(owner, action);
    if let Err(cause) = &result
        && let GuiShellAction::CreateOffsetPlane {
            document,
            base,
            offset,
        } = action
    {
        window.reject_offset_plane(*document, *base, offset, cause);
    }
    if let Err(cause) = &result
        && let GuiShellAction::EditDimensionValue(value) = action
    {
        window.reject_dimension_edit(value, cause);
    }
    if let Err(cause) = &result
        && let GuiShellAction::EditExtrudeDistance(request) = action
    {
        window.reject_extrusion_edit(request, cause);
    }
    if let Err(cause) = &result
        && let GuiShellAction::Extrude {
            document,
            sketch,
            profile,
            distance,
        } = action
    {
        window.reject_extrusion(
            &oblikovati_gui::GuiExtrudePreview {
                generation: None,
                document: *document,
                sketch: *sketch,
                profile: *profile,
                distance: distance.clone(),
            },
            cause,
        );
    }
    if matches!(action, GuiShellAction::PreviewFilletEdit(_))
        && result.as_ref().is_ok_and(|changed| *changed)
    {
        window.show_command_notice(crate::DesktopCommandNotice::Ready)?;
    }
    let changed = report_authoring_result(window, result)?;
    Ok(synchronize_sketch_view(window, owner)? || changed)
}
fn report_authoring_result(
    window: &mut dyn HeadWindow,
    result: Result<bool, String>,
) -> Result<bool, String> {
    match result {
        Ok(changed) => Ok(changed),
        Err(cause) => {
            window.show_command_notice(crate::DesktopCommandNotice::AuthoringFailed { cause })?;
            Ok(true)
        }
    }
}
fn action_key(action: &GuiShellAction) -> Option<ViewportKey> {
    Some(match action {
        GuiShellAction::SelectBlockGeometry(_)
        | GuiShellAction::CreateSketchBlock(_)
        | GuiShellAction::PlaceSketchBlock(_) => {
            return None;
        }
        GuiShellAction::SetMarkingMenu(_)
        | GuiShellAction::SetKeymap(_)
        | GuiShellAction::SetUiScale(_)
        | GuiShellAction::SetDocumentUnits { .. }
        | GuiShellAction::CreateDrawingBaseView(_)
        | GuiShellAction::PlaceComponent { .. }
        | GuiShellAction::PlaceComponentFile { .. }
        | GuiShellAction::PreviewComponent(_)
        | GuiShellAction::PreviewComponentFile { .. }
        | GuiShellAction::PreviewDrawingBaseView(_)
        | GuiShellAction::SetSketchOptions(_)
        | GuiShellAction::SetPartOptions(_)
        | GuiShellAction::LineCommand(_)
        | GuiShellAction::PrecisePoint(_)
        | GuiShellAction::PreciseTriad(_)
        | GuiShellAction::ToggleCenterPointFormat
        | GuiShellAction::SetCenterPoint { .. }
        | GuiShellAction::DeleteSketchRelation(_)
        | GuiShellAction::SelectSketchContextCandidate { .. }
        | GuiShellAction::DismissSketchContext { .. }
        | GuiShellAction::Inspect(_)
        | GuiShellAction::EditDimensionValue(_)
        | GuiShellAction::SetPromptFrequency { .. }
        | GuiShellAction::CaptureWindowPreview
        | GuiShellAction::CaptureIsometricPreview { .. }
        | GuiShellAction::Exit
        | GuiShellAction::EditThemeColor { .. }
        | GuiShellAction::DuplicateTheme { .. }
        | GuiShellAction::DeleteTheme { .. }
        | GuiShellAction::SaveTheme { .. }
        | GuiShellAction::SetColorScheme(_)
        | GuiShellAction::SetTheme(_)
        | GuiShellAction::Parameter { .. }
        | GuiShellAction::SetDocumentProperty(_)
        | GuiShellAction::OpenIProperties
        | GuiShellAction::UpdatePart
        | GuiShellAction::SetSelectionMode(_)
        | GuiShellAction::OpenSelectionFilter
        | GuiShellAction::OpenHistoryBrowser
        | GuiShellAction::JumpHistoryTo { .. }
        | GuiShellAction::SetSelectionEnabled { .. }
        | GuiShellAction::MoveSelectionKind { .. }
        | GuiShellAction::ResetSelectionFilter
        | GuiShellAction::ActivateDocument(_)
        | GuiShellAction::OpenDocument(_)
        | GuiShellAction::LoadFullDocument(_)
        | GuiShellAction::OpenDeferredDrawing(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::SetSketchVisibility { .. }
        | GuiShellAction::SelectBrowserSketch { .. }
        | GuiShellAction::SelectBrowserBlock { .. }
        | GuiShellAction::SelectBrowserFeature { .. }
        | GuiShellAction::BrowserChoice { .. }
        | GuiShellAction::SetSketchLineStyle { .. }
        | GuiShellAction::EditFeature { .. }
        | GuiShellAction::OpenExtrudeEdit(_)
        | GuiShellAction::PreviewExtrudeEdit(_)
        | GuiShellAction::EditExtrudeDistance(_)
        | GuiShellAction::ClickModel { .. }
        | GuiShellAction::InspectFile(_)
        | GuiShellAction::KeepDocumentPicture { .. }
        | GuiShellAction::OrientView { .. }
        | GuiShellAction::SetViewProjection(_)
        | GuiShellAction::SetOrientationSelectionLock(_)
        | GuiShellAction::ViewCommand(_)
        | GuiShellAction::UpdateBodyProperties { .. }
        | GuiShellAction::AssignPartAsset { .. }
        | GuiShellAction::SetBodyAppearance { .. }
        | GuiShellAction::ClearSelectedFaceAppearance { .. }
        | GuiShellAction::ClearSelectedMixedAppearance { .. }
        | GuiShellAction::ClearSelectedBodyAppearance { .. }
        | GuiShellAction::OpenClearBodyAppearance { .. }
        | GuiShellAction::EditSketch { .. }
        | GuiShellAction::Dimension(_)
        | GuiShellAction::PreviewSketchPattern(_)
        | GuiShellAction::CreateSketchPattern(_)
        | GuiShellAction::OpenSketchPattern { .. }
        | GuiShellAction::PreviewSketchPatternEdit { .. }
        | GuiShellAction::EditSketchPattern { .. }
        | GuiShellAction::SketchMove(_)
        | GuiShellAction::ConstrainSketch(_)
        | GuiShellAction::PlaceSketchTool(_)
        | GuiShellAction::ToggleSketchFormat(_)
        | GuiShellAction::DeleteSketchSelection
        | GuiShellAction::CreateDocument(_)
        | GuiShellAction::NewPart
        | GuiShellAction::ChooseSketchPlane
        | GuiShellAction::ChooseSketchPlaneForExtrude
        | GuiShellAction::ShowHome
        | GuiShellAction::Recent(_)
        | GuiShellAction::ActivateWindow(_)
        | GuiShellAction::NewDocumentWindow
        | GuiShellAction::CloseDocumentWindow(_)
        | GuiShellAction::StartSketchOn(_)
        | GuiShellAction::StartSketchOnWorkPlane(_)
        | GuiShellAction::StartSketchAt { .. }
        | GuiShellAction::FinishSketch
        | GuiShellAction::OpenExtrude
        | GuiShellAction::OpenFillet
        | GuiShellAction::OpenFilletEdit(_)
        | GuiShellAction::PreviewFilletEdit(_)
        | GuiShellAction::EditFilletRadius(_)
        | GuiShellAction::LookAtSelection
        | GuiShellAction::PreviewFillet(_)
        | GuiShellAction::CreateFillet(_)
        | GuiShellAction::OpenOffsetPlane
        | GuiShellAction::ValidateOffsetPlane { .. }
        | GuiShellAction::CreateOffsetPlane { .. }
        | GuiShellAction::ValidateExtrudeDistance { .. }
        | GuiShellAction::PreviewExtrude(_)
        | GuiShellAction::Extrude { .. } => {
            return None;
        }
        GuiShellAction::Save => ViewportKey::Save,
        GuiShellAction::Undo => ViewportKey::Undo,
        GuiShellAction::Redo => ViewportKey::Redo,
        GuiShellAction::Home => ViewportKey::Home,
    })
}
/// The whole model in view, with the margin Home leaves.
const FIT_THE_MODEL: oblikovati_viewport::ViewportNavigation =
    oblikovati_viewport::ViewportNavigation::Fit { margin: 0.1 };
/// Keep the camera on the chosen projection as the viewpoint changes, by whatever means.
fn follow_view_projection(window: &mut dyn HeadWindow) {
    let Ok(camera) = window.view_camera() else {
        return;
    };
    let mode = window.view_projection();
    let perspective = crate::view_orientation::projects_in_perspective(&camera, mode);
    if perspective != camera.perspective {
        // A window that cannot take the camera keeps its view; nothing here depends on it.
        let _ = window.set_view_camera(oblikovati_viewport::ViewportCamera {
            perspective,
            ..camera
        });
    }
}
pub(super) fn dispatch_frame_input(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: &ViewportEvent,
) -> Result<bool, String> {
    let event = semantic_command_event(*event);
    let event = &event;
    let mut changed = window.process_view_input(event)?;
    changed |= owner.process_window_inputs(&[*event], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    changed |= update_tool_candidate(window, owner, event)?;
    Ok(changed)
}
fn update_tool_candidate(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: &ViewportEvent,
) -> Result<bool, String> {
    let ViewportEvent::Input(ViewportInput::Motion { position, .. }) = event else {
        return window.set_tool_feedback(None);
    };
    if window.navigation_active() {
        return window.set_tool_feedback(None);
    }
    let candidate = match owner.sketch_tool_feedback(*position, window) {
        Ok(candidate) => candidate,
        Err(cause) => {
            window.show_command_notice(crate::DesktopCommandNotice::SelectionFailed { cause })?;
            window.set_tool_feedback(None)?;
            return Ok(true);
        }
    };
    window.set_tool_feedback(candidate.as_ref())
}

fn semantic_command_event(event: ViewportEvent) -> ViewportEvent {
    let ViewportEvent::Input(ViewportInput::PhysicalKey {
        command: Some(key),
        pressed,
        ..
    }) = event
    else {
        return event;
    };
    ViewportEvent::Input(ViewportInput::Key { key, pressed })
}
pub(super) fn earliest_wake(left: Option<Duration>, right: Option<Duration>) -> Option<Duration> {
    left.into_iter().chain(right).min()
}
#[cfg(test)]
#[path = "../../tests/gui_frames.rs"]
mod tests;
