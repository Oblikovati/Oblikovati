// SPDX-License-Identifier: GPL-2.0-only
//! Capture the usable toolbar choices for each document kind.
use super::{AddedEntry, QuickAccessToolbar, STANDARD, StandardCommand};
use oblikovati_api::types::{
    DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION, DocumentType,
};
use serde_json::{Value, json};
use std::collections::BTreeMap;

impl QuickAccessToolbar {
    pub(crate) fn export_settings(&self, active: Option<DocumentType>) -> Value {
        let mut kinds = self.kinds.clone();
        kinds.entry(active).or_default();
        let contexts: BTreeMap<_, _> = kinds
            .into_iter()
            .map(|(document, selection)| {
                let standard: Vec<_> = STANDARD
                    .into_iter()
                    .filter(|command| selection.shows(*command))
                    .map(StandardCommand::label)
                    .collect();
                let added: Vec<_> = selection
                    .added
                    .into_iter()
                    .map(|entry| match entry {
                        AddedEntry::Command(name) => json!({"command": name}),
                        AddedEntry::Separator => json!({"separator": true}),
                    })
                    .collect();
                (
                    context_name(document),
                    json!({"standard": standard, "added": added}),
                )
            })
            .collect();
        json!({
            "placement": if self.below_ribbon { "below-ribbon" } else { "title-bar" },
            "active-context": context_name(active),
            "contexts": contexts,
        })
    }
}

fn context_name(document: Option<DocumentType>) -> String {
    match document {
        None => "home".into(),
        Some(DOCUMENT_ASSEMBLY) => "assembly".into(),
        Some(DOCUMENT_DRAWING) => "drawing".into(),
        Some(DOCUMENT_PART) => "part".into(),
        Some(DOCUMENT_PRESENTATION) => "presentation".into(),
        Some(other) => format!("document-{}", other.0),
    }
}
