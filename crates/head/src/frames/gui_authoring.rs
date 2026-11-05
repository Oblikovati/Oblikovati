// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWork;
use oblikovati_gui::GuiShellAction;

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::EditThemeColor { name, token, value } => {
            owner.edit_theme_color_from_gui(name, token, value.clone())
        }
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
        GuiShellAction::EditSketch { document, sketch } => {
            owner.edit_existing_sketch_from_gui(*document, *sketch)
        }
        GuiShellAction::StartSketchXY | GuiShellAction::FinishSketch => {
            owner.edit_sketch_from_gui(*action == GuiShellAction::StartSketchXY)
        }
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
        for action in [edit, save] {
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
