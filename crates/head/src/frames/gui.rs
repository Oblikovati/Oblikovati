// SPDX-License-Identifier: GPL-2.0-only
use super::{HeadWindow, HeadWork, ViewportEvent, synchronize_sketch_view};
use oblikovati_gui::GuiShellAction;
use oblikovati_viewport::ViewportInput;
use std::time::Duration;
#[path = "gui_action_key.rs"]
mod action_key;
#[path = "resource_browser.rs"]
mod resource_browser;
use action_key::action_key;
#[path = "gui_authoring.rs"]
mod authoring;
#[path = "gui_input.rs"]
mod input;
#[path = "gui_model_click.rs"]
mod model_click;
use input::{semantic_command_event, update_tool_candidate};
#[path = "gui_dimension_labels.rs"]
mod dimension_labels;
/// GUI work joins the existing event and wake paths; lifecycle events remain owner-visible.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct HeadGuiUpdate {
    pub window_action: Option<oblikovati_gui::GuiWindowAction>,
    pub changed: bool,
    pub consumed_input: bool,
    pub sketch_double_click: bool,
    pub deferred: bool,
    pub actions: Vec<GuiShellAction>,
    pub parameter_actions: Vec<oblikovati_gui::GuiParameterAction>,
    pub constraint_hover: Option<Option<oblikovati_gui::GuiConstraintHover>>,
    pub wake_after: Option<Duration>,
}

fn project_model_pick(window: &dyn HeadWindow, pick: &mut Option<oblikovati_gui::GuiModelPickCue>) {
    if let Some(cue) = pick {
        match window.model_pixel(cue.position) {
            Ok(pixel) => cue.pixel = pixel,
            Err(_) => *pick = None,
        }
    }
}
pub(super) fn advance_gui(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    let (camera_changed, camera_wake) = window.tick_orientation_transition()?;
    let metadata_changed = window.gui_parameters_open() && publish_parameter_view(window, owner)?;
    let mut commands = owner.gui_command_state()?;
    if let Some(cue) = commands.sketch_pick_cue.as_mut() {
        match window.sketch_pixel(cue.plane, cue.position) {
            Ok(pixel) => cue.pixel = pixel,
            Err(_) => commands.sketch_pick_cue = None,
        }
    }
    project_model_pick(window, &mut commands.face_toolbar_anchor);
    project_model_pick(window, &mut commands.model_pick_cue);
    let (labels, projection_resized) = dimension_labels::projected_dimension_labels(window, owner)?;
    commands.dimension_labels = labels;
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
    commands.sketch_axes = dimension_labels::projected_sketch_axes(window, &commands);
    dimension_labels::project_precise_triad(window, owner, &mut commands);
    let mut update = window.gui_frame(event, &commands)?;
    update.changed |= metadata_changed || projection_resized || camera_changed;
    update.wake_after = earliest_wake(update.wake_after, camera_wake);
    if let Err(cause) = dispatch_frame_actions(window, owner, &mut update) {
        if window.has_pending_ribbon_model_pick() {
            window.resolve_ribbon_model_pick(false);
        }
        return Err(cause);
    }
    Ok(update)
}
fn dispatch_frame_actions(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    update: &mut HeadGuiUpdate,
) -> Result<(), String> {
    if let Some(action) = update.window_action {
        update.changed |= window.window_action(action)?;
    }
    for action in &update.parameter_actions {
        let result = owner.apply_gui_parameter(action);
        update.changed |= result.as_ref().copied().unwrap_or(false);
        window.acknowledge_gui_parameter(action, result.map(|_| ()))?;
    }
    for action in &update.actions {
        if let GuiShellAction::SetDocumentUnits { document, request } = action {
            let result = owner.set_document_units_from_gui(*document, request);
            update.changed |= result.as_ref().copied().unwrap_or(false);
            window.acknowledge_gui_document_units(*document, result.map(|_| ()))?;
            update.changed = true;
            continue;
        }
        update.changed |= dispatch_gui_action(window, owner, action)?;
    }
    Ok(())
}

fn dispatch_constraint_settings(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    action: &GuiShellAction,
    document: u64,
    expected: &oblikovati_api::types::SketchSettings,
    settings: &oblikovati_api::types::SketchSettings,
) -> Result<bool, String> {
    // #4172: the owner call and acknowledgement run in one dispatcher turn, without another GUI frame.
    let active = match owner.gui_command_state() {
        Ok(state) => state.document,
        Err(error) => {
            window.acknowledge_gui_constraint_settings(action, Some(document), Err(error))?;
            return Ok(true);
        }
    };
    if active != Some(document) {
        window.acknowledge_gui_constraint_settings(
            action,
            active,
            Err(format!(
                "sketch settings document {document}: expected the active document"
            )),
        )?;
        return Ok(true);
    }
    let result = window
        .validate_gui_constraint_settings(action)
        .and_then(|()| owner.set_document_sketch_settings_from_gui(document, expected, settings));
    window.acknowledge_gui_constraint_settings(action, active, result.map(|_| ()))?;
    Ok(true)
}

fn orient_view(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    command: oblikovati_gui::GuiOrientCommand,
    fit: bool,
    animate: bool,
) -> Result<bool, String> {
    let current = window.view_camera()?;
    let (camera, selection_pivot) = orientation_camera(owner, &current, command)?;
    if animate {
        return animate_orientation(window, camera, fit, selection_pivot);
    }
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
fn animate_orientation(
    window: &mut dyn HeadWindow,
    camera: oblikovati_viewport::ViewportCamera,
    fit: bool,
    selection_pivot: bool,
) -> Result<bool, String> {
    if fit && selection_pivot {
        let frame = window.orientation_fit_frame()?;
        let target = crate::view_orientation::selection_fit::fit_about_selection(
            &camera,
            &[frame.bounds],
            frame.extent,
            frame.area,
        )?;
        return window.start_orientation_transition(target, None);
    }
    window.start_orientation_transition(camera, fit.then_some(FIT_THE_MODEL))
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
        GuiShellAction::SetDocumentSketchSettings {
            document,
            expected,
            settings,
            ..
        } => {
            return dispatch_constraint_settings(
                window, owner, action, *document, expected, settings,
            );
        }
        GuiShellAction::JumpHistoryTo { document, position } => {
            return owner.jump_history_with_window(*document, *position, window);
        }
        GuiShellAction::BrowserChoice {
            document,
            choice:
                choice @ (oblikovati_gui::GuiBrowserChoice::CopyPartView(_)
                | oblikovati_gui::GuiBrowserChoice::ActivatePartView(_)
                | oblikovati_gui::GuiBrowserChoice::SavePartViewCamera(_)
                | oblikovati_gui::GuiBrowserChoice::RestorePartViewCamera(_)),
        } => {
            return owner.part_view_choice_from_gui(*document, choice, window);
        }
        GuiShellAction::CaptureWindowPreview => return window.capture_window_preview(),
        GuiShellAction::CaptureIsometricPreview { document } => {
            super::publish_owner_scene(window, owner)?;
            return window.capture_isometric_preview(*document);
        }
        GuiShellAction::Exit => return window.request_close(),
        GuiShellAction::OpenTeamWebInBrowser(url) => {
            return report_authoring_result(window, resource_browser::open(url).map(|()| true));
        }
        GuiShellAction::OrientView {
            command,
            fit,
            animate,
        } => {
            let result = orient_view(window, owner, *command, *fit, *animate);
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
            return model_click::dispatch(window, owner, *document, *pixel, *toggle);
        }
        GuiShellAction::ContextClickModel { document, pixel } => {
            let line = window.view_line(*pixel)?;
            let depth_window = model_click::depth_window(window)?;
            return owner.context_click_model(*document, line, depth_window);
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
            direction,
            second_distance,
        } = action
    {
        window.reject_extrusion(
            &oblikovati_gui::GuiExtrudePreview {
                generation: None,
                document: *document,
                sketch: *sketch,
                profile: *profile,
                distance: distance.clone(),
                direction: *direction,
                second_distance: second_distance.clone(),
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
    changed |= owner.process_live_window_inputs(&[*event], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    changed |= update_tool_candidate(window, owner, event)?;
    Ok(changed)
}

pub(super) fn dispatch_sketch_double_click(
    window: &dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: ViewportEvent,
) -> Result<bool, String> {
    use oblikovati_viewport::{ViewportButton, ViewportModifiers};
    let ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Left,
        pressed: true,
        position,
        modifiers,
    }) = event
    else {
        return Ok(false);
    };
    if modifiers != ViewportModifiers::default() || window.navigation_active() {
        return Ok(false);
    }
    owner.edit_sketch_text_at(position, window)
}
pub(super) fn earliest_wake(left: Option<Duration>, right: Option<Duration>) -> Option<Duration> {
    left.into_iter().chain(right).min()
}
#[cfg(test)]
#[path = "../../tests/gui_frames.rs"]
mod tests;
