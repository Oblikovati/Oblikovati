// SPDX-License-Identifier: GPL-2.0-only
//! Read native template metadata at project load, outside the frame loop.
use super::*;
use oblikovati_api::types::*;
use oblikovati_gui::{
    GuiProjectTemplate, GuiTemplateFolder, GuiTemplateMaterial, GuiTemplateSheet,
};
use oblikovati_model::quantity::Dimension;
use oblikovati_persistence::{
    load::{NativePackageReader, PackageFileReader},
    package::DocumentPackage,
};
use std::{
    path::{Path, PathBuf},
    sync::Arc,
};

#[derive(Default)]
pub(in super::super) struct ProjectTemplateCatalog {
    pub templates: Arc<[GuiProjectTemplate]>,
    pub issue: Option<String>,
}

impl ProjectTemplateCatalog {
    pub(in super::super) fn load(
        root: Option<&Path>,
        part_material: impl Fn(&DocumentPackage) -> GuiTemplateMaterial,
    ) -> Self {
        let Some(root) = root else {
            return Self::default();
        };
        let mut templates = Vec::new();
        let mut issues = Vec::new();
        for (directory, folder) in [
            (root.to_path_buf(), GuiTemplateFolder::Root),
            (
                root.join("Metric"),
                GuiTemplateFolder::Units(GuiTemplateUnits::Metric),
            ),
            (
                root.join("English"),
                GuiTemplateFolder::Units(GuiTemplateUnits::Inch),
            ),
        ] {
            match native_files(&directory) {
                Ok(files) => {
                    collect_templates(files, folder, &part_material, &mut templates, &mut issues)
                }
                Err(cause) => issues.push(cause),
            }
        }
        let issue = (!issues.is_empty()).then(|| issues.join("\n"));
        Self {
            templates: templates.into(),
            issue,
        }
    }
}

fn native_files(directory: &Path) -> Result<Vec<PathBuf>, String> {
    let entries = match std::fs::read_dir(directory) {
        Ok(entries) => entries,
        Err(cause) if cause.kind() == std::io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(cause) => return Err(format!("Template folder {directory:?}: {cause}")),
    };
    let entries: Vec<_> = entries
        .collect::<Result<_, _>>()
        .map_err(|cause| format!("Template folder {directory:?}: {cause}"))?;
    let mut files: Vec<_> = entries
        .into_iter()
        .map(|entry| entry.path())
        .filter(|path| path.is_file() && file_kind(path).is_some())
        .collect();
    files.sort();
    Ok(files)
}

fn file_kind(path: &Path) -> Option<DocumentType> {
    let extension = path.extension()?.to_str()?;
    [
        (PART_FILE_EXTENSION, DOCUMENT_PART),
        (ASSEMBLY_FILE_EXTENSION, DOCUMENT_ASSEMBLY),
        (DRAWING_FILE_EXTENSION, DOCUMENT_DRAWING),
        (PRESENTATION_FILE_EXTENSION, DOCUMENT_PRESENTATION),
    ]
    .into_iter()
    .find_map(|(suffix, kind)| {
        extension
            .eq_ignore_ascii_case(suffix.trim_start_matches('.'))
            .then_some(kind)
    })
}

fn collect_templates(
    files: Vec<PathBuf>,
    folder: GuiTemplateFolder,
    part_material: &impl Fn(&DocumentPackage) -> GuiTemplateMaterial,
    templates: &mut Vec<GuiProjectTemplate>,
    issues: &mut Vec<String>,
) {
    for path in files {
        match read_template(&path, folder, part_material) {
            Ok((template, _)) => templates.push(template),
            Err(cause) => issues.push(format!("Template {path:?}: {cause}")),
        }
    }
}

fn read_template(
    path: &Path,
    folder: GuiTemplateFolder,
    part_material: impl Fn(&DocumentPackage) -> GuiTemplateMaterial,
) -> Result<(GuiProjectTemplate, DocumentPackage), String> {
    let bytes = NativePackageReader
        .read(path)
        .map_err(|cause| cause.to_string())?;
    let package = DocumentPackage::decode(&bytes).map_err(|cause| cause.to_string())?;
    let manifest = package.manifest();
    let expected =
        file_kind(path).ok_or_else(|| "expected a native document extension".to_owned())?;
    if manifest.document_type != expected.0 {
        return Err(format!(
            "document type {}: expected {}",
            manifest.document_type, expected.0
        ));
    }
    let kind = template_kind(expected, &manifest.sub_type)?;
    let title = path
        .file_stem()
        .and_then(|name| name.to_str())
        .ok_or_else(|| "expected a UTF-8 file name".to_owned())?
        .to_owned();
    let units = package
        .document_units()
        .map_err(|cause| cause.to_string())?;
    let sheet_preview = if kind == GuiTemplateKind::Drawing {
        package
            .restore_drawing()
            .map_err(|cause| cause.to_string())?
            .and_then(|drawing| {
                GuiTemplateSheet::from_recipe(&drawing.sheets()[drawing.active_sheet()])
            })
            .map(Arc::new)
    } else {
        None
    };
    Ok((
        GuiProjectTemplate {
            title,
            path: path.to_owned(),
            folder,
            kind,
            length_unit: units.preferred_name(Dimension::Length),
            size: bytes.len() as u64,
            material: matches!(kind, GuiTemplateKind::Part | GuiTemplateKind::SheetMetal)
                .then(|| part_material(&package)),
            sheet_preview,
        },
        package,
    ))
}

fn template_kind(kind: DocumentType, subtype: &str) -> Result<GuiTemplateKind, String> {
    match kind {
        DOCUMENT_PART if subtype == SUB_TYPE_SHEET_METAL_PART.0 => Ok(GuiTemplateKind::SheetMetal),
        DOCUMENT_PART => Ok(GuiTemplateKind::Part),
        DOCUMENT_ASSEMBLY => Ok(GuiTemplateKind::Assembly),
        DOCUMENT_DRAWING => Ok(GuiTemplateKind::Drawing),
        DOCUMENT_PRESENTATION => Ok(GuiTemplateKind::Presentation),
        _ => Err(format!(
            "document type {}: expected a supported template kind",
            kind.0
        )),
    }
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in super::super) fn create_project_template(
        &mut self,
        selected: &GuiProjectTemplate,
    ) -> Result<bool, String> {
        self.clear_layout_request()?;
        let package = self.project_template_package(selected)?;
        self.create_project_recipe(selected.kind, package)
            .map(|_| true)
    }

    pub(super) fn project_template_package(
        &mut self,
        selected: &GuiProjectTemplate,
    ) -> Result<DocumentPackage, String> {
        let state = self.gui_command_state()?;
        if state.transaction_open || state.authoring == oblikovati_gui::GuiAuthoringContext::Sketch
        {
            return Err(
                "Project template creation requires the active edit to be finished.".into(),
            );
        }
        if !self.project_template_catalog.templates.contains(selected) {
            return Err(format!(
                "Template {:?}: expected a file in the active project catalog",
                selected.path
            ));
        }
        let (current, package) = read_template(&selected.path, selected.folder, |package| {
            template_material::project_material(&self.files, package)
        })?;
        if &current != selected {
            return Err(format!(
                "Template {:?} changed since the project was loaded",
                selected.path
            ));
        }
        Ok(package)
    }

    pub(super) fn create_project_recipe(
        &mut self,
        kind: GuiTemplateKind,
        package: DocumentPackage,
    ) -> Result<u64, String> {
        let template = GuiDocumentTemplate {
            kind,
            ..Default::default()
        };
        let name = self.next_new_document_name(template.name_prefix());
        let package = package.with_copy_metadata(&NewFileMetadata {
            display_name: name,
            ..Default::default()
        });
        let observer = self.factory.create()?;
        let created = self
            .files
            .create_unbound(package, observer, &self.version, &mut self.guids)
            .map_err(|cause| cause.to_string())?;
        Ok(created.session_id)
    }
}
