// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWork;
use oblikovati_gui::GuiShellAction;

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::SetUiScale(options) => owner.set_ui_scale_from_gui(*options),
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
        GuiShellAction::SetSketchVisibility {
            document,
            sketch,
            visible,
        } => owner.set_sketch_visibility_from_gui(*document, *sketch, *visible),
        GuiShellAction::EditSketch { document, sketch } => {
            owner.edit_existing_sketch_from_gui(*document, *sketch)
        }
        GuiShellAction::StartSketchXY | GuiShellAction::FinishSketch => {
            owner.edit_sketch_from_gui(*action == GuiShellAction::StartSketchXY)
        }
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

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn idle_sketch_delete_names_the_missing_editor_owner() {
        let mut owner = crate::work::IdleHeadWork;
        let cause = apply(&mut owner, &GuiShellAction::DeleteSketchSelection).unwrap_err();
        assert!(cause.contains("Delete Sketch Selection") && cause.contains("expected"));
    }
    #[test]
    fn idle_theme_editor_requests_return_named_owner_declines() {
        let mut owner = crate::work::IdleHeadWork;
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
        let mut owner = crate::work::IdleHeadWork;
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
        let cause = apply(&mut crate::work::IdleHeadWork, &action).unwrap_err();
        assert!(cause.contains("document 9") && cause.contains("selection owner"));
    }
}
