// SPDX-License-Identifier: GPL-2.0-only
use crate::editor::{EditorPosition, ScriptEditor};
use serde::{Deserialize, Serialize};
use std::collections::{BTreeMap, BTreeSet};
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MethodParameter {
    pub name: String,
    pub type_name: String,
}
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MethodHelp {
    pub wire: String,
    pub summary: String,
    pub parameters: Vec<MethodParameter>,
}
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct MethodHelpSnapshot {
    pub api_version: String,
    pub api_revision: String,
    pub methods: Vec<MethodHelp>,
}
impl MethodHelp {
    /// Render the Lua table-call signature; example: `help.signature()` gives `documents.create{ type }`.
    pub fn signature(&self) -> String {
        if self.parameters.is_empty() {
            return format!("{}{{}}", self.wire);
        }
        let names = self
            .parameters
            .iter()
            .map(|parameter| parameter.name.as_str())
            .collect::<Vec<_>>()
            .join(", ");
        format!("{}{{ {names} }}", self.wire)
    }
}
/// Immutable help for available host methods; the full API catalog does not promise implementation.
pub struct MethodHelpCatalog {
    by_wire: BTreeMap<String, MethodHelp>,
}
impl MethodHelpCatalog {
    /// Index an injected contract snapshot; example: `MethodHelpCatalog::new(snapshot)?;`.
    pub fn new(snapshot: MethodHelpSnapshot) -> Result<Self, String> {
        let mut by_wire = BTreeMap::new();
        for help in snapshot.methods {
            if by_wire.contains_key(&help.wire) {
                return Err(format!(
                    "help method {:?}: expected a unique wire name",
                    help.wire
                ));
            }
            by_wire.insert(help.wire.clone(), help);
        }
        Ok(Self { by_wire })
    }
    /// Load pinned help and retain only available methods; example: `MethodHelpCatalog::published(&host_methods)?;`.
    pub fn published(methods: &[String]) -> Result<Self, String> {
        let mut snapshot: MethodHelpSnapshot = serde_json::from_str(include_str!(
            "../resources/method-help.json"
        ))
        .map_err(|error| format!("embedded method help: expected a contract snapshot: {error}"))?;
        let available: BTreeSet<&str> = methods.iter().map(String::as_str).collect();
        snapshot
            .methods
            .retain(|help| available.contains(help.wire.as_str()));
        Self::new(snapshot)
    }
    /// Read an exact canonical name; example: `catalog.lookup("documents.create")`.
    pub fn lookup(&self, wire: &str) -> Option<&MethodHelp> {
        self.by_wire.get(wire)
    }
    fn chain_help(&self, chain: Vec<String>) -> Option<&MethodHelp> {
        if chain.len() < 2 || chain[0] != "oblikovati" {
            return None;
        }
        self.lookup(&chain[1..].join("."))
    }
    /// Resolve the caret's enclosing host call; example: `catalog.signature_help(&editor)`.
    pub fn signature_help(&self, editor: &ScriptEditor) -> Option<&MethodHelp> {
        self.chain_help(editor.enclosing_call()?)
    }
    /// Resolve the hovered host method; example: `catalog.hover_help(&editor, position)`.
    pub fn hover_help(
        &self,
        editor: &ScriptEditor,
        position: EditorPosition,
    ) -> Option<&MethodHelp> {
        self.chain_help(editor.method_chain_at(position)?)
    }
}
