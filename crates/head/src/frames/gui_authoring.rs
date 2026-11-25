// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use crate::HeadWork;
use oblikovati_gui::GuiShellAction;
#[path = "gui_authoring_sketch.rs"]
mod sketch;

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::SetSelectionMode(mode) => owner.set_selection_mode_from_gui(*mode),
        GuiShellAction::SetSelectionEnabled { kind, enabled } => {
            owner.set_selection_enabled_from_gui(*kind, *enabled)
        }
        GuiShellAction::MoveSelectionKind { from, to } => {
            owner.move_selection_kind_from_gui(*from, *to)
        }
        GuiShellAction::ResetSelectionFilter => owner.reset_selection_filter_from_gui(),
        GuiShellAction::OpenSelectionFilter | GuiShellAction::OpenHistoryBrowser => Ok(false),
        GuiShellAction::JumpHistoryTo { document, position } => {
            owner.jump_history_from_gui(*document, *position)
        }
        GuiShellAction::SetMarkingMenu(args) => {
            owner.set_marking_menu_from_gui(Some(&args.menu), None)
        }
        GuiShellAction::SetMarkingMenuPreferences(menu, display) => {
            owner.set_marking_menu_from_gui(menu.as_ref(), Some(*display))
        }
        GuiShellAction::SetKeymap(delta) => owner.set_keymap_from_gui(delta),
        GuiShellAction::PlaceSketchBlock(name) => owner.place_sketch_block_from_gui(name),
        GuiShellAction::UpdatePart => owner.update_part_from_gui(),
        GuiShellAction::SelectInferenceGeometry(entities) => {
            owner.select_inference_geometry_from_gui(entities.as_deref())
        }
        GuiShellAction::SetInferenceScope(scope) => owner.set_inference_scope_from_gui(scope),
        GuiShellAction::OpenInferenceScope => Ok(false),
        GuiShellAction::SelectBlockGeometry(active) => {
            owner.select_block_geometry_from_gui(*active)
        }
        GuiShellAction::CreateSketchBlock(name) => owner.create_sketch_block_from_gui(name),
        GuiShellAction::ExplodeSketchBlock {
            document,
            sketch,
            instance,
        } => owner.explode_sketch_block_from_gui(*document, *sketch, *instance),
        GuiShellAction::SetDocumentUnits { document, request } => {
            owner.set_document_units_from_gui(*document, request)
        }
        GuiShellAction::CreateDrawingBaseView(request) => owner.base_view_from_gui(request),
        GuiShellAction::PlaceComponent { assembly, request } => {
            owner.place_component_from_gui(*assembly, request)
        }
        GuiShellAction::PlaceComponentFile {
            assembly,
            path,
            name,
        } => owner.place_component_file_from_gui(*assembly, path, name),
        GuiShellAction::PreviewComponent(request) => owner.preview_component_from_gui(*request),
        GuiShellAction::PreviewComponentFile { assembly, path } => {
            owner.preview_component_file_from_gui(*assembly, path)
        }
        GuiShellAction::PreviewDrawingBaseView(request) => {
            owner.base_view_preview_from_gui(request.as_ref())
        }
        GuiShellAction::SetUiScale(options) => owner.set_ui_scale_from_gui(*options),
        GuiShellAction::SetSketchOptions(options) => {
            owner.set_sketch_options_from_gui(options.clone())
        }
        GuiShellAction::SetSketchIsolation(isolated) => {
            owner.set_sketch_isolation_from_gui(*isolated)
        }
        GuiShellAction::SetPartOptions(options) => owner.set_part_options_from_gui(*options),
        GuiShellAction::LineCommand(command) => owner.line_command_from_gui(*command),
        GuiShellAction::PrecisePoint(point) => owner.precise_point_from_gui(point),
        GuiShellAction::PreciseTriad(command) => owner.precise_triad_from_gui(*command),
        GuiShellAction::DeleteSketchRelation(id) => owner.delete_sketch_relation_from_gui(*id),
        GuiShellAction::SketchCurvature(action) => owner.sketch_curvature_from_gui(*action),
        GuiShellAction::DismissSketchContext { token } => owner.dismiss_sketch_context(*token),
        GuiShellAction::SelectContextConstraintGlyph { token, glyph } => {
            owner.select_context_constraint_glyph(*token, *glyph)
        }
        GuiShellAction::HideContextConstraintGlyph { token, glyph } => {
            owner.hide_context_constraint_glyph(*token, *glyph)
        }
        GuiShellAction::SetContextConstraintGlyphs { token, shown } => {
            owner.set_context_constraint_glyphs(*token, *shown)
        }
        GuiShellAction::ExplodeSketchContext { token, instance } => {
            owner.explode_sketch_context_from_gui(*token, *instance)
        }
        GuiShellAction::SelectSketchContextCandidate { token, entity } => {
            owner.select_sketch_context_candidate(*token, *entity)
        }
        GuiShellAction::ToggleCenterPointFormat => owner.toggle_center_point_format_from_gui(),
        GuiShellAction::SetCenterPoint { point, center } => {
            owner.set_center_point_from_gui(*point, *center)
        }
        GuiShellAction::Inspect(action) => owner.inspect_from_gui(*action),
        GuiShellAction::OpenFaceProperties { document, key } => {
            owner.open_face_properties_from_gui(*document, key)
        }
        GuiShellAction::CloseFaceProperties => owner.close_face_properties_from_gui(),
        GuiShellAction::MeasureSelectedFace { document, key } => {
            owner.measure_selected_face_from_gui(*document, key)
        }
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
        GuiShellAction::VerifyModelStateTable { document, request } => {
            owner.verify_model_state_table_from_gui(*document, request)
        }
        GuiShellAction::SetModelStateTable {
            document,
            request_id,
            request,
        } => owner.set_model_state_table_from_gui(*document, *request_id, request),
        GuiShellAction::SetDocumentProperty(request) => {
            owner.set_document_property_from_gui(request)
        }
        GuiShellAction::Parameter { mutation, request } => {
            owner.edit_parameter_from_gui(*mutation, request)
        }
        GuiShellAction::ActivateDocument(_)
        | GuiShellAction::OpenDocument(_)
        | GuiShellAction::LoadFullDocument(_)
        | GuiShellAction::OpenDeferredDrawing(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::CreateDocument(_)
        | GuiShellAction::CreateProjectTemplate(_)
        | GuiShellAction::NewPart => document(owner, action),
        _ => sketch::apply(owner, action),
    }
}
fn document(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::ActivateDocument(request) => owner.activate_document_from_gui(request),
        GuiShellAction::OpenDocument(request) => owner.open_document_from_gui(request),
        GuiShellAction::LoadFullDocument(id) => owner.load_full_from_gui(*id),
        GuiShellAction::OpenDeferredDrawing(path) => owner.open_deferred_drawing_from_gui(path),
        GuiShellAction::CloseDocument(request) => owner.close_document_from_gui(request),
        GuiShellAction::SaveDocumentAs(request) => owner.complete_save_path(
            request.document,
            Some(std::path::Path::new(&request.new_full_document_name)),
        ),
        GuiShellAction::NewPart => owner.create_part_document(),
        GuiShellAction::CreateDocument(template) => owner.create_document_template(*template),
        GuiShellAction::CreateProjectTemplate(template) => {
            owner.create_project_template_document(template)
        }
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
            frame_selected_edges(window, &edges, *extent)
        }
        ZoomFaceAt { pixel, extent } => {
            let line = window.view_line(*pixel)?;
            let edges = owner.face_edges_on_line(line)?;
            frame_selected_edges(window, &edges, *extent)
        }
    }
}
/// How much of the view a framed face fills, leaving it a border.
const FACE_FILL: f64 = 0.85;
fn frame_selected_edges(
    window: &mut dyn HeadWindow,
    edges: &[[[f64; 3]; 2]],
    extent: [u32; 2],
) -> Result<bool, String> {
    let points: Vec<[f64; 3]> = edges.iter().flatten().copied().collect();
    if points.is_empty() {
        return Ok(false);
    }
    let current = window.view_camera()?;
    let frame = window.orientation_fit_frame()?;
    let fitted = selected_fit_camera(current, &points, extent, frame)?;
    apply_selected_fit(window, current, fitted)
}
fn selected_fit_camera(
    camera: oblikovati_viewport::ViewportCamera,
    points: &[[f64; 3]],
    extent: [u32; 2],
    frame: crate::HeadOrientationFitFrame,
) -> Result<oblikovati_viewport::ViewportCamera, String> {
    if frame.extent != extent {
        return Err(format!(
            "zoom extent {extent:?}: expected current extent {:?}",
            frame.extent
        ));
    }
    let centered = crate::view_orientation::framed(&camera, points, extent, FACE_FILL);
    crate::view_orientation::selection_fit::fit_selected_points(
        &centered,
        points,
        frame.extent,
        frame.area,
        FACE_FILL,
    )
}
fn apply_selected_fit(
    window: &mut dyn HeadWindow,
    current: oblikovati_viewport::ViewportCamera,
    fitted: oblikovati_viewport::ViewportCamera,
) -> Result<bool, String> {
    window.navigate_view(oblikovati_viewport::ViewportNavigation::Fit { margin: 0.1 })?;
    match window.set_view_camera(fitted) {
        Ok(changed) => Ok(changed),
        Err(cause) => match window.previous_view() {
            Ok(_) => Err(cause),
            Err(restore) => Err(format!(
                "zoom selected failed: {cause}; camera restore failed: {restore}; original {current:?}"
            )),
        },
    }
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
    fn extrusion_action_reaches_the_document_owner() {
        let action = GuiShellAction::Extrude {
            document: 7,
            sketch: oblikovati_model::document_sketch::DocumentSketchId(4),
            profile: 0,
            distance: "size".into(),
            second_distance: None,
            direction: Default::default(),
        };
        let cause = apply(&mut crate::work_idle::IdleHeadWork, &action).unwrap_err();
        assert!(
            cause.contains("document 7")
                && cause.contains("size")
                && cause.contains("certified extrusion owner")
        );
    }
    #[test]
    fn idle_sketch_delete_names_the_missing_editor_owner() {
        let mut owner = crate::work_idle::IdleHeadWork;
        let cause = apply(&mut owner, &GuiShellAction::DeleteSketchSelection).unwrap_err();
        assert!(cause.contains("Delete Sketch Selection") && cause.contains("expected"));
    }
    #[test]
    fn idle_context_choice_names_the_missing_editor_owner() {
        let mut owner = crate::work_idle::IdleHeadWork;
        let action = GuiShellAction::SelectSketchContextCandidate {
            token: 9,
            entity: oblikovati_model::sketch::SketchEntityId(12),
        };
        let cause = apply(&mut owner, &action).unwrap_err();
        assert!(cause.contains('9') && cause.contains("12") && cause.contains("expected"));
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

#[cfg(test)]
#[path = "gui_authoring_zoom_tests.rs"]
mod zoom_tests;
