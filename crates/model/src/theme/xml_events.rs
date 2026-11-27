// SPDX-License-Identifier: GPL-2.0-only
use xml::reader::{ParserConfig, XmlEvent};
use xml::writer::{EmitterConfig, EventWriter};
use xml::{attribute::OwnedAttribute, name::OwnedName, namespace::Namespace};
const MAX_THEME_XML_BYTES: usize = 16 * 1024 * 1024;
const MAX_THEME_XML_DEPTH: usize = 64;

pub(super) fn parse_events(bytes: &[u8]) -> Result<Vec<XmlEvent>, String> {
    check_size(bytes)?;
    let parser = ParserConfig::new()
        .ignore_comments(false)
        .create_reader(bytes);
    let mut events = Vec::new();
    let mut depth = 0;
    for entry in parser {
        let event = entry
            .map_err(|cause| format!("theme XML {cause}: expected valid attribute-carrying XML"))?;
        check_depth(&event, &mut depth)?;
        events.push(event);
    }
    Ok(events)
}
pub(super) fn write_events(events: &[XmlEvent]) -> Result<Vec<u8>, String> {
    let mut output = Vec::new();
    let config = EmitterConfig::new().write_document_declaration(false);
    let mut writer = EventWriter::new_with_config(&mut output, config);
    for event in events {
        write_retained_event(&mut writer, event)?;
    }
    drop(writer);
    if !output.ends_with(b"\n") {
        output.push(b'\n');
    }
    Ok(output)
}
fn write_retained_event(
    writer: &mut EventWriter<&mut Vec<u8>>,
    event: &XmlEvent,
) -> Result<(), String> {
    if matches!(
        event,
        XmlEvent::StartDocument { .. } | XmlEvent::EndDocument
    ) {
        return Ok(());
    }
    if let Some(event) = event.as_writer_event() {
        writer.write(event).map_err(|cause| {
            format!("theme XML writer {cause}: expected serializable retained events")
        })?;
    }
    Ok(())
}
fn check_size(bytes: &[u8]) -> Result<(), String> {
    if bytes.len() > MAX_THEME_XML_BYTES {
        return Err(format!(
            "theme XML size {}: expected at most {MAX_THEME_XML_BYTES} bytes",
            bytes.len()
        ));
    }
    Ok(())
}
fn check_depth(event: &XmlEvent, depth: &mut usize) -> Result<(), String> {
    match event {
        XmlEvent::StartElement { .. } => *depth += 1,
        XmlEvent::EndElement { .. } => *depth = depth.saturating_sub(1),
        _ => (),
    }
    if *depth > MAX_THEME_XML_DEPTH {
        return Err(format!(
            "theme XML depth {depth}: expected at most {MAX_THEME_XML_DEPTH} elements"
        ));
    }
    Ok(())
}
pub(super) fn matching_end(events: &[XmlEvent], start: usize) -> Result<usize, String> {
    let mut depth = 0usize;
    for (index, event) in events.iter().enumerate().skip(start) {
        match event {
            XmlEvent::StartElement { .. } => depth += 1,
            XmlEvent::EndElement { .. } => {
                depth -= 1;
                if depth == 0 {
                    return Ok(index);
                }
            }
            _ => (),
        }
    }
    Err(format!(
        "theme snapshot at event {start}: expected a balanced closing element"
    ))
}
pub(super) fn snapshot_events(name: &str, colors: Vec<(String, String)>) -> [XmlEvent; 2] {
    let attributes = snapshot_attributes(name, colors);
    let name = OwnedName::local("oblikovati_tokens");
    [
        XmlEvent::StartElement {
            name: name.clone(),
            attributes,
            namespace: Namespace::empty(),
        },
        XmlEvent::EndElement { name },
    ]
}
fn snapshot_attributes(name: &str, colors: Vec<(String, String)>) -> Vec<OwnedAttribute> {
    std::iter::once(("name".into(), name.into()))
        .chain(colors)
        .map(|(name, value)| OwnedAttribute {
            name: OwnedName::local(name),
            value,
        })
        .collect()
}
pub(super) fn replace_attribute(attributes: &mut Vec<OwnedAttribute>, name: &str, value: String) {
    if let Some(attribute) = attributes
        .iter_mut()
        .find(|attribute| attribute.name.local_name == name)
    {
        attribute.value = value;
        return;
    }
    attributes.push(OwnedAttribute {
        name: OwnedName::local(name),
        value,
    });
}
