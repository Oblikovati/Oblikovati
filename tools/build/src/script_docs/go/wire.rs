// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(crate) fn parse_wire(source: &str, help: &mut WireHelp) -> BuildResult<()> {
    let source = GoSource::parse(source)?;
    for constant in source.nodes(source.tree.root_node(), "const_spec") {
        collect_constant(&source, constant, help)?;
    }
    for declaration in source.nodes(source.tree.root_node(), "type_spec") {
        let shape = source.field(declaration, "type")?;
        if shape.kind() != "struct_type" {
            continue;
        }
        help.requests.insert(
            source.field_text(declaration, "name")?.into(),
            request_fields(&source, shape)?,
        );
    }
    Ok(())
}
fn collect_constant(
    source: &GoSource<'_>,
    constant: Node<'_>,
    help: &mut WireHelp,
) -> BuildResult<()> {
    let Some(value) = constant.child_by_field_name("value") else {
        return Ok(());
    };
    let names = source
        .children(constant)
        .into_iter()
        .filter(|child| child.kind() == "identifier");
    for (name, value) in names.zip(source.children(value)) {
        if !matches!(
            value.kind(),
            "interpreted_string_literal" | "raw_string_literal"
        ) {
            continue;
        }
        help.constants
            .insert(source.text(name).into(), string_value(source.text(value))?);
    }
    Ok(())
}
fn string_value(value: &str) -> BuildResult<String> {
    if let Some(raw) = value
        .strip_prefix('`')
        .and_then(|raw| raw.strip_suffix('`'))
    {
        return Ok(raw.into());
    }
    serde_json::from_str(value).map_err(|error| {
        format!("Go string {value:?}: expected a JSON-compatible API string literal: {error}")
    })
}
fn shape_fields<'tree>(source: &GoSource<'_>, shape: Node<'tree>) -> Vec<Node<'tree>> {
    let Some(list) = source
        .children(shape)
        .into_iter()
        .find(|child| child.kind() == "field_declaration_list")
    else {
        return Vec::new();
    };
    source
        .children(list)
        .into_iter()
        .filter(|child| child.kind() == "field_declaration")
        .collect()
}
fn request_fields(source: &GoSource<'_>, shape: Node<'_>) -> BuildResult<Vec<MethodParameter>> {
    let mut parameters = Vec::new();
    for field in shape_fields(source, shape) {
        parameters.extend(field_parameters(source, field)?);
    }
    Ok(parameters)
}
fn field_parameters(source: &GoSource<'_>, field: Node<'_>) -> BuildResult<Vec<MethodParameter>> {
    let names = field_names(source, field);
    let json = field_json_name(source, field)?;
    if json.as_deref() == Some("-") {
        return Ok(Vec::new());
    }
    let type_name = compact_type(source, field)?;
    Ok(names
        .map(|name| MethodParameter {
            name: json
                .as_deref()
                .filter(|name| !name.is_empty())
                .unwrap_or(source.text(name))
                .into(),
            type_name: type_name.clone(),
        })
        .collect())
}
fn compact_type(source: &GoSource<'_>, field: Node<'_>) -> BuildResult<String> {
    Ok(source
        .field_text(field, "type")?
        .split_whitespace()
        .collect::<String>())
}

fn field_json_name(source: &GoSource<'_>, field: Node<'_>) -> BuildResult<Option<String>> {
    let tag = field
        .child_by_field_name("tag")
        .map(|tag| string_value(source.text(tag)))
        .transpose()?
        .unwrap_or_default();
    json_field(&tag)
}

fn field_names<'tree>(
    source: &GoSource<'_>,
    field: Node<'tree>,
) -> impl Iterator<Item = Node<'tree>> {
    source
        .children(field)
        .into_iter()
        .filter(|child| child.kind() == "field_identifier")
}

fn json_field(tag: &str) -> BuildResult<Option<String>> {
    let mut rest = tag.trim();
    while !rest.is_empty() {
        let (name, value) = rest
            .split_once(':')
            .ok_or_else(|| format!("Go tag {tag:?}: expected key:quoted-value entries"))?;
        let end = quoted_tag_end(value)?;
        let parsed = string_value(&value[..end])?;
        if name == "json" {
            return Ok(Some(parsed.split(',').next().unwrap_or_default().into()));
        }
        rest = value[end..].trim_start();
    }
    Ok(None)
}
fn quoted_tag_end(value: &str) -> BuildResult<usize> {
    if !value.starts_with('"') {
        return Err(format!(
            "Go tag value {value:?}: expected double-quoted text"
        ));
    }
    let mut escaped = false;
    for (index, character) in value.char_indices().skip(1) {
        if character == '"' && !escaped {
            return Ok(index + 1);
        }
        escaped = character == '\\' && !escaped;
    }
    Err(format!("Go tag value {value:?}: expected a closing quote"))
}
