// SPDX-License-Identifier: GPL-2.0-only
//! A versioned format for ribbon customization settings.
use super::interface_export::{SECTIONS, Sections};
use super::state::{COMMANDS, Command, Entry};
use crate::ribbon::RibbonTab;
use crate::ribbon_command_layout::RibbonAppearance;
use std::collections::{BTreeMap, BTreeSet};
use xml::reader::XmlEvent;
use xml::writer::{EmitterConfig, XmlEvent as WriteEvent};

pub(super) type RibbonSettings = BTreeMap<RibbonTab, Vec<Entry>>;
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub(super) struct SettingsDocument {
    pub commands: RibbonSettings,
    pub expanded_builtins: BTreeSet<String>,
    pub ungrouped_builtins: BTreeSet<String>,
    pub large_builtins: BTreeSet<String>,
    pub appearance: RibbonAppearance,
}

pub(super) const MAX_SETTINGS_BYTES: usize = 1024 * 1024;
const TABS: [(RibbonTab, &str); 7] = [
    (RibbonTab::Model, "model"),
    (RibbonTab::Sketch, "sketch"),
    (RibbonTab::Annotate, "annotate"),
    (RibbonTab::Tools, "tools"),
    (RibbonTab::Manage, "manage"),
    (RibbonTab::View, "view"),
    (RibbonTab::Assemble, "assembly-design"),
];

pub(super) fn encode(settings: &SettingsDocument) -> Result<Vec<u8>, String> {
    encode_document(settings, None)
}

pub(super) fn encode_with_interface(
    settings: &SettingsDocument,
    sections: &Sections,
) -> Result<Vec<u8>, String> {
    encode_document(settings, Some(sections))
}

fn encode_document(
    settings: &SettingsDocument,
    interface: Option<&Sections>,
) -> Result<Vec<u8>, String> {
    let version = if interface.is_some() {
        "8"
    } else if settings.commands.contains_key(&RibbonTab::Assemble) {
        "7"
    } else if settings.large_builtins.is_empty() {
        "5"
    } else {
        "6"
    };
    let mut writer = EmitterConfig::new()
        .perform_indent(true)
        .create_writer(Vec::new());
    writer
        .write(
            WriteEvent::start_element("ribbon-customization")
                .attr("version", version)
                .attr("appearance", settings.appearance.id()),
        )
        .map_err(|error| format!("XML root: expected writable settings: {error}"))?;
    if interface.is_some() {
        writer
            .write(WriteEvent::start_element("custom-panels"))
            .map_err(|error| error.to_string())?;
    }
    for (tab, entries) in &settings.commands {
        let id = TABS
            .iter()
            .find(|(known, _)| known == tab)
            .map(|(_, id)| *id)
            .ok_or_else(|| format!("ribbon tab {tab:?}: expected a known ribbon target"))?;
        writer
            .write(WriteEvent::start_element("tab").attr("id", id))
            .map_err(|error| error.to_string())?;
        for entry in entries {
            writer
                .write(
                    WriteEvent::start_element("command")
                        .attr("id", entry.command.id())
                        .attr("large", if entry.large { "true" } else { "false" })
                        .attr("text", if entry.text { "true" } else { "false" })
                        .attr("expanded", if entry.expanded { "true" } else { "false" }),
                )
                .map_err(|error| error.to_string())?;
            writer
                .write(WriteEvent::end_element())
                .map_err(|error| error.to_string())?;
        }
        writer
            .write(WriteEvent::end_element())
            .map_err(|error| error.to_string())?;
    }
    if interface.is_some() {
        writer
            .write(WriteEvent::end_element())
            .map_err(|error| error.to_string())?;
    }
    for (element, ids) in [
        ("expanded-builtin", &settings.expanded_builtins),
        ("ungrouped-builtin", &settings.ungrouped_builtins),
        ("large-builtin", &settings.large_builtins),
    ] {
        for id in ids {
            validate_builtin_id(id)?;
            writer
                .write(WriteEvent::start_element(element).attr("id", id))
                .map_err(|error| error.to_string())?;
            writer
                .write(WriteEvent::end_element())
                .map_err(|error| error.to_string())?;
        }
    }
    if let Some(interface) = interface {
        for (name, value) in SECTIONS.into_iter().zip(interface) {
            let state = value.to_string();
            writer
                .write(WriteEvent::start_element(name).attr("state", &state))
                .map_err(|error| error.to_string())?;
            writer
                .write(WriteEvent::end_element())
                .map_err(|error| error.to_string())?;
        }
    }
    writer
        .write(WriteEvent::end_element())
        .map_err(|error| error.to_string())?;
    Ok(writer.into_inner())
}

pub(super) fn decode(bytes: &[u8]) -> Result<SettingsDocument, String> {
    let source = std::str::from_utf8(bytes)
        .map_err(|error| format!("settings bytes: expected UTF-8 XML: {error}"))?;
    if source.to_ascii_lowercase().contains("<!doctype") {
        return Err("XML document type: expected settings without a document type".into());
    }
    let mut decoder = SettingsDecoder::default();
    for event in xml::EventReader::from_str(source) {
        decoder
            .event(event.map_err(|error| format!("settings XML: expected valid XML: {error}"))?)?;
    }
    if !decoder.root || !decoder.path.is_empty() {
        return Err("XML document: expected one complete ribbon-customization root".into());
    }
    Ok(SettingsDocument {
        commands: decoder.settings,
        expanded_builtins: decoder.expanded_builtins,
        ungrouped_builtins: decoder.ungrouped_builtins,
        large_builtins: decoder.large_builtins,
        appearance: decoder.appearance,
    })
}
#[derive(Default)]
struct SettingsDecoder {
    settings: RibbonSettings,
    expanded_builtins: BTreeSet<String>,
    ungrouped_builtins: BTreeSet<String>,
    large_builtins: BTreeSet<String>,
    path: Vec<String>,
    current: Option<RibbonTab>,
    root: bool,
    version: u8,
    appearance: RibbonAppearance,
    export_sections: BTreeSet<String>,
}
impl SettingsDecoder {
    fn event(&mut self, event: XmlEvent) -> Result<(), String> {
        match event {
            XmlEvent::StartElement {
                name, attributes, ..
            } => {
                let attrs: Vec<_> = attributes
                    .iter()
                    .map(|attr| (attr.name.local_name.as_str(), attr.value.as_str()))
                    .collect();
                self.start(name.local_name, &attrs)?;
            }
            XmlEvent::EndElement { .. } => {
                self.path.pop();
                if self.path.len() == 1 {
                    self.current = None;
                }
            }
            XmlEvent::Characters(text) if !text.trim().is_empty() => {
                return Err(format!("XML text {text:?}: expected no text content"));
            }
            XmlEvent::StartDocument { .. }
            | XmlEvent::EndDocument
            | XmlEvent::Whitespace(_)
            | XmlEvent::Characters(_)
            | XmlEvent::Comment(_) => {}
            other => {
                return Err(format!(
                    "XML event {other:?}: expected ribbon settings elements"
                ));
            }
        }
        Ok(())
    }
    fn start(&mut self, tag: String, attrs: &[(&str, &str)]) -> Result<(), String> {
        match (self.path.as_slice(), tag.as_str()) {
            ([], "ribbon-customization") if !self.root => self.root_element(attrs)?,
            ([parent], "tab") if parent == "ribbon-customization" && self.version < 8 => {
                self.tab(attrs)?
            }
            ([parent], "custom-panels")
                if parent == "ribbon-customization" && self.version == 8 =>
            {
                if !attrs.is_empty() || !self.export_sections.insert(tag.clone()) {
                    return Err("custom-panels: expected one section without attributes".into());
                }
            }
            ([root, panels], "tab")
                if root == "ribbon-customization" && panels == "custom-panels" =>
            {
                self.tab(attrs)?
            }
            ([parent], section)
                if parent == "ribbon-customization"
                    && self.version == 8
                    && SECTIONS.contains(&section) =>
            {
                self.export_section(section, attrs)?;
            }
            ([parent], "expanded-builtin")
                if parent == "ribbon-customization" && self.version >= 3 =>
            {
                self.builtin(attrs)?
            }
            ([parent], "ungrouped-builtin")
                if parent == "ribbon-customization" && self.version >= 5 =>
            {
                let id = attribute(attrs, "id", 1)?;
                validate_builtin_id(id)?;
                if !self.ungrouped_builtins.insert(id.to_owned()) {
                    return Err(format!(
                        "built-in group {id:?}: expected a unique group state"
                    ));
                }
            }
            ([parent], "large-builtin")
                if parent == "ribbon-customization" && self.version >= 6 =>
            {
                let id = attribute(attrs, "id", 1)?;
                validate_builtin_id(id)?;
                if !self.large_builtins.insert(id.to_owned()) {
                    return Err(format!("built-in size {id:?}: expected a unique size"));
                }
            }
            ([root, tab], "command") if root == "ribbon-customization" && tab == "tab" => {
                self.command(attrs)?
            }
            ([root, panels, tab], "command")
                if root == "ribbon-customization" && panels == "custom-panels" && tab == "tab" =>
            {
                self.command(attrs)?;
            }
            _ => {
                return Err(format!(
                    "XML element {tag:?}: expected version 1 through 8 ribbon settings"
                ));
            }
        }
        self.path.push(tag);
        Ok(())
    }
    fn root_element(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let version = attrs
            .iter()
            .find(|(name, _)| *name == "version")
            .map(|(_, value)| *value);
        self.version = match version {
            Some("1") => 1,
            Some("2") => 2,
            Some("3") => 3,
            Some("4") => 4,
            Some("5") => 5,
            Some("6") => 6,
            Some("7") => 7,
            Some("8") => 8,
            _ => {
                return Err(format!("ribbon version {version:?}: expected 1 through 8"));
            }
        };
        let count = if self.version >= 4 { 2 } else { 1 };
        attribute(attrs, "version", count)?;
        if self.version >= 4 {
            self.appearance = RibbonAppearance::from_id(attribute(attrs, "appearance", count)?)?;
        }
        self.root = true;
        Ok(())
    }
    fn export_section(&mut self, section: &str, attrs: &[(&str, &str)]) -> Result<(), String> {
        let raw = attribute(attrs, "state", 1)?;
        let payload = serde_json::from_str::<serde_json::Value>(raw).map_err(|error| {
            format!("{section}: expected an interface settings object: {error}")
        })?;
        if !payload.is_object() || !self.export_sections.insert(section.to_owned()) {
            return Err(format!("{section}: expected one interface settings object"));
        }
        Ok(())
    }
    fn builtin(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let id = attribute(attrs, "id", 1)?;
        validate_builtin_id(id)?;
        if !self.expanded_builtins.insert(id.to_owned()) {
            return Err(format!(
                "built-in command {id:?}: expected a unique placement"
            ));
        }
        Ok(())
    }
    fn tab(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let id = attribute(attrs, "id", 1)?;
        let tab = TABS
            .iter()
            .find(|(_, value)| *value == id)
            .map(|(tab, _)| *tab)
            .ok_or_else(|| format!("ribbon target {id:?}: expected a known ribbon tab"))?;
        if tab == RibbonTab::Assemble && self.version < 7 {
            return Err("assembly design ribbon settings require version 7".into());
        }
        if self.settings.insert(tab, Vec::new()).is_some() {
            return Err(format!("ribbon target {id:?}: expected a single tab entry"));
        }
        self.current = Some(tab);
        Ok(())
    }
    fn command(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let entry = decode_entry(attrs, self.version)?;
        let tab = self.current.ok_or("command: expected a parent tab")?;
        let rows = self
            .settings
            .get_mut(&tab)
            .ok_or("command: expected a known tab")?;
        if entry.command != Command::Separator
            && rows.iter().any(|row| row.command == entry.command)
        {
            return Err(format!(
                "command {:?}: expected a unique command per tab",
                entry.command.label()
            ));
        }
        rows.push(entry);
        Ok(())
    }
}
fn attribute<'a>(attrs: &[(&'a str, &'a str)], key: &str, count: usize) -> Result<&'a str, String> {
    if attrs.len() != count {
        return Err(format!(
            "XML attributes {attrs:?}: expected {count} known attributes"
        ));
    }
    attrs
        .iter()
        .find(|(name, _)| *name == key)
        .map(|(_, value)| *value)
        .ok_or_else(|| format!("XML attributes {attrs:?}: expected {key:?}"))
}
fn decode_entry(attrs: &[(&str, &str)], version: u8) -> Result<Entry, String> {
    let count = if version == 1 { 3 } else { 4 };
    let name = attribute(attrs, "id", count)?;
    let command = COMMANDS
        .into_iter()
        .find(|item| item.id() == name)
        .ok_or_else(|| format!("ribbon command {name:?}: expected a known command"))?;
    let flag = |key| {
        attribute(attrs, key, count)?
            .parse::<bool>()
            .map_err(|_| format!("{key} in {attrs:?}: expected true or false"))
    };
    Ok(Entry {
        command,
        large: flag("large")?,
        text: flag("text")?,
        expanded: version >= 2 && flag("expanded")?,
    })
}

fn validate_builtin_id(id: &str) -> Result<(), String> {
    let parts: Vec<_> = id.split('.').collect();
    if parts.len() == 4
        && parts.iter().all(|part| {
            !part.is_empty()
                && part.len() <= 64
                && part
                    .bytes()
                    .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
        })
    {
        return Ok(());
    }
    Err(format!(
        "built-in command id {id:?}: expected four dot-separated lowercase identifiers"
    ))
}

#[cfg(test)]
#[path = "settings_xml_tests.rs"]
mod tests;
