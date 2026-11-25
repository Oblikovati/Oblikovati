// SPDX-License-Identifier: GPL-2.0-only
//! A versioned format for ribbon customization settings.
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
const TABS: [(RibbonTab, &str); 6] = [
    (RibbonTab::Model, "model"),
    (RibbonTab::Sketch, "sketch"),
    (RibbonTab::Annotate, "annotate"),
    (RibbonTab::Tools, "tools"),
    (RibbonTab::Manage, "manage"),
    (RibbonTab::View, "view"),
];

pub(super) fn encode(settings: &SettingsDocument) -> Result<Vec<u8>, String> {
    let version = if settings.large_builtins.is_empty() {
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
    for (tab, entries) in &settings.commands {
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
            ([parent], "tab") if parent == "ribbon-customization" => self.tab(attrs)?,
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
            _ => {
                return Err(format!(
                    "XML element {tag:?}: expected version 1 through 6 ribbon settings"
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
            _ => {
                return Err(format!("ribbon version {version:?}: expected 1 through 6"));
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
            .ok_or_else(|| format!("ribbon target {id:?}: expected a known part tab"))?;
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
mod tests {
    use super::*;
    #[test]
    fn round_trip_preserves_tabs_commands_and_options() {
        let commands = RibbonSettings::from([(
            RibbonTab::Tools,
            vec![
                Entry {
                    command: Command::Sketches,
                    large: true,
                    text: false,
                    expanded: true,
                },
                Entry {
                    command: Command::Separator,
                    large: false,
                    text: false,
                    expanded: false,
                },
                Entry {
                    command: Command::Zoom(
                        crate::application::viewport_orientation::ZoomAction::Window,
                    ),
                    large: false,
                    text: true,
                    expanded: false,
                },
            ],
        )]);
        let settings = SettingsDocument {
            commands,
            expanded_builtins: BTreeSet::from(["part.tools.clipboard.copy".into()]),
            ungrouped_builtins: BTreeSet::from(["part.model.work-features.plane".into()]),
            large_builtins: BTreeSet::from([crate::shell_ribbon::ID.into()]),
            appearance: RibbonAppearance::Compact,
        };
        assert_eq!(decode(&encode(&settings).unwrap()).unwrap(), settings);
    }
    #[test]
    fn every_command_has_a_distinct_stable_identifier() {
        let ids: std::collections::BTreeSet<_> = COMMANDS.iter().map(|item| item.id()).collect();
        assert_eq!(ids.len(), COMMANDS.len());
    }
    #[test]
    fn version_one_commands_remain_in_the_main_panel() {
        let xml = b"<ribbon-customization version=\"1\"><tab id=\"tools\"><command id=\"sketch-visibility\" large=\"true\" text=\"false\"/></tab></ribbon-customization>";
        let settings = decode(xml).unwrap();
        assert!(!settings.commands[&RibbonTab::Tools][0].expanded);
        assert!(settings.commands[&RibbonTab::Tools][0].large);
        assert_eq!(decode(&encode(&settings).unwrap()).unwrap(), settings);
    }
    #[test]
    fn version_two_and_invalid_builtin_locations_are_checked() {
        let old = b"<ribbon-customization version=\"2\"><tab id=\"tools\"><command id=\"measure\" large=\"false\" text=\"true\" expanded=\"true\"/></tab></ribbon-customization>";
        assert!(decode(old).unwrap().commands[&RibbonTab::Tools][0].expanded);
        for xml in [
            "<ribbon-customization version=\"3\"><expanded-builtin id=\"bad id\"/></ribbon-customization>",
            "<ribbon-customization version=\"3\"><expanded-builtin id=\"part.tools.clipboard.copy\"/><expanded-builtin id=\"part.tools.clipboard.copy\"/></ribbon-customization>",
            "<ribbon-customization version=\"2\"><expanded-builtin id=\"part.tools.clipboard.copy\"/></ribbon-customization>",
        ] {
            assert!(decode(xml.as_bytes()).is_err(), "{xml}");
        }
    }
    #[test]
    fn old_settings_default_to_grouped_and_invalid_group_entries_are_rejected() {
        let old = b"<ribbon-customization version=\"4\" appearance=\"normal\"/>";
        assert!(decode(old).unwrap().ungrouped_builtins.is_empty());
        for xml in [
            "<ribbon-customization version=\"4\" appearance=\"normal\"><ungrouped-builtin id=\"part.model.work-features.plane\"/></ribbon-customization>",
            "<ribbon-customization version=\"5\" appearance=\"normal\"><ungrouped-builtin id=\"bad id\"/></ribbon-customization>",
            "<ribbon-customization version=\"5\" appearance=\"normal\"><ungrouped-builtin id=\"part.model.work-features.plane\"/><ungrouped-builtin id=\"part.model.work-features.plane\"/></ribbon-customization>",
        ] {
            assert!(decode(xml.as_bytes()).is_err(), "{xml}");
        }
    }
    #[test]
    fn legacy_settings_default_to_normal_and_invalid_appearances_fail() {
        for version in ["1", "2", "3"] {
            let xml = format!("<ribbon-customization version=\"{version}\"/>");
            assert_eq!(
                decode(xml.as_bytes()).unwrap().appearance,
                RibbonAppearance::Normal
            );
        }
        for xml in [
            "<ribbon-customization version=\"4\"/>",
            "<ribbon-customization version=\"4\" appearance=\"unknown\"/>",
            "<ribbon-customization version=\"3\" appearance=\"normal\"/>",
        ] {
            assert!(decode(xml.as_bytes()).is_err(), "{xml}");
        }
    }
    #[test]
    fn supported_versions_load_and_unknown_version_reports_the_expected_set() {
        for version in 1..=6 {
            let appearance = if version >= 4 {
                " appearance=\"normal\""
            } else {
                ""
            };
            let xml = format!("<ribbon-customization version=\"{version}\"{appearance}/>");
            assert!(
                decode(xml.as_bytes()).unwrap().large_builtins.is_empty(),
                "version {version} should default to standard button sizes"
            );
        }
        assert_eq!(
            decode(b"<ribbon-customization version=\"99\"/>").unwrap_err(),
            "ribbon version Some(\"99\"): expected 1 through 6"
        );
    }
    #[test]
    fn rejects_invalid_versions_duplicate_commands_and_dtd() {
        for text in [
            "<ribbon-customization version=\"99\"/>",
            "<!DOCTYPE x><ribbon-customization version=\"1\"/>",
            "<ribbon-customization version=\"1\"><tab id=\"tools\"><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/><command id=\"sketch-visibility\" large=\"false\" text=\"false\"/></tab></ribbon-customization>",
        ] {
            assert!(decode(text.as_bytes()).is_err());
        }
    }
}
