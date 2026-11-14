// SPDX-License-Identifier: GPL-2.0-only
//! Go head/icon/svgroles.go: inherited sentinel paints become separate coverage passes.
use std::collections::BTreeMap;
use xml::{
    reader::XmlEvent as ReadEvent,
    writer::{EventWriter, XmlEvent as WriteEvent},
};
const PAINTS: [&str; 4] = ["#00ff00", "#0000ff", "#ff0000", "#000000"];
#[derive(Clone, Copy)]
struct PaintState {
    stroke: Option<usize>,
    fill: Option<usize>,
}
impl Default for PaintState {
    fn default() -> Self {
        Self {
            stroke: None,
            fill: Some(3),
        }
    }
}
struct SvgRolePass {
    writer: EventWriter<Vec<u8>>,
    parents: Vec<PaintState>,
    role: Option<usize>,
    elements: usize,
}
pub(super) fn role_document(source: &[u8], role: Option<usize>) -> Result<Vec<u8>, String> {
    if source.len() > 65536 || source.windows(9).any(|bytes| bytes == b"<!DOCTYPE") {
        return Err(format!(
            "icon SVG {} bytes: expected at most 65536 bytes without a document type",
            source.len()
        ));
    }
    let mut pass = SvgRolePass {
        writer: EventWriter::new(Vec::new()),
        parents: Vec::new(),
        role,
        elements: 0,
    };
    for event in xml::EventReader::new(source) {
        pass.consume(
            event.map_err(|cause| format!("icon SVG: expected well-formed XML: {cause}"))?,
        )?;
    }
    Ok(pass.writer.into_inner())
}
impl SvgRolePass {
    fn consume(&mut self, event: ReadEvent) -> Result<(), String> {
        match event {
            ReadEvent::StartElement {
                name, attributes, ..
            } => self.start(name, attributes),
            ReadEvent::EndElement { .. } => {
                self.parents.pop();
                self.write(WriteEvent::end_element().into())
            }
            ReadEvent::StartDocument { .. }
            | ReadEvent::EndDocument
            | ReadEvent::Comment(_)
            | ReadEvent::Whitespace(_) => Ok(()),
            ReadEvent::Characters(text) if text.trim().is_empty() => Ok(()),
            other => Err(format!(
                "icon SVG event {other:?}: expected self-contained path shapes without text or external resources"
            )),
        }
    }
    fn start(
        &mut self,
        name: xml::name::OwnedName,
        attributes: Vec<xml::attribute::OwnedAttribute>,
    ) -> Result<(), String> {
        validate_namespace(name.namespace.as_deref())?;
        let name = name.local_name;
        self.elements += 1;
        validate_element(&name, &attributes, self.parents.len(), self.elements)?;
        let (attrs, state) = self.attributes(&name, attributes)?;
        self.parents.push(state);
        let mut element = WriteEvent::start_element(name.as_str());
        for (key, value) in &attrs {
            element = element.attr(key.as_str(), value.as_str());
        }
        self.write(element.into())
    }
    fn attributes(
        &self,
        name: &str,
        attributes: Vec<xml::attribute::OwnedAttribute>,
    ) -> Result<(BTreeMap<String, String>, PaintState), String> {
        let mut attrs = attributes
            .into_iter()
            .map(|attr| (attr.name.local_name, attr.value))
            .collect();
        let state = inherited_paints(&attrs, self.parents.last().copied().unwrap_or_default())?;
        if name == "svg" {
            attrs.insert("xmlns".into(), "http://www.w3.org/2000/svg".into());
        }
        if !matches!(name, "svg" | "g") {
            attrs.insert("stroke".into(), self.keep(state.stroke).into());
            attrs.insert("fill".into(), self.keep(state.fill).into());
        }
        Ok((attrs, state))
    }
    fn keep(&self, paint: Option<usize>) -> &'static str {
        if paint.is_some() && (self.role.is_none() || self.role == paint) {
            "#000000"
        } else {
            "none"
        }
    }
    fn write(&mut self, event: WriteEvent<'_>) -> Result<(), String> {
        self.writer
            .write(event)
            .map_err(|cause| format!("icon SVG: expected a complete role document: {cause}"))
    }
}
fn validate_element(
    name: &str,
    attrs: &[xml::attribute::OwnedAttribute],
    depth: usize,
    count: usize,
) -> Result<(), String> {
    let shape = matches!(
        name,
        "svg" | "g" | "path" | "rect" | "circle" | "ellipse" | "line" | "polyline" | "polygon"
    );
    let external = attrs.iter().any(|attr| {
        matches!(
            attr.name.local_name.as_str(),
            "href" | "style" | "class" | "filter" | "mask" | "clip-path"
        )
    });
    if !shape || external || depth >= 64 || count > 4096 {
        return Err(format!(
            "icon element {name:?}, depth {depth}, count {count}, external/style {external}: expected self-contained vector shapes, depth below 64 and at most 4096 elements"
        ));
    }
    Ok(())
}
fn inherited_paints(
    attrs: &BTreeMap<String, String>,
    mut state: PaintState,
) -> Result<PaintState, String> {
    if let Some(value) = attrs.get("stroke") {
        state.stroke = paint_role(value)?;
    }
    if let Some(value) = attrs.get("fill") {
        state.fill = paint_role(value)?;
    }
    Ok(state)
}
fn paint_role(value: &str) -> Result<Option<usize>, String> {
    let lower = value.trim().to_ascii_lowercase();
    let expanded = match lower.as_str() {
        "none" | "transparent" | "" => return Ok(None),
        "black" | "#000" => "#000000",
        "red" | "#f00" => "#ff0000",
        "lime" | "#0f0" => "#00ff00",
        "blue" | "#00f" => "#0000ff",
        other => other,
    };
    PAINTS
        .iter()
        .position(|paint| *paint == expanded)
        .map(Some)
        .ok_or_else(|| {
            format!(
                "icon paint {value:?}: expected a role sentinel (black, red, blue, lime) or none"
            )
        })
}

fn validate_namespace(namespace: Option<&str>) -> Result<(), String> {
    if namespace.is_some_and(|uri| uri != "http://www.w3.org/2000/svg") {
        return Err(format!(
            "icon namespace {namespace:?}: expected the SVG namespace"
        ));
    }
    Ok(())
}
