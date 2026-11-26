// SPDX-License-Identifier: GPL-2.0-only
//! The Open dialog: choose a file, look at it, and say how it should open.
use crate::application::document_saving::file_list::FileRequest;
use crate::application::document_saving::folder_browser::{
    BROWSER_COLUMN_WIDTH, DialogEnd, DialogEscape, FileTypeList, FolderBrowser, center_dialog,
    dialog_button_clicks, file_name_field, file_type_field, full_path,
};
use crate::application::file_system::{GuiFolders, NativeFolders};
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::OpenDocumentArgs;
use std::rc::Rc;

#[path = "file_choices.rs"]
mod file_choices;
#[path = "file_properties.rs"]
mod file_properties;
use file_choices::{FileChoices, FileKind};
pub use file_properties::GuiInspectedFile;
use file_properties::{FileInspections, FileProperties};

const TITLE: &str = "Open";
const NO_PREVIEW: &str = "No preview image";
/// The list's first row: Open starts by showing every document in the folder.
const ALL_DOCUMENT_TYPES: usize = 0;
const DIALOG_SIZE: [f32; 2] = [1000., 520.];
const PREVIEW_HEIGHT: f32 = 110.;
const DEFER_TITLE: &str = "Deferred Updates";
const DEFER_MESSAGE: &str = "Drawing updates are deferred. Change the defer status to update \
views and to associate added annotations.";

/// What is read of a selected file, to tell the owner once when the selection changes.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
enum OpenPane {
    #[default]
    Options,
    Details,
}
struct OpenDraft {
    name: String,
    types: FileTypeList,
    file_type: usize,
    escape: DialogEscape,
    opening: bool,
    pane: OpenPane,
    choices: FileChoices,
    inspected: Option<String>,
    inspections: FileInspections,
    /// Where a maximized dialog goes back to.
    restored: Option<([f32; 2], [f32; 2])>,
    geometry: Option<([f32; 2], [f32; 2])>,
}
impl OpenDraft {
    fn new(name: String) -> Self {
        Self {
            name,
            types: FileTypeList::opening(),
            file_type: ALL_DOCUMENT_TYPES,
            escape: DialogEscape::default(),
            opening: true,
            pane: OpenPane::default(),
            choices: FileChoices::default(),
            inspected: None,
            inspections: FileInspections::default(),
            restored: None,
            geometry: None,
        }
    }
}
pub(crate) struct OpenDialog {
    browser: FolderBrowser,
    draft: Option<OpenDraft>,
    properties: FileProperties,
    /// Where the last file opened from, so a reopened dialog starts there.
    last_opened: Option<(String, String)>,
    /// A deferred open, waiting for its message to be acknowledged.
    deferred: Option<GuiShellAction>,
    defer_message_off: bool,
    defer_opening: bool,
}
impl Default for OpenDialog {
    fn default() -> Self {
        Self::with_folders(Rc::new(NativeFolders))
    }
}
impl OpenDialog {
    pub(crate) fn register_dockables(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        self.properties.register_dockable(registry)
    }
    pub(crate) fn with_folders(folders: Rc<dyn GuiFolders>) -> Self {
        let mut browser = FolderBrowser::new(folders.clone());
        browser.folders_select_first = true;
        Self {
            browser,
            draft: None,
            properties: FileProperties::new(folders),
            last_opened: None,
            deferred: None,
            defer_message_off: false,
            defer_opening: false,
        }
    }
    /// Open where the last file opened from, or at the last folder; example: Control+O.
    pub(crate) fn request(&mut self) {
        let name = match &self.last_opened {
            Some((folder, name)) => {
                self.browser.folder.clone_from(folder);
                name.clone()
            }
            None => String::new(),
        };
        self.browser.begin_showing();
        self.browser.refresh();
        if !name.is_empty() {
            self.browser.select_file(&name);
        }
        self.draft = Some(OpenDraft::new(name));
    }
    /// A frame must follow so the popup opens without waiting for an event.
    pub(crate) fn is_opening(&self) -> bool {
        self.draft.as_ref().is_some_and(|draft| draft.opening)
            || self.draft.is_some() && self.browser.selection_redraw
            || self.properties.is_opening()
            || self.defer_opening
    }
    pub(crate) fn validate(&self) -> Result<(), String> {
        let name = self.draft.as_ref().map_or("", |draft| draft.name.as_str());
        if name.contains('\0') || self.browser.folder.contains('\0') {
            return Err(format!(
                "open dialog name {name:?}, folder {:?}: expected text without NUL",
                self.browser.folder
            ));
        }
        Ok(())
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.draft.is_none()
            && ui.io().key_ctrl
            && ui.is_key_pressed_no_repeat(imgui::Key::O)
            && !crate::widget_abi::any_popup_open(ui)
        {
            self.request();
        }
        self.defer_message(ui, actions);
        let Some(draft) = &mut self.draft else {
            return;
        };
        if std::mem::take(&mut draft.opening) {
            ui.open_popup(TITLE);
        }
        center_dialog(ui, DIALOG_SIZE);
        if let Some((position, size)) = draft.geometry.take() {
            crate::widget_abi::window_geometry(ui, position, size, imgui::Condition::Always);
        }
        let Some(_dialog) = ui.begin_modal_popup(TITLE) else {
            return;
        };
        self.dialog_contents(ui, state, actions);
    }
    fn dialog_contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(draft) = &mut self.draft else {
            return;
        };
        maximize_button(ui, draft);
        let opened = contents(
            ui,
            state,
            &mut self.browser,
            draft,
            &mut self.properties,
            actions,
        );
        self.properties.panel(ui, &draft.inspections);
        let Some(end) = opened else {
            return;
        };
        self.properties.close();
        match end {
            DialogOutcome::Cancelled => {}
            DialogOutcome::Opened { name, defer } => {
                self.last_opened = Some((self.browser.folder.clone(), name));
                self.defer_or_send(defer, actions);
            }
        }
        self.draft = None;
        ui.close_current_popup();
    }
    fn defer_or_send(&mut self, defer: Option<GuiShellAction>, actions: &mut Vec<GuiShellAction>) {
        let Some(action) = defer else {
            return;
        };
        if self.defer_message_off {
            actions.push(action);
        } else {
            self.deferred = Some(action);
            self.defer_opening = true;
        }
    }
    /// The message a deferred open shows first; acknowledging it opens the file.
    fn defer_message(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if std::mem::take(&mut self.defer_opening) {
            ui.open_popup(DEFER_TITLE);
        }
        let Some(_message) = ui.begin_modal_popup(DEFER_TITLE) else {
            return;
        };
        ui.text_wrapped(DEFER_MESSAGE);
        ui.checkbox(
            "Do not show this message again",
            &mut self.defer_message_off,
        );
        if crate::widget_abi::button(ui, "OK", [80., 0.]) {
            actions.extend(self.deferred.take());
            ui.close_current_popup();
        }
    }
}
enum DialogOutcome {
    Cancelled,
    /// The dialog is done; `defer` is an open that waits for its message.
    Opened {
        name: String,
        defer: Option<GuiShellAction>,
    },
}
fn maximize_button(ui: &Ui, draft: &mut OpenDraft) {
    let label = if draft.restored.is_some() {
        "Restore"
    } else {
        "Maximize"
    };
    if !crate::widget_abi::button(ui, label, [0.; 2]) {
        return;
    }
    draft.geometry = Some(match draft.restored.take() {
        Some(geometry) => geometry,
        None => {
            draft.restored = Some((ui.window_pos(), ui.window_size()));
            ([0., 0.], ui.io().display_size)
        }
    });
}
#[allow(clippy::too_many_arguments)]
fn contents(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &mut FolderBrowser,
    draft: &mut OpenDraft,
    properties: &mut FileProperties,
    actions: &mut Vec<GuiShellAction>,
) -> Option<DialogOutcome> {
    ui.columns(2, "open-layout", true);
    ui.set_column_width(0, BROWSER_COLUMN_WIDTH);
    if let Some(name) = browser.controls(ui, "Look in", draft.types.filter(draft.file_type)) {
        if draft.name != name {
            draft.choices = FileChoices::default();
        }
        draft.name = name;
    }
    if !browser.several_selected().is_empty() {
        draft.name = quoted_names(browser.several_selected());
    }
    ui.next_column();
    let path = full_path(&browser.folder, &draft.name);
    inspect_selection(&path, draft, browser, properties);
    draft.inspections.follow(state, browser);
    let selected = draft.inspected.clone();
    let filter = draft.types.filter(draft.file_type);
    let wanted = selected
        .into_iter()
        .chain(properties.shown_path())
        .chain(browser.listed_for_pictures(filter));
    draft.inspections.ask_next(wanted, actions);
    preview(ui, &draft.inspections, browser, &path);
    if browser.several_selected().is_empty() && browser.highlighted_folder().is_none() {
        file_pane(ui, draft, properties, &path);
    }
    ui.columns(1, "open-layout-end", false);
    ui.separator();
    let entered = file_name_field(ui, &mut draft.name);
    file_type_field(ui, &draft.types, &mut draft.file_type);
    accept(ui, state, browser, draft, properties, entered, actions)
}
/// Read what the file holds once, on the frame a click selects it: a typed name is not a choice
/// yet, and a frame loop must not touch the disk.
fn inspect_selection(
    path: &str,
    draft: &mut OpenDraft,
    browser: &FolderBrowser,
    properties: &mut FileProperties,
) {
    let shown = browser.selected_file().map(|_| path);
    if shown != draft.inspected.as_deref() {
        draft.inspected = shown.map(str::to_owned);
        if let Some(path) = shown {
            properties.read_facts(path);
        }
    }
}
fn preview(ui: &Ui, inspections: &FileInspections, browser: &FolderBrowser, path: &str) {
    ui.child_window("##open-preview")
        .size([0., PREVIEW_HEIGHT])
        .border(true)
        .build(|| {
            // A folder row and several files have no one picture to show.
            let placeholder =
                browser.highlighted_folder().is_some() || !browser.several_selected().is_empty();
            match inspections.picture_of(path).filter(|_| !placeholder) {
                Some(image) => {
                    crate::application::document_saving::save_as::draw_preview_image(ui, image)
                }
                None => ui.text_disabled(NO_PREVIEW),
            }
        });
}
fn file_pane(ui: &Ui, draft: &mut OpenDraft, properties: &mut FileProperties, path: &str) {
    for (label, pane) in [
        ("Options##open-pane", OpenPane::Options),
        ("Details##open-pane", OpenPane::Details),
    ] {
        if crate::widget_abi::dialog_selectable(ui, label, draft.pane == pane) {
            draft.pane = pane;
        }
    }
    let inspected = draft.inspections.of(path);
    match draft.pane {
        OpenPane::Options => {
            let kind = FileKind::of(path);
            let views = inspected.map_or(&[][..], |file| file.design_views.as_slice());
            draft.choices.panel(ui, kind, views);
        }
        OpenPane::Details => file_properties::details_rows(ui, properties, path, inspected),
    }
    if crate::widget_abi::button(ui, "Properties", [0.; 2]) {
        properties.request(path);
    }
}
/// The names of several selected files, as the name field shows them.
fn quoted_names(names: &[String]) -> String {
    names
        .iter()
        .map(|name| format!("\"{name}\""))
        .collect::<Vec<_>>()
        .join(" ")
}
fn accept(
    ui: &Ui,
    state: &GuiCommandState,
    browser: &mut FolderBrowser,
    draft: &mut OpenDraft,
    properties: &mut FileProperties,
    entered: bool,
    actions: &mut Vec<GuiShellAction>,
) -> Option<DialogOutcome> {
    let several: Vec<String> = browser.several_selected().to_vec();
    let requested = match browser.take_request() {
        FileRequest::None => None,
        FileRequest::Open(name) => Some(name),
        FileRequest::Properties(name) => {
            properties.request(&full_path(&browser.folder, &name));
            None
        }
    };
    if let Some(name) = requested {
        return Some(open_one(browser, draft, name, actions));
    }
    let can_open = browser.highlighted_folder().is_some()
        || !several.is_empty()
        || (!draft.name.is_empty()
            && open_action(
                &full_path(&browser.folder, &draft.name),
                false,
                draft.choices.express_for(&draft.name),
            )
            .dispatchable(state));
    let clicked = dialog_button_clicks(ui, "Open", can_open, entered);
    let escaped = draft.escape.pressed(ui)
        && ui.is_window_focused_with_flags(imgui::WindowFocusedFlags::CHILD_WINDOWS);
    match clicked.or(escaped.then_some(DialogEnd::Cancelled))? {
        DialogEnd::Cancelled => Some(DialogOutcome::Cancelled),
        DialogEnd::Accepted => opened(browser, draft, &several, state, actions),
    }
}
/// Open on a folder row enters it; the dialog stays, with the name it had.
fn opened(
    browser: &mut FolderBrowser,
    draft: &OpenDraft,
    several: &[String],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) -> Option<DialogOutcome> {
    if browser.enter_highlighted_folder() {
        return None;
    }
    if several.is_empty() {
        return Some(open_one(browser, draft, draft.name.clone(), actions));
    }
    for path in in_opening_order(browser, several, state) {
        actions.push(open_action(&path, false, draft.choices.express_for(&path)));
    }
    Some(DialogOutcome::Opened {
        name: several[0].clone(),
        defer: None,
    })
}
fn open_one(
    browser: &FolderBrowser,
    draft: &OpenDraft,
    name: String,
    actions: &mut Vec<GuiShellAction>,
) -> DialogOutcome {
    let deferred = draft.choices.defers() && FileKind::of(&name) == FileKind::Drawing;
    let action = open_action(
        &full_path(&browser.folder, &name),
        deferred,
        draft.choices.express_for(&name),
    );
    if deferred {
        return DialogOutcome::Opened {
            name,
            defer: Some(action),
        };
    }
    actions.push(action);
    DialogOutcome::Opened { name, defer: None }
}
/// What is open already comes first, so a newly opened file ends up in front.
fn in_opening_order(
    browser: &FolderBrowser,
    names: &[String],
    state: &GuiCommandState,
) -> Vec<String> {
    let mut paths: Vec<String> = names
        .iter()
        .map(|name| full_path(&browser.folder, name))
        .collect();
    let is_open = |path: &String| {
        state
            .document_storage
            .values()
            .any(|storage| storage.path.as_deref() == Some(path.as_str()))
    };
    paths.sort_by_key(|path| !is_open(path));
    paths
}
fn open_action(path: &str, defer_content: bool, express: bool) -> GuiShellAction {
    if defer_content {
        return GuiShellAction::OpenDeferredDrawing(path.to_owned());
    }
    GuiShellAction::OpenDocument(OpenDocumentArgs {
        full_document_name: path.to_string(),
        visible: true,
        defer_content: false,
        express,
    })
}
#[cfg(test)]
#[path = "../../../tests/application/document-opening/open_dialog.rs"]
mod tests;
