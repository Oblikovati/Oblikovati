// SPDX-License-Identifier: GPL-2.0-only
//! Keep named files distinct when they share a document kind and units.
use super::template_sort::{ordered_kinds, sort_templates, template_size};
use super::{
    GuiCommandState, GuiDocumentTemplate, GuiTemplateKind, TemplateChooser, TemplateFolder,
};
use crate::{GuiProjectTemplate, GuiShellAction, GuiTemplateUnits};

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) enum TemplateChoice {
    BuiltIn(GuiDocumentTemplate),
    Project(GuiProjectTemplate),
}

impl From<GuiDocumentTemplate> for TemplateChoice {
    fn from(template: GuiDocumentTemplate) -> Self {
        Self::BuiltIn(template)
    }
}

impl TemplateChoice {
    pub(super) fn title(&self) -> String {
        match self {
            Self::BuiltIn(template) => template.title(),
            Self::Project(template) => template.title.clone(),
        }
    }

    pub(super) fn kind(&self) -> GuiTemplateKind {
        match self {
            Self::BuiltIn(template) => template.kind,
            Self::Project(template) => template.kind,
        }
    }

    pub(super) fn type_label(&self) -> &'static str {
        GuiDocumentTemplate {
            kind: self.kind(),
            ..Default::default()
        }
        .type_label()
    }

    pub(super) fn unit_label(&self) -> String {
        match self {
            Self::BuiltIn(template) => template.unit_label().into(),
            Self::Project(template) => match template.length_unit {
                "mm" => "Length: millimeters (mm)".into(),
                "in" => "Length: inches (in)".into(),
                unit => format!("Length: {unit}"),
            },
        }
    }

    pub(super) fn bytes(&self, state: &GuiCommandState) -> Option<u64> {
        match self {
            Self::BuiltIn(template) => template_size(state, template.kind),
            Self::Project(template) => Some(template.size),
        }
    }

    pub(super) fn create_action(&self) -> GuiShellAction {
        match self {
            Self::BuiltIn(template) => GuiShellAction::CreateDocument(*template),
            Self::Project(template) => GuiShellAction::CreateProjectTemplate(template.clone()),
        }
    }
}

impl TemplateChooser {
    pub(crate) fn remember_project_template(&mut self, path: Option<std::path::PathBuf>) {
        self.remembered_project_template = path;
    }

    pub(crate) fn project_template_path(&self) -> Option<&std::path::Path> {
        match &self.selected {
            Some(TemplateChoice::Project(template)) => Some(&template.path),
            Some(TemplateChoice::BuiltIn(_)) => None,
            None => self.remembered_project_template.as_deref(),
        }
    }

    pub(super) fn restore_project_selection(&mut self, state: &GuiCommandState) {
        if self.selected.is_none()
            && let Some(path) = self.remembered_project_template.as_deref()
            && let Some(template) = state
                .project_templates
                .iter()
                .find(|entry| entry.path == path)
        {
            self.folder = template.folder;
            self.selected = Some(TemplateChoice::Project(template.clone()));
            self.remembered_project_template = None;
        }
    }

    pub(super) fn reveal_project_root(&mut self, state: &GuiCommandState) {
        let has_entries = |folder| {
            state
                .project_templates
                .iter()
                .any(|entry| entry.folder == folder)
        };
        if self.selected.is_none() && !has_entries(self.folder) && has_entries(TemplateFolder::Root)
        {
            self.folder = TemplateFolder::Root;
        }
    }

    pub(super) fn catalog(
        &self,
        state: &GuiCommandState,
        kinds: &[GuiTemplateKind],
    ) -> Vec<TemplateChoice> {
        if state.project_templates.is_empty() {
            return match self.folder {
                TemplateFolder::Units(units) => built_in_catalog(state, kinds, self.sort, units),
                TemplateFolder::Root => Vec::new(),
            };
        }
        let mut entries: Vec<_> = state
            .project_templates
            .iter()
            .filter(|entry| entry.folder == self.folder && kinds.contains(&entry.kind))
            .cloned()
            .map(TemplateChoice::Project)
            .collect();
        sort_templates(&mut entries, self.sort, |entry| {
            (entry.title(), entry.bytes(state))
        });
        entries
    }
}

fn built_in_catalog(
    state: &GuiCommandState,
    kinds: &[GuiTemplateKind],
    sort: super::TemplateSort,
    units: GuiTemplateUnits,
) -> Vec<TemplateChoice> {
    ordered_kinds(kinds, sort, state, units)
        .into_iter()
        .map(|kind| GuiDocumentTemplate { kind, units }.into())
        .collect()
}
