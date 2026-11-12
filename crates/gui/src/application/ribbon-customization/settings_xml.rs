// SPDX-License-Identifier: GPL-2.0-only
//! A versioned exchange format for the customization draft.
use super::state::{COMMANDS, Command, Entry};
use crate::ribbon::RibbonTab;
use std::collections::BTreeMap;
use xml::reader::XmlEvent;
use xml::writer::{EmitterConfig, XmlEvent as WriteEvent};

pub(super) type RibbonSettings = BTreeMap<RibbonTab, Vec<Entry>>;
pub(super) const MAX_SETTINGS_BYTES: usize = 1024 * 1024;
const TABS: [(RibbonTab, &str); 6] = [
    (RibbonTab::Model, "model"),
    (RibbonTab::Sketch, "sketch"),
    (RibbonTab::Annotate, "annotate"),
    (RibbonTab::Tools, "tools"),
    (RibbonTab::Manage, "manage"),
    (RibbonTab::View, "view"),
];

pub(super) fn encode(settings: &RibbonSettings) -> Result<Vec<u8>, String> {
    let mut writer = EmitterConfig::new()
        .perform_indent(true)
        .create_writer(Vec::new());
    writer
        .write(WriteEvent::start_element("ribbon-customization").attr("version", "1"))
        .map_err(|error| format!("XML root: expected writable settings: {error}"))?;
    for (tab, entries) in settings {
        let id = TABS
            .iter()
            .find(|(known, _)| known == tab)
            .map(|(_, id)| *id)
            .ok_or_else(|| format!("ribbon tab {tab:?}: expected a part ribbon target"))?;
        writer
            .write(WriteEvent::start_element("tab").attr("id", id))
            .map_err(|error| error.to_string())?;
        for entry in entries {
            writer
                .write(
                    WriteEvent::start_element("command")
                        .attr("id", entry.command.id())
                        .attr("large", if entry.large { "true" } else { "false" })
                        .attr("text", if entry.text { "true" } else { "false" }),
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
    writer
        .write(WriteEvent::end_element())
        .map_err(|error| error.to_string())?;
    Ok(writer.into_inner())
}

pub(super) fn decode(bytes: &[u8]) -> Result<RibbonSettings, String> {
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
    Ok(decoder.settings)
}
#[derive(Default)]
struct SettingsDecoder {
    settings: RibbonSettings,
    path: Vec<String>,
    current: Option<RibbonTab>,
    root: bool,
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
            ([], "ribbon-customization") if !self.root && attrs == [("version", "1")] => {
                self.root = true
            }
            ([parent], "tab") if parent == "ribbon-customization" => self.tab(attrs)?,
            ([root, tab], "command") if root == "ribbon-customization" && tab == "tab" => {
                self.command(attrs)?
            }
            _ => {
                return Err(format!(
                    "XML element {tag:?}: expected version 1 ribbon settings"
                ));
            }
        }
        self.path.push(tag);
        Ok(())
    }
    fn tab(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let id = attribute(attrs, "id", 1)?;
        let tab = TABS
            .iter()
            .find(|(_, value)| *value == id)
            .map(|(tab, _)| *tab)
            .ok_or_else(|| format!("ribbon target {id:?}: expected a known part tab"))?;
        if self.settings.insert(tab, Vec::new()).is_some() {
            return Err(format!("ribbon target {id:?}: expected a single tab entry"));
        }
        self.current = Some(tab);
        Ok(())
    }
    fn command(&mut self, attrs: &[(&str, &str)]) -> Result<(), String> {
        let entry = decode_entry(attrs)?;
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
fn decode_entry(attrs: &[(&str, &str)]) -> Result<Entry, String> {
    let name = attribute(attrs, "id", 3)?;
    let command = COMMANDS
        .into_iter()
        .find(|item| item.id() == name)
        .ok_or_else(|| format!("ribbon command {name:?}: expected a known command"))?;
    let flag = |key| {
        attribute(attrs, key, 3)?
            .parse::<bool>()
            .map_err(|_| format!("{key} in {attrs:?}: expected true or false"))
    };
    Ok(Entry {
        command,
        large: flag("large")?,
        text: flag("text")?,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn round_trip_preserves_tabs_commands_and_options() {
        let settings = RibbonSettings::from([(
            RibbonTab::Tools,
            vec![
                Entry {
                    command: Command::Sketches,
                    large: true,
                    text: false,
                },
                Entry {
                    command: Command::Separator,
                    large: false,
                    text: false,
                },
            ],
        )]);
        assert_eq!(decode(&encode(&settings).unwrap()).unwrap(), settings);
    }
    #[test]
    fn every_command_has_a_distinct_stable_identifier() {
        let ids: std::collections::BTreeSet<_> = COMMANDS.iter().map(|item| item.id()).collect();
        assert_eq!(ids.len(), COMMANDS.len());
    }
    #[test]
    fn rejects_invalid_versions_duplicate_commands_and_dtd() {
        for text in [
            "<ribbon-customization version=\"2\"/>",
            "<!DOCTYPE x><ribbon-customization version=\"1\"/>",
            "<ribbon-customization version=\"1\"><tab id=\"tools\"><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/></tab></ribbon-customization>",
        ] {
            assert!(decode(text.as_bytes()).is_err());
        }
    }
}
