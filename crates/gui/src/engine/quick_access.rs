// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn apply_home_requests(
    home: &mut crate::application::home_recent_documents::HomeWorkspace,
    files: &mut crate::files::GuiDocumentFiles,
) -> Option<String> {
    let requests = std::mem::take(&mut home.requests);
    if requests.new_document {
        files.request_new();
    }
    if requests.open_document {
        files.request_open();
    }
    if requests.open_projects {
        files.request_projects();
    }
    requests.file_properties
}
/// The toolbar and ribbon ask; the owners that can answer are the engine's other parts.
pub(super) fn apply_quick_access(
    toolbar: &mut crate::application::quick_access_toolbar::QuickAccessToolbar,
    files: &mut crate::files::GuiDocumentFiles,
    sketch_support: (
        &mut crate::application::quick_access_toolbar::plane_selection::PlaneSelection,
        &mut crate::application::workspace_layout::WorkspaceLayout,
    ),
    extrude: &mut crate::extrude_dialog::ExtrudeDialog,
    browsers: (
        &mut crate::material_browser::MaterialBrowser,
        &mut crate::material_browser::MaterialBrowser,
    ),
    parameters: (
        &mut crate::parameters::GuiParameterEditor,
        &crate::GuiCommandState,
    ),
    selected: &mut Vec<GuiShellAction>,
) {
    let (planes, workspace) = sketch_support;
    let requests = toolbar.take_requests();
    if requests.new_document {
        files.request_new();
    }
    if requests.open_document {
        files.request_open();
    }
    if requests.open_material_browser {
        browsers.0.open();
    }
    if requests.open_appearance_browser {
        browsers.1.open();
    }
    if requests.open_projects {
        files.request_projects();
    }
    if requests.open_parameters {
        parameters.0.open(parameters.1.document);
    }
    let extrude_sketch = requests.choose_plane_for_extrude
        || selected.contains(&GuiShellAction::ChooseSketchPlaneForExtrude);
    if extrude_sketch {
        extrude.wait_for_sketch(parameters.1.document);
    }
    if requests.choose_plane
        || extrude_sketch
        || selected.contains(&GuiShellAction::ChooseSketchPlane)
    {
        if let Some(plane) = workspace.take_selected_origin_plane() {
            planes.active = false;
            selected.push(GuiShellAction::StartSketchOn(plane));
        } else if let Some(plane) = selected_work_plane(parameters.1) {
            planes.active = false;
            selected.push(GuiShellAction::StartSketchOnWorkPlane(plane));
        } else {
            planes.active = true;
        }
    }
    extrude.requested |= requests.open_extrude || selected.contains(&GuiShellAction::OpenExtrude);
}
