// SPDX-License-Identifier: GPL-2.0-only
use super::xml_events::{
    matching_end, parse_events, replace_attribute, snapshot_events, write_events,
};
use std::collections::BTreeMap;
use xml::reader::XmlEvent;
const SNAPSHOT_PATH: &[&str] = &["oblikovati_tokens"];

#[derive(Clone, Debug)]
pub(super) struct BlenderThemeDocument {
    events: Vec<XmlEvent>,
}
impl BlenderThemeDocument {
    pub(super) fn parse(bytes: &[u8]) -> Result<Self, String> {
        Ok(Self {
            events: parse_events(bytes)?,
        })
    }
    pub(super) fn attribute(&self, path: &[&str], name: &str) -> Option<&str> {
        let index = self.element_index(path)?;
        let XmlEvent::StartElement { attributes, .. } = &self.events[index] else {
            return None;
        };
        attributes
            .iter()
            .find(|attribute| attribute.name.local_name == name)
            .map(|attribute| attribute.value.as_str())
    }
    pub(super) fn set_attribute(
        &mut self,
        path: &[&str],
        name: &str,
        value: String,
    ) -> Result<(), String> {
        let index = self
            .element_index(path)
            .ok_or_else(|| format!("theme element {path:?}: expected a mapped Blender element"))?;
        let XmlEvent::StartElement { attributes, .. } = &mut self.events[index] else {
            return Err(format!("theme event {index}: expected an authored element"));
        };
        replace_attribute(attributes, name, value);
        Ok(())
    }
    pub(super) fn snapshot(&self) -> BTreeMap<String, String> {
        let Some(index) = self.element_index(SNAPSHOT_PATH) else {
            return BTreeMap::new();
        };
        let XmlEvent::StartElement { attributes, .. } = &self.events[index] else {
            return BTreeMap::new();
        };
        attributes
            .iter()
            .map(|attribute| (attribute.name.local_name.clone(), attribute.value.clone()))
            .collect()
    }
    pub(super) fn encode(
        &self,
        name: &str,
        colors: Vec<(String, String)>,
    ) -> Result<Vec<u8>, String> {
        let mut document = self.clone();
        document.replace_snapshot(name, colors)?;
        write_events(&document.events)
    }
    fn element_index(&self, path: &[&str]) -> Option<usize> {
        let mut index = self
            .events
            .iter()
            .position(|event| matches!(event, XmlEvent::StartElement { .. }))?;
        for name in path {
            index = self.first_child(index, name)?;
        }
        Some(index)
    }
    fn first_child(&self, parent: usize, wanted: &str) -> Option<usize> {
        let end = matching_end(&self.events, parent).ok()?;
        let mut index = parent + 1;
        while index < end {
            let XmlEvent::StartElement { name, .. } = &self.events[index] else {
                index += 1;
                continue;
            };
            if name.local_name == wanted {
                return Some(index);
            }
            index = matching_end(&self.events, index).ok()? + 1;
        }
        None
    }
    fn replace_snapshot(
        &mut self,
        name: &str,
        colors: Vec<(String, String)>,
    ) -> Result<(), String> {
        while let Some(start) = self.element_index(SNAPSHOT_PATH) {
            let end = matching_end(&self.events, start)?;
            self.events.drain(start..=end);
        }
        let end = self
            .events
            .iter()
            .rposition(|event| matches!(event, XmlEvent::EndElement { .. }))
            .ok_or("theme XML events: expected a complete root element")?;
        self.events.splice(end..end, snapshot_events(name, colors));
        Ok(())
    }
}
