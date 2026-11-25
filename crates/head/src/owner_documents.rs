// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{ActivateDocumentArgs, DocumentInfo},
};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Copy canonical file metadata; example: `owner.gui_document_rows()?` for one GUI snapshot.
    pub(super) fn gui_document_rows(&self) -> Result<Vec<DocumentInfo>, String> {
        self.files
            .workspace()
            .documents()
            .iter()
            .copied()
            .map(|id| {
                self.files
                    .document_info(id)
                    .map_err(|cause| cause.to_string())
            })
            .collect()
    }
    /// Name a new document so two open tabs never read the same; example: `Part1`, then `Part2`.
    /// Each count only grows in a session: closing `Part2` does not reuse its name.
    pub(super) fn next_new_document_name(&mut self, prefix: &'static str) -> String {
        let open = self.gui_document_rows().unwrap_or_default();
        let count = self.new_document_counts.entry(prefix).or_default();
        loop {
            *count += 1;
            let name = format!("{prefix}{count}");
            if open.iter().all(|row| row.name != name) {
                return name;
            }
        }
    }
    /// Report loaded projects and their settings; example: for the Projects dialog.
    pub(super) fn gui_projects(&self) -> Vec<oblikovati_gui::GuiProject> {
        let active = self.files.active_project().map(|project| &project.path);
        self.files
            .projects()
            .iter()
            .map(|project| oblikovati_gui::GuiProject {
                name: project.configuration.name.clone(),
                location: project.path.display().to_string(),
                active: active == Some(&project.path),
                settings: project_settings(&project.configuration),
                groups: project_groups(&project.configuration),
            })
            .collect()
    }
    /// Register a validated project. Example: `owner.load_desktop_project(path)?`.
    pub(crate) fn load_desktop_project(
        &mut self,
        path: &std::path::Path,
    ) -> Result<String, String> {
        let first = self.files.active_project().is_none();
        let project = self.files.load_project(path).map_err(|cause| {
            format!("project {path:?}: expected a readable native project: {cause}")
        })?;
        let location = project.path.display().to_string();
        if first {
            let root = project.bound.lookup.templates_root().map(ToOwned::to_owned);
            self.project_template_catalog =
                document_creation::ProjectTemplateCatalog::load(root.as_deref(), |package| {
                    document_creation::project_material(&self.files, package)
                });
        }
        Ok(location)
    }
    /// Report each document's file and whether closing it would lose work.
    /// Example: `owner.gui_document_storage()?` for one GUI snapshot.
    pub(super) fn gui_document_storage(
        &self,
    ) -> Result<std::collections::BTreeMap<u64, oblikovati_gui::GuiDocumentStorage>, String> {
        let workspace = self.files.workspace();
        workspace
            .documents()
            .iter()
            .copied()
            .map(|id| {
                let path = self
                    .files
                    .path(id)
                    .ok()
                    .map(|path| path.display().to_string());
                // A bound document's identity counts; a new part's unused identity does not.
                let lost = if path.is_some() {
                    workspace.needs_save(id)
                } else {
                    workspace.recipe_changed(id)
                };
                let unsaved_edits = lost.map_err(|cause| cause.to_string())?;
                Ok((
                    id,
                    oblikovati_gui::GuiDocumentStorage {
                        path,
                        unsaved_edits,
                    },
                ))
            })
            .collect()
    }
    pub(super) fn gui_document_model_states(
        &self,
    ) -> Result<std::collections::BTreeMap<u64, String>, String> {
        let workspace = self.files.workspace();
        let mut result = std::collections::BTreeMap::new();
        for &id in workspace.documents() {
            let session = workspace.session(id).map_err(|cause| cause.to_string())?;
            if let Some(active) = session
                .snapshot()
                .model_states
                .as_ref()
                .and_then(|states| states.active())
            {
                result.insert(id, active.name.clone());
            }
        }
        Ok(result)
    }
    /// Guard and apply the existing document activation; example: `owner.activate_gui_document(&request)?`.
    pub(super) fn activate_gui_document(
        &mut self,
        request: &ActivateDocumentArgs,
    ) -> Result<bool, String> {
        self.require_gui_document_switch(request.id)?;
        if self.files.workspace().active_document() == Some(request.id) {
            return Ok(false);
        }
        self.model_pick_cue = None;
        self.park_active_sketch_edit()?;
        Client::new(self.document_host())
            .documents()
            .activate(request)
            .map_err(|cause| format!("Activate document {}: {cause}", request.id))?;
        self.active_window = None;
        let slot = self.active_view_slot();
        self.resume_parked_sketch_edit(slot, request.id)?;
        Ok(true)
    }
    /// An edit stays with its document while another one is active; nothing in its recipe changes.
    pub(super) fn park_active_sketch_edit(&mut self) -> Result<(), String> {
        let Some(edit) = self.active_sketch_edit() else {
            return Ok(());
        };
        let slot = self.active_view_slot();
        self.retire_sketch_tool()?;
        self.sketch_edit = None;
        self.parked_sketch_edits.insert(slot, edit.sketch);
        Ok(())
    }
    pub(super) fn resume_parked_sketch_edit(
        &mut self,
        slot: u64,
        document: u64,
    ) -> Result<(), String> {
        match self.parked_sketch_edits.remove(&slot) {
            Some(sketch) => self.begin_sketch_edit(document, sketch),
            None => Ok(()),
        }
    }
    pub(super) fn require_gui_document_switch(&self, target: u64) -> Result<(), String> {
        let row = self
            .files
            .document_info(target)
            .map_err(|cause| cause.to_string())?;
        if !row.visible {
            return Err(format!(
                "Activate document {target}, visible={}: expected a visible document",
                row.visible
            ));
        }
        for document in self
            .files
            .workspace()
            .active_document()
            .into_iter()
            .chain([target])
        {
            self.require_switch_outside_transaction(target, document)?;
        }
        Ok(())
    }
    fn require_switch_outside_transaction(&self, target: u64, document: u64) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.in_transaction() {
            return Err(format!(
                "Activate document {target}, open transaction in {document}: expected both documents outside transactions"
            ));
        }
        Ok(())
    }
}

fn project_settings(
    configuration: &oblikovati_persistence::project_file::ProjectConfiguration,
) -> Vec<(String, String)> {
    vec![
        ("Workspace".to_string(), configuration.workspace.clone()),
        ("Libraries".to_string(), configuration.libraries.join(", ")),
    ]
}
/// A project file holds these folders; this build keeps no other per-project options, so the
/// rows it cannot read say what the application does for every project.
fn project_groups(
    configuration: &oblikovati_persistence::project_file::ProjectConfiguration,
) -> Vec<oblikovati_gui::GuiProjectGroup> {
    let group = |label: &str, rows: Vec<String>| oblikovati_gui::GuiProjectGroup {
        label: label.to_string(),
        rows,
    };
    vec![
        group(
            "Folder Options",
            vec![
                format!("Templates: {}", configuration.templates),
                format!("Design Data: {}", configuration.design_data),
                "Presets: Default".to_string(),
            ],
        ),
        group(
            "Options",
            vec![
                "Old Versions To Keep: 0".to_string(),
                "Using Unique File Names: No".to_string(),
            ],
        ),
        group(
            "Appearance Libraries",
            vec!["Default Appearances".to_string()],
        ),
        group("Material Libraries", vec!["Default Materials".to_string()]),
    ]
}
