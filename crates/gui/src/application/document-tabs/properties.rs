// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiCommandState;
use crate::application::file_system::{FileFacts, GuiFolders, NativeFolders};
use imgui::Ui;
use oblikovati_api::wire::DocumentInfo;
use std::{path::Path, rc::Rc, time::SystemTime};

const POPUP: &str = "###document-properties";
const NOT_AVAILABLE: &str = "Not available";
const NOT_SAVED: &str = "Not saved";

struct PropertiesTarget {
    id: u64,
    /// Read once when the dialog opens: a frame loop must not touch the disk.
    facts: Result<FileFacts, String>,
}
pub(crate) struct DocumentProperties {
    folders: Rc<dyn GuiFolders>,
    target: Option<PropertiesTarget>,
    opening: bool,
}
impl Default for DocumentProperties {
    fn default() -> Self {
        Self::with_folders(Rc::new(NativeFolders))
    }
}
impl DocumentProperties {
    pub(crate) fn with_folders(folders: Rc<dyn GuiFolders>) -> Self {
        Self {
            folders,
            target: None,
            opening: false,
        }
    }
    /// Inspect a document without activating it; example: `properties.request(&state, 7)`.
    pub(crate) fn request(&mut self, state: &GuiCommandState, id: u64) {
        let path = state
            .document_storage
            .get(&id)
            .and_then(|row| row.path.as_deref());
        let facts = path.map_or(Ok(FileFacts::default()), |path| {
            self.folders.file_facts(path)
        });
        self.target = Some(PropertiesTarget { id, facts });
        self.opening = true;
    }
    /// A frame must follow so the dialog opens without waiting for an event.
    pub(crate) fn needs_frame(&self) -> bool {
        self.opening
    }
    /// Draw the read-only dialog; example: `properties.panel(ui, &state)`.
    pub(crate) fn panel(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(target) = &self.target else {
            return;
        };
        let row = state.documents.iter().find(|row| row.id == target.id);
        let name = row.map_or("Document", |row| row.name.as_str());
        if std::mem::take(&mut self.opening) {
            ui.open_popup(POPUP);
        }
        let title = format!("{name} Properties{POPUP}");
        let Some(_dialog) = ui.begin_modal_popup(&title) else {
            return;
        };
        if let Some(_pages) = ui.tab_bar("##property-pages")
            && let Some(_general) = ui.tab_item("General")
        {
            general_page(ui, state, target, row);
        }
        let closed = crate::widget_abi::button(ui, "Close", [80., 0.])
            || ui.is_key_pressed(imgui::Key::Escape);
        if closed {
            ui.close_current_popup();
        }
    }
}
fn general_page(
    ui: &Ui,
    state: &GuiCommandState,
    target: &PropertiesTarget,
    row: Option<&DocumentInfo>,
) {
    let path = state
        .document_storage
        .get(&target.id)
        .and_then(|row| row.path.as_deref());
    let fields = general_fields(row, path, &target.facts);
    for (label, value) in &fields {
        ui.text(format!("{label}: {value}"));
    }
}
/// The General page's rows; an unsaved document names what it does not have yet.
fn general_fields(
    row: Option<&DocumentInfo>,
    path: Option<&str>,
    facts: &Result<FileFacts, String>,
) -> Vec<(&'static str, String)> {
    let name = row.map_or(String::new(), |row| row.name.clone());
    let kind = row.map_or(String::new(), |row| row.type_.clone());
    let path = path.map(Path::new);
    let file_name = path
        .and_then(Path::file_name)
        .map(|file| file.to_string_lossy().into_owned());
    let location = path
        .and_then(Path::parent)
        .map(|folder| folder.display().to_string());
    vec![
        ("Name", name),
        (
            "File name",
            file_name.unwrap_or_else(|| NOT_SAVED.to_string()),
        ),
        ("Type", kind),
        (
            "Location",
            location.unwrap_or_else(|| NOT_SAVED.to_string()),
        ),
        ("Size", size_text(path.is_some(), facts)),
        ("Created", timestamp_field(facts, |facts| facts.created)),
        ("Modified", timestamp_field(facts, |facts| facts.modified)),
        ("Accessed", timestamp_field(facts, |facts| facts.accessed)),
    ]
}
fn size_text(saved: bool, facts: &Result<FileFacts, String>) -> String {
    match facts {
        Ok(facts) if saved => format!("{} bytes", facts.size),
        Ok(_) => NOT_AVAILABLE.to_string(),
        Err(cause) => cause.clone(),
    }
}
fn timestamp_field(
    facts: &Result<FileFacts, String>,
    read: impl Fn(&FileFacts) -> Option<SystemTime>,
) -> String {
    let stamp = facts.as_ref().ok().and_then(read);
    stamp.map_or_else(|| NOT_AVAILABLE.to_string(), timestamp_text)
}
/// UTC, because no time-zone source is available here; example: `2026-11-06 09:30:00 UTC`.
fn timestamp_text(stamp: SystemTime) -> String {
    let seconds = stamp
        .duration_since(SystemTime::UNIX_EPOCH)
        .map_or(0, |since| since.as_secs());
    let (year, month, day) = civil_date((seconds / 86_400) as i64);
    let (hour, minute, second) = (seconds / 3600 % 24, seconds / 60 % 60, seconds % 60);
    format!("{year:04}-{month:02}-{day:02} {hour:02}:{minute:02}:{second:02} UTC")
}
/// Howard Hinnant's days-to-civil algorithm for the proleptic Gregorian calendar.
fn civil_date(days: i64) -> (i64, u32, u32) {
    let shifted = days + 719_468;
    let era = shifted.div_euclid(146_097);
    let day_of_era = shifted.rem_euclid(146_097);
    let year_of_era =
        (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let shifted_month = (5 * day_of_year + 2) / 153;
    let day = (day_of_year - (153 * shifted_month + 2) / 5 + 1) as u32;
    let month = (if shifted_month < 10 {
        shifted_month + 3
    } else {
        shifted_month - 9
    }) as u32;
    let year = year_of_era + era * 400 + i64::from(month <= 2);
    (year, month, day)
}
#[cfg(test)]
#[path = "../../../tests/application/document-tabs/properties.rs"]
mod tests;
