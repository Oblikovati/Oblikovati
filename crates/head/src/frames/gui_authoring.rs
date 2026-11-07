// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use crate::HeadWork;
use oblikovati_gui::GuiShellAction;

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::SetUiScale(options) => owner.set_ui_scale_from_gui(*options),
        GuiShellAction::SetSketchOptions(options) => {
            owner.set_sketch_options_from_gui(options.clone())
        }
        GuiShellAction::LineCommand(command) => owner.line_command_from_gui(*command),
        GuiShellAction::PrecisePoint(point) => owner.precise_point_from_gui(point),
        GuiShellAction::PreciseTriad(command) => owner.precise_triad_from_gui(*command),
        GuiShellAction::ToggleCenterPointFormat => owner.toggle_center_point_format_from_gui(),
        GuiShellAction::SetCenterPoint { point, center } => {
            owner.set_center_point_from_gui(*point, *center)
        }
        GuiShellAction::Inspect(action) => owner.inspect_from_gui(*action),
        GuiShellAction::EditDimensionValue(value) => owner.edit_dimension_value_from_gui(value),
        GuiShellAction::SetPromptFrequency { prompt, frequency } => {
            owner.set_prompt_frequency_from_gui(*prompt, *frequency)
        }
        GuiShellAction::EditThemeColor { name, token, value } => {
            owner.edit_theme_color_from_gui(name, token, value.clone())
        }
        GuiShellAction::DuplicateTheme { base, name } => owner.duplicate_theme_from_gui(base, name),
        GuiShellAction::DeleteTheme { name } => owner.delete_theme_from_gui(name),
        GuiShellAction::SaveTheme { name } => owner.save_theme_from_gui(name),
        GuiShellAction::SetColorScheme(request) => owner.select_color_scheme_from_gui(request),
        GuiShellAction::SetTheme(name) => owner.select_theme_from_gui(name),
        GuiShellAction::Parameter { mutation, request } => {
            owner.edit_parameter_from_gui(*mutation, request)
        }
        GuiShellAction::ActivateDocument(_)
        | GuiShellAction::OpenDocument(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::CreateDocument(_)
        | GuiShellAction::NewPart => document(owner, action),
        _ => sketch(owner, action),
    }
}
fn document(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::ActivateDocument(request) => owner.activate_document_from_gui(request),
        GuiShellAction::OpenDocument(request) => owner.open_document_from_gui(request),
        GuiShellAction::CloseDocument(request) => owner.close_document_from_gui(request),
        GuiShellAction::SaveDocumentAs(request) => owner.complete_save_path(
            request.document,
            Some(std::path::Path::new(&request.new_full_document_name)),
        ),
        GuiShellAction::NewPart => owner.create_part_document(),
        GuiShellAction::CreateDocument(template) => owner.create_document_template(*template),
        _ => unsupported(action),
    }
}
fn sketch(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::SelectBrowserSketch {
            document,
            sketch,
            toggle,
        } => owner.select_browser_sketch_from_gui(*document, *sketch, *toggle),
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
        GuiShellAction::ChooseSketchPlane | GuiShellAction::ShowHome => Ok(false),
        GuiShellAction::Recent(action) => owner.recent_from_gui(action),
        GuiShellAction::ActivateWindow(id) => owner.document_window_from_gui(*id, false),
        GuiShellAction::CloseDocumentWindow(id) => owner.document_window_from_gui(*id, true),
        GuiShellAction::NewDocumentWindow => owner.new_document_window_from_gui(),
        GuiShellAction::StartSketchOn(plane) => owner.start_sketch_on_plane_from_gui(*plane),
        GuiShellAction::FinishSketch => owner.edit_sketch_from_gui(false),
        GuiShellAction::ToggleSketchFormat(format) => owner.toggle_sketch_format_from_gui(*format),
        GuiShellAction::DeleteSketchSelection => owner.delete_sketch_selection_from_gui(),
        GuiShellAction::Dimension(action) => owner.dimension_tool_from_gui(action),
        GuiShellAction::ConstrainSketch(choice) => owner.constrain_sketch_from_gui(*choice),
        GuiShellAction::PlaceSketchTool(tool) => owner.place_sketch_tool_from_gui(*tool),
        _ => unsupported(action),
    }
}
fn unsupported(action: &GuiShellAction) -> Result<bool, String> {
    Err(format!(
        "GUI action {action:?}: expected an owner authoring action"
    ))
}

/// What the navigation bar asks of the view; the owner holds what only the document knows.
pub(super) fn view_command(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    command: &oblikovati_gui::GuiViewCommand,
) -> Result<bool, String> {
    use oblikovati_gui::GuiViewCommand::{
        FitAll, MarkView, Orbit, Pan, Previous, Prompt, RestoreMarkedView, Roll, Zoom, ZoomFaceAt,
        ZoomSelected, ZoomWindow,
    };
    use oblikovati_viewport::ViewportNavigation as Navigation;
    match command {
        Prompt(prompt) => owner.navigation_prompt_from_gui(*prompt),
        Pan(delta) => window.navigate_view(Navigation::Pan(*delta)),
        Zoom(factor) => window.navigate_view(Navigation::Zoom {
            factor: *factor,
            anchor: None,
        }),
        Orbit { yaw, pitch } => window.navigate_view(Navigation::Orbit {
            yaw: *yaw,
            pitch: *pitch,
        }),
        FitAll => window.navigate_view(Navigation::Fit { margin: 0.1 }),
        Previous => window.previous_view(),
        Roll(angle) => {
            let camera = window.view_camera()?;
            window.set_view_camera(crate::view_orientation::rolled(&camera, *angle))
        }
        MarkView => {
            window.mark_view()?;
            Ok(false)
        }
        RestoreMarkedView => window.restore_marked_view(),
        ZoomWindow { corners, extent } => {
            let points = [
                window.view_line(corners[0])?[0],
                window.view_line(corners[1])?[0],
            ];
            frame_view_on(window, &points, *extent, 1.)
        }
        ZoomSelected { extent } => {
            let edges = owner.selected_browser_outline();
            frame_on_edges(window, &edges, *extent)
        }
        ZoomFaceAt { pixel, extent } => {
            let line = window.view_line(*pixel)?;
            let edges = owner.face_edges_on_line(line)?;
            frame_on_edges(window, &edges, *extent)
        }
    }
}
/// How much of the view a framed face fills, leaving it a border.
const FACE_FILL: f64 = 0.85;
fn frame_on_edges(
    window: &mut dyn HeadWindow,
    edges: &[[[f64; 3]; 2]],
    extent: [u32; 2],
) -> Result<bool, String> {
    let points: Vec<[f64; 3]> = edges.iter().flatten().copied().collect();
    frame_view_on(window, &points, extent, FACE_FILL)
}
fn frame_view_on(
    window: &mut dyn HeadWindow,
    points: &[[f64; 3]],
    extent: [u32; 2],
    fill: f64,
) -> Result<bool, String> {
    if points.is_empty() {
        return Ok(false);
    }
    let camera = window.view_camera()?;
    window.set_view_camera(crate::view_orientation::framed(
        &camera, points, extent, fill,
    ))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn idle_sketch_delete_names_the_missing_editor_owner() {
        let mut owner = crate::work_idle::IdleHeadWork;
        let cause = apply(&mut owner, &GuiShellAction::DeleteSketchSelection).unwrap_err();
        assert!(cause.contains("Delete Sketch Selection") && cause.contains("expected"));
    }
    #[test]
    fn idle_theme_editor_requests_return_named_owner_declines() {
        let mut owner = crate::work_idle::IdleHeadWork;
        let color = oblikovati_api::types::Rgba {
            r: 1.,
            g: 0.,
            b: 0.,
            a: 1.,
        };
        let edit = GuiShellAction::EditThemeColor {
            name: "Workshop".into(),
            token: oblikovati_api::types::TOKEN_CHROME_ACCENT,
            value: color,
        };
        let save = GuiShellAction::SaveTheme {
            name: "Workshop".into(),
        };
        let duplicate = GuiShellAction::DuplicateTheme {
            base: "Dark".into(),
            name: "Workshop".into(),
        };
        let delete = GuiShellAction::DeleteTheme {
            name: "Workshop".into(),
        };
        for action in [edit, save, duplicate, delete] {
            let cause = apply(&mut owner, &action).unwrap_err();
            assert!(cause.contains("Workshop") && cause.contains("expected"));
        }
    }
    #[test]
    fn semantic_keys_and_wrong_authoring_groups_are_refused_without_owner_work() {
        let mut owner = crate::work_idle::IdleHeadWork;
        for action in [
            GuiShellAction::Save,
            GuiShellAction::Undo,
            GuiShellAction::Redo,
            GuiShellAction::Home,
        ] {
            assert!(
                apply(&mut owner, &action)
                    .unwrap_err()
                    .contains("expected an owner authoring action")
            );
        }
        assert!(
            document(
                &mut owner,
                &GuiShellAction::PlaceSketchTool(oblikovati_gui::GuiSketchTool::Rectangle)
            )
            .unwrap_err()
            .contains("Rectangle")
        );
    }
}

#[cfg(test)]
mod browser_selection_tests {
    use super::*;
    #[test]
    fn selection_requires_an_application_owner() {
        let action = GuiShellAction::SelectBrowserSketch {
            document: 9,
            sketch: oblikovati_model::document_sketch::DocumentSketchId(3),
            toggle: false,
        };
        let cause = apply(&mut crate::work_idle::IdleHeadWork, &action).unwrap_err();
        assert!(cause.contains("document 9") && cause.contains("selection owner"));
    }
}
