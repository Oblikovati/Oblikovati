// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(crate) struct ClientHelp {
    pub constant: String,
    pub request: Option<String>,
    pub summary: String,
}
pub(crate) fn parse_client(text: &str) -> BuildResult<Vec<ClientHelp>> {
    let source = GoSource::parse(text)?;
    let mut result = Vec::new();
    for method in source.nodes(source.tree.root_node(), "method_declaration") {
        let Some(body) = method.child_by_field_name("body") else {
            continue;
        };
        if let Some((constant, argument)) = transport_call(&source, body)? {
            let request = request_type(&source, method, body, argument)?;
            result.push(ClientHelp {
                constant,
                request,
                summary: summary_before(&source, method),
            });
        }
    }
    Ok(result)
}
fn transport_call<'tree>(
    source: &GoSource<'_>,
    body: Node<'tree>,
) -> BuildResult<Option<(String, Node<'tree>)>> {
    for call in source.nodes(body, "call_expression") {
        let Some((arguments, index)) = transport_arguments(source, call)? else {
            continue;
        };
        let Some(constant) = arguments
            .get(index)
            .and_then(|node| source.wire_name(*node))
        else {
            continue;
        };
        let argument = arguments.get(index + 1).ok_or_else(|| {
            format!("Go transport wire.{constant}: expected request argument after wire name")
        })?;
        return Ok(Some((constant, *argument)));
    }
    Ok(None)
}
fn transport_arguments<'tree>(
    source: &GoSource<'_>,
    call: Node<'tree>,
) -> BuildResult<Option<(Vec<Node<'tree>>, usize)>> {
    let function = source.field_text(call, "function")?;
    let callable = function.split('[').next().unwrap_or_default();
    let index = match callable {
        "call" => 1,
        name if name.ends_with(".call") => 0,
        _ => return Ok(None),
    };
    Ok(Some((
        source.children(source.field(call, "arguments")?),
        index,
    )))
}
fn expression_type(source: &GoSource<'_>, mut expression: Node<'_>) -> BuildResult<Option<String>> {
    if expression.kind() == "unary_expression" {
        expression = source.field(expression, "operand")?;
    }
    if expression.kind() != "composite_literal" {
        return Ok(None);
    }
    Ok(source.wire_name(source.field(expression, "type")?))
}
fn request_type(
    source: &GoSource<'_>,
    method: Node<'_>,
    body: Node<'_>,
    argument: Node<'_>,
) -> BuildResult<Option<String>> {
    if empty_request(source, argument)? {
        return Ok(None);
    }
    if let Some(shape) = expression_type(source, argument)? {
        return Ok(Some(shape));
    }
    let name = source.text(argument);
    if let Some(shape) = parameter_request(source, method, name)? {
        return Ok(Some(shape));
    }
    local_request(source, body, name)?.map(Some).ok_or_else(|| {
        format!("Go request {name:?}: expected nil, wire DTO literal, parameter or local DTO")
    })
}
fn parameter_request(
    source: &GoSource<'_>,
    method: Node<'_>,
    name: &str,
) -> BuildResult<Option<String>> {
    for parameter in source.nodes(source.field(method, "parameters")?, "parameter_declaration") {
        if !source
            .children(parameter)
            .iter()
            .any(|child| child.kind() == "identifier" && source.text(*child) == name)
        {
            continue;
        }
        let shape = source.field(parameter, "type")?;
        return source.wire_name(shape).map(Some).ok_or_else(|| {
            format!(
                "Go request parameter {name:?} type {:?}: expected canonical wire DTO",
                source.text(shape)
            )
        });
    }
    Ok(None)
}
fn local_request(source: &GoSource<'_>, body: Node<'_>, name: &str) -> BuildResult<Option<String>> {
    if let Some(shape) = declared_request(source, body, name) {
        return Ok(Some(shape));
    }
    for declaration in source.nodes(body, "short_var_declaration") {
        let left = source.children(source.field(declaration, "left")?);
        let right = source.children(source.field(declaration, "right")?);
        for (left, right) in left.into_iter().zip(right) {
            if source.text(left) == name {
                return expression_type(source, right);
            }
        }
    }
    Ok(None)
}
fn declared_request(source: &GoSource<'_>, body: Node<'_>, name: &str) -> Option<String> {
    for declaration in source.nodes(body, "var_spec") {
        if !source
            .children(declaration)
            .iter()
            .any(|node| node.kind() == "identifier" && source.text(*node) == name)
        {
            continue;
        }
        if let Some(shape) = declaration
            .child_by_field_name("type")
            .and_then(|node| source.wire_name(node))
        {
            return Some(shape);
        }
    }
    None
}
fn summary_before(source: &GoSource<'_>, method: Node<'_>) -> String {
    source.text[..method.start_byte()]
        .lines()
        .rev()
        .take_while(|line| line.trim_start().starts_with("//"))
        .filter_map(|line| line.trim_start().strip_prefix("//").map(str::trim))
        .find_map(|line| line.strip_prefix("mcp:summary ").map(String::from))
        .unwrap_or_default()
}

fn empty_request(source: &GoSource<'_>, argument: Node<'_>) -> BuildResult<bool> {
    if source.text(argument) == "nil" {
        return Ok(true);
    }
    if argument.kind() != "composite_literal" {
        return Ok(false);
    }
    let shape = source.field(argument, "type")?;
    Ok(shape.kind() == "struct_type" && source.nodes(shape, "field_declaration").is_empty())
}
