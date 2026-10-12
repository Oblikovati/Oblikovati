// SPDX-License-Identifier: GPL-2.0-only
mod clients;
mod generic_methods;
mod wire;
use crate::process::BuildResult;
pub(super) use clients::{ClientHelp, parse_client};
use oblikovati_script::help::MethodParameter;
use std::collections::BTreeMap;
use tree_sitter::{Node, Parser, Tree};
pub(super) use wire::parse_wire;
#[derive(Default)]
pub(super) struct WireHelp {
    pub constants: BTreeMap<String, String>,
    pub requests: BTreeMap<String, Vec<MethodParameter>>,
}
struct GoSource<'source> {
    tree: Tree,
    text: &'source str,
}
impl<'source> GoSource<'source> {
    fn parse(text: &'source str) -> BuildResult<Self> {
        let tree = parsed_tree(text)?;
        let source = Self { tree, text };
        if source.tree.root_node().has_error() {
            let root = source.tree.root_node();
            let error = source
                .nodes(root, "ERROR")
                .into_iter()
                .next()
                .unwrap_or(root);
            return Err(format!(
                "Go source at {:?} {:?}: expected valid API syntax",
                error.start_position(),
                source.text(error)
            ));
        }
        Ok(source)
    }
    fn text(&self, node: Node<'_>) -> &str {
        &self.text[node.start_byte()..node.end_byte().min(self.text.len())]
    }
    fn field<'tree>(&self, node: Node<'tree>, name: &str) -> BuildResult<Node<'tree>> {
        node.child_by_field_name(name).ok_or_else(|| {
            format!(
                "Go {} {:?}: expected field {name:?}",
                node.kind(),
                self.text(node)
            )
        })
    }
    fn nodes<'tree>(&self, root: Node<'tree>, kind: &str) -> Vec<Node<'tree>> {
        let mut result = Vec::new();
        let mut pending = vec![root];
        while let Some(node) = pending.pop() {
            if node.kind() == kind {
                result.push(node);
            }
            let mut cursor = node.walk();
            pending.extend(
                node.named_children(&mut cursor)
                    .collect::<Vec<_>>()
                    .into_iter()
                    .rev(),
            );
        }
        result
    }
    fn children<'tree>(&self, node: Node<'tree>) -> Vec<Node<'tree>> {
        let mut cursor = node.walk();
        node.named_children(&mut cursor)
            .filter(|child| child.kind() != "comment")
            .collect()
    }
    fn field_text(&self, node: Node<'_>, name: &str) -> BuildResult<&str> {
        Ok(self.text(self.field(node, name)?))
    }
    fn wire_name(&self, node: Node<'_>) -> Option<String> {
        let value = self.text(node);
        value
            .strip_prefix("wire.")
            .filter(|name| !name.contains(['{', '}', '[', ']', ' ']))
            .map(String::from)
    }
}

#[cfg(test)]
#[path = "../../tests/script_docs_go.rs"]
mod tests;

fn parsed_tree(text: &str) -> BuildResult<Tree> {
    let mut parser = Parser::new();
    parser
        .set_language(&tree_sitter_go::LANGUAGE.into())
        .map_err(|error| {
            format!("Go grammar: expected compatible tree-sitter language: {error}")
        })?;
    let mut input = generic_methods::normalized(text)?;
    // The grammar requires a final line terminator where Go's scanner inserts one at EOF.
    if !input.ends_with('\n') {
        input.push('\n');
    }
    parser
        .parse(input, None)
        .ok_or_else(|| "Go source: expected a parsed syntax tree".into())
}
