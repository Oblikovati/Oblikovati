// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWork;
use oblikovati_gui::GuiShellAction;

pub(super) fn apply(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
        GuiShellAction::Parameter { mutation, request } => {
            owner.edit_parameter_from_gui(*mutation, request)
        }
        GuiShellAction::OpenDocument(_)
        | GuiShellAction::CloseDocument(_)
        | GuiShellAction::SaveDocumentAs(_)
        | GuiShellAction::NewPart => document(owner, action),
        _ => sketch(owner, action),
    }
}
fn document(owner: &mut dyn HeadWork, action: &GuiShellAction) -> Result<bool, String> {
    match action {
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
        GuiShellAction::Rectangle => owner.place_rectangle_from_gui(),
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
            document(&mut owner, &GuiShellAction::Rectangle)
                .unwrap_err()
                .contains("Rectangle")
        );
    }
}
