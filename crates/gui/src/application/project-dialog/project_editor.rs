// SPDX-License-Identifier: GPL-2.0-only
//! Stage changes to one project until the owner validates and saves them.
use crate::{GuiCommandState, GuiProject, GuiProjectAction, GuiProjectSettings, GuiShellAction};
use imgui::Ui;
use std::path::PathBuf;

#[derive(Default)]
pub(super) struct ProjectEditor {
    draft: Option<ProjectDraft>,
}

#[derive(Clone, PartialEq, Eq)]
struct ProjectFields {
    name: String,
    settings: GuiProjectSettings,
}

struct ProjectDraft {
    path: PathBuf,
    original: ProjectFields,
    edited: ProjectFields,
    request: Option<u64>,
    error: Option<String>,
}

impl ProjectEditor {
    pub(super) fn request(&mut self, project: &GuiProject) {
        let original = ProjectFields {
            name: project.name.clone(),
            settings: project.settings.clone(),
        };
        self.draft = Some(ProjectDraft {
            path: project.location.clone().into(),
            edited: original.clone(),
            original,
            request: None,
            error: None,
        });
    }

    pub(super) fn is_open(&self) -> bool {
        self.draft.is_some()
    }

    pub(super) fn cancel(&mut self) {
        self.draft = None;
    }

    pub(super) fn finish(&mut self, state: &GuiCommandState) {
        let Some(draft) = &mut self.draft else {
            return;
        };
        let Some((request, result)) = &state.project_file_result else {
            return;
        };
        if draft.request != Some(*request) {
            return;
        }
        match result {
            Ok(_) => self.cancel(),
            Err(error) => {
                draft.request = None;
                draft.error = Some(error.clone());
            }
        }
    }

    pub(super) fn panel(&mut self, ui: &Ui) -> bool {
        let Some(draft) = &mut self.draft else {
            return false;
        };
        ui.text("Project Settings");
        ui.text_wrapped(draft.path.display().to_string());
        let _pending = ui.begin_disabled(draft.request.is_some());
        ui.child_window("##project-fields")
            .size([0., 240.])
            .border(true)
            .build(|| draft.edited.panel(ui));
        if let Some(error) = &draft.error {
            ui.text_wrapped(error);
        }
        crate::widget_abi::button(ui, "Cancel Edit", [0.; 2])
    }

    pub(super) fn save_button(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let ready = self.draft.as_mut().filter(|draft| {
            draft.request.is_none()
                && draft.original != draft.edited
                && !draft.edited.name.trim().is_empty()
        });
        let _disabled = ui.begin_disabled(ready.is_none());
        if crate::widget_abi::button(ui, "Save", [0.; 2])
            && let Some(draft) = ready
        {
            let request = super::next_project_request();
            draft.request = Some(request);
            draft.error = None;
            actions.push(GuiShellAction::ProjectFile(GuiProjectAction::Save {
                request,
                path: draft.path.clone(),
                name: draft.edited.name.clone(),
                settings: draft.edited.settings.clone(),
            }));
        }
    }
}

impl ProjectFields {
    fn panel(&mut self, ui: &Ui) {
        ui.input_text("Project name", &mut self.name).build();
        ui.input_text("Workspace folder", &mut self.settings.workspace)
            .build();
        ui.input_text("Templates folder", &mut self.settings.templates)
            .build();
        ui.input_text("Design data folder", &mut self.settings.design_data)
            .build();
        ui.text("Reference libraries");
        library_rows(ui, &mut self.settings.libraries);
    }
}

fn library_rows(ui: &Ui, libraries: &mut Vec<String>) {
    let mut removed = None;
    for (index, root) in libraries.iter_mut().enumerate() {
        let _identity = ui.push_id_usize(index);
        ui.input_text("Reference folder", root).build();
        ui.same_line();
        if crate::widget_abi::button(ui, "Remove", [0.; 2]) {
            removed = Some(index);
        }
    }
    if let Some(index) = removed {
        libraries.remove(index);
    }
    if crate::widget_abi::button(ui, "Add Library", [0.; 2]) {
        libraries.push(String::new());
    }
}
