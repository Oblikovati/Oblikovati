// SPDX-License-Identifier: GPL-2.0-only
//! Exchange applied ribbon settings and stage imports through the shared file browser.
use super::settings_xml::{self, SettingsDocument};
use super::{RibbonCustomization, interface_export::Sections};
use crate::application::document_saving::folder_browser::{
    DialogEnd, DialogEscape, FileFilter, FolderBrowser, center_dialog, dialog_buttons,
    file_name_field_focused, full_path,
};
use crate::application::file_system::{GuiFolders, NativeFolders};
use imgui::Ui;
use std::{
    io::{Read, Write},
    path::Path,
    rc::Rc,
};

const MAX_SETTINGS_BYTES: u64 = settings_xml::MAX_SETTINGS_BYTES as u64;
const MISSING_FILE: &str = "File Not Found";

impl RibbonCustomization {
    pub(super) fn settings_file_picker(&mut self, ui: &Ui, export: impl FnOnce() -> Sections) {
        let document = self.applied_document();
        match self.settings.panel(ui, &document, export) {
            Some(Ok(settings)) => {
                self.draft.import(settings.commands);
                self.builtins.import(
                    settings.expanded_builtins,
                    settings.ungrouped_builtins,
                    settings.large_builtins,
                );
                self.staged_appearance = settings.appearance;
                self.settings_error = None;
            }
            Some(Err(error)) => self.settings_error = Some(error),
            None => {}
        }
    }
}
#[derive(Clone, Copy)]
pub(super) enum Mode {
    Import,
    Export,
}
impl Mode {
    fn title(self) -> &'static str {
        match self {
            Self::Import => "Import Customization Settings",
            Self::Export => "Export Customization Settings",
        }
    }
    fn accept(self) -> &'static str {
        match self {
            Self::Import => "Open",
            Self::Export => "Save",
        }
    }
}
trait SettingsFiles {
    fn read(&self, path: &str) -> Result<Vec<u8>, SettingsError>;
    fn create(&self, path: &str, contents: &[u8]) -> Result<(), String>;
}
#[derive(Debug)]
enum SettingsError {
    Missing(String),
    InvalidImport(String),
    Other(String),
}
struct NativeSettingsFiles;
impl SettingsFiles for NativeSettingsFiles {
    fn read(&self, path: &str) -> Result<Vec<u8>, SettingsError> {
        let file = std::fs::File::open(path).map_err(|error| {
            if error.kind() == std::io::ErrorKind::NotFound {
                return SettingsError::Missing(path.to_owned());
            }
            SettingsError::Other(format!("file {path:?}: expected readable XML: {error}"))
        })?;
        let mut bytes = Vec::new();
        file.take(MAX_SETTINGS_BYTES + 1)
            .read_to_end(&mut bytes)
            .map_err(|error| {
                SettingsError::Other(format!("file {path:?}: expected readable XML: {error}"))
            })?;
        if bytes.len() as u64 > MAX_SETTINGS_BYTES {
            return Err(SettingsError::Other(format!(
                "file {path:?}: expected at most {MAX_SETTINGS_BYTES} bytes"
            )));
        }
        Ok(bytes)
    }
    fn create(&self, path: &str, contents: &[u8]) -> Result<(), String> {
        let mut file = std::fs::OpenOptions::new()
            .write(true)
            .create_new(true)
            .open(path)
            .map_err(|error| format!("file {path:?}: expected a new writable file: {error}"))?;
        if let Err(error) = file.write_all(contents) {
            drop(file);
            let _ = std::fs::remove_file(path);
            return Err(format!(
                "file {path:?}: expected complete XML output: {error}"
            ));
        }
        Ok(())
    }
}
struct DialogDraft {
    mode: Mode,
    name: String,
    opening: bool,
    error: Option<String>,
    missing: Option<String>,
    warning_opening: bool,
    focus_name: bool,
    escape: DialogEscape,
}
pub(super) struct SettingsDialog {
    browser: FolderBrowser,
    files: Rc<dyn SettingsFiles>,
    draft: Option<DialogDraft>,
}
impl Default for SettingsDialog {
    fn default() -> Self {
        Self::with_services(Rc::new(NativeFolders), Rc::new(NativeSettingsFiles))
    }
}
impl SettingsDialog {
    fn with_services(folders: Rc<dyn GuiFolders>, files: Rc<dyn SettingsFiles>) -> Self {
        Self {
            browser: FolderBrowser::new(folders),
            files,
            draft: None,
        }
    }
    pub fn request(&mut self, mode: Mode) {
        self.browser.reset();
        self.draft = Some(DialogDraft {
            mode,
            name: String::new(),
            opening: true,
            error: None,
            missing: None,
            warning_opening: false,
            focus_name: false,
            escape: DialogEscape::default(),
        });
    }
    pub fn panel(
        &mut self,
        ui: &Ui,
        snapshot: &SettingsDocument,
        export: impl FnOnce() -> Sections,
    ) -> Option<Result<SettingsDocument, String>> {
        let draft = self.draft.as_mut()?;
        if std::mem::take(&mut draft.opening) {
            ui.open_popup(draft.mode.title());
        }
        center_dialog(ui, [760., 390.]);
        let _dialog = ui.begin_modal_popup(draft.mode.title())?;
        let chosen = self
            .browser
            .controls(ui, "Look in", FileFilter::Extension("xml"));
        if let Some(name) = chosen {
            draft.name = name;
        }
        ui.separator();
        let entered =
            file_name_field_focused(ui, &mut draft.name, std::mem::take(&mut draft.focus_name));
        ui.text("Files of type");
        ui.same_line();
        ui.text("XML Files (*.xml)");
        if let Some(error) = &draft.error {
            ui.text_wrapped(error);
        }
        let end = dialog_buttons(
            ui,
            draft.mode.accept(),
            !draft.name.trim().is_empty(),
            entered,
            &mut draft.escape,
        );
        if end == Some(DialogEnd::Cancelled) {
            self.draft = None;
            ui.close_current_popup();
            return None;
        }
        if end == Some(DialogEnd::Accepted) {
            let path = settings_path(&self.browser.folder, &draft.name, draft.mode);
            let sections = matches!(draft.mode, Mode::Export).then(export);
            match perform(
                self.files.as_ref(),
                draft.mode,
                &path,
                snapshot,
                sections.as_ref(),
            ) {
                Ok(imported) => {
                    self.draft = None;
                    ui.close_current_popup();
                    return imported.map(Ok);
                }
                Err(SettingsError::InvalidImport(error)) => {
                    self.draft = None;
                    ui.close_current_popup();
                    return Some(Err(error));
                }
                Err(SettingsError::Missing(path)) => {
                    draft.missing = Some(path);
                    draft.warning_opening = true;
                }
                Err(SettingsError::Other(error)) => draft.error = Some(error),
            }
        }
        missing_warning(ui, draft);
        None
    }
}
fn missing_warning(ui: &Ui, draft: &mut DialogDraft) {
    let Some(path) = &draft.missing else {
        return;
    };
    if std::mem::take(&mut draft.warning_opening) {
        ui.open_popup(MISSING_FILE);
    }
    center_dialog(ui, [460., 140.]);
    let Some(_warning) = ui.begin_modal_popup(MISSING_FILE) else {
        return;
    };
    ui.text_wrapped(format!("File {path:?} was not found."));
    ui.text("Check the file name.");
    if ui.button("OK") || ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
        draft.missing = None;
        draft.focus_name = true;
        ui.close_current_popup();
    }
}
fn settings_path(folder: &str, name: &str, mode: Mode) -> String {
    let path = full_path(folder, name.trim());
    if matches!(mode, Mode::Export) && Path::new(&path).extension().is_none() {
        return format!("{path}.xml");
    }
    path
}
fn perform(
    files: &dyn SettingsFiles,
    mode: Mode,
    path: &str,
    snapshot: &SettingsDocument,
    export: Option<&Sections>,
) -> Result<Option<SettingsDocument>, SettingsError> {
    match mode {
        Mode::Import => settings_xml::decode(&files.read(path)?)
            .map(Some)
            .map_err(|error| {
                SettingsError::InvalidImport(format!("Cannot import ribbon customization: {error}"))
            }),
        Mode::Export => {
            let contents = match export {
                Some(sections) => settings_xml::encode_with_interface(snapshot, sections),
                None => settings_xml::encode(snapshot),
            }
            .map_err(SettingsError::Other)?;
            if contents.len() as u64 > MAX_SETTINGS_BYTES {
                return Err(SettingsError::Other(format!(
                    "settings {} bytes: expected at most {MAX_SETTINGS_BYTES} bytes",
                    contents.len()
                )));
            }
            files
                .create(path, &contents)
                .map_err(SettingsError::Other)?;
            Ok(None)
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use std::{cell::RefCell, collections::BTreeMap};
    #[derive(Default)]
    struct MemorySettingsFiles(RefCell<BTreeMap<String, Vec<u8>>>);
    impl SettingsFiles for MemorySettingsFiles {
        fn read(&self, path: &str) -> Result<Vec<u8>, SettingsError> {
            self.0
                .borrow()
                .get(path)
                .cloned()
                .ok_or_else(|| SettingsError::Missing(path.to_owned()))
        }
        fn create(&self, path: &str, contents: &[u8]) -> Result<(), String> {
            let mut files = self.0.borrow_mut();
            if files.contains_key(path) {
                return Err(format!("file {path:?}: expected a new file"));
            }
            files.insert(path.into(), contents.into());
            Ok(())
        }
    }
    #[test]
    fn exported_settings_can_be_imported() {
        let files = MemorySettingsFiles::default();
        let mut settings = SettingsDocument::default();
        settings
            .expanded_builtins
            .insert("part.tools.clipboard.copy".into());
        assert_eq!(
            perform(&files, Mode::Export, "settings.xml", &settings, None).unwrap(),
            None
        );
        assert_eq!(
            perform(&files, Mode::Import, "settings.xml", &settings, None).unwrap(),
            Some(settings)
        );
        assert!(
            perform(
                &files,
                Mode::Export,
                "settings.xml",
                &SettingsDocument::default(),
                None,
            )
            .is_err()
        );
        assert!(
            perform(
                &files,
                Mode::Import,
                "missing.xml",
                &SettingsDocument::default(),
                None,
            )
            .is_err()
        );
    }
    #[test]
    fn export_adds_an_extension_only_when_missing() {
        assert!(settings_path("/tmp", "settings", Mode::Export).ends_with("settings.xml"));
        assert!(settings_path("/tmp", "settings.xml", Mode::Import).ends_with("settings.xml"));
    }
    #[test]
    fn export_refuses_settings_the_importer_cannot_read() {
        use crate::application::ribbon_customization::state::{Command, Entry};
        use crate::ribbon::RibbonTab;
        let files = MemorySettingsFiles::default();
        let entry = Entry {
            command: Command::Separator,
            large: false,
            text: false,
            expanded: false,
        };
        let settings = SettingsDocument {
            commands: [(RibbonTab::Tools, vec![entry; 30_000])].into(),
            ..Default::default()
        };
        assert!(perform(&files, Mode::Export, "large.xml", &settings, None).is_err());
        assert!(files.0.borrow().is_empty());
    }
    #[test]
    fn missing_import_is_distinct_from_invalid_xml() {
        let files = MemorySettingsFiles::default();
        let empty = SettingsDocument::default();
        assert!(
            matches!(perform(&files, Mode::Import, "missing.xml", &empty, None), Err(SettingsError::Missing(path)) if path == "missing.xml")
        );
        files.create("invalid.xml", b"<invalid").unwrap();
        assert!(matches!(
            perform(&files, Mode::Import, "invalid.xml", &empty, None),
            Err(SettingsError::InvalidImport(_))
        ));
    }
}
