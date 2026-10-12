// SPDX-License-Identifier: GPL-2.0-only
use super::{EditorPosition, ScriptEditor, calls::identifier_start, search::current_epoch};
use crate::vocabulary;
use std::{
    collections::BTreeMap,
    rc::{Rc, Weak},
};
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum CompletionKind {
    Keyword,
    Builtin,
    Module,
    Method,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CompletionCandidate {
    pub text: String,
    pub kind: CompletionKind,
    pub detail: String,
}
#[derive(Default)]
struct MethodNode {
    children: BTreeMap<String, MethodNode>,
    method: bool,
}
/// Immutable method snapshot. Rebuild when the host's available methods change.
pub struct CompletionEngine {
    root: MethodNode,
}
impl CompletionEngine {
    /// Index canonical host method names; example: `CompletionEngine::new(&host_methods)`.
    pub fn new(methods: &[String]) -> Self {
        let mut root = MethodNode::default();
        for method in methods {
            let mut node = &mut root;
            for segment in method.split('.') {
                node = node.children.entry(segment.into()).or_default();
            }
            node.method = true;
        }
        Self { root }
    }
    /// Suggest sorted segments and their character replacement start; example: `engine.suggest("lo", 2)`.
    pub fn suggest(&self, line: &str, column: usize) -> (Vec<CompletionCandidate>, usize) {
        let characters: Vec<char> = line.chars().collect();
        let (chain, start) = completion_chain(&characters, column);
        let prefix = chain.last().expect("one completion segment");
        let candidates = match chain.as_slice() {
            [_] => bare_candidates(prefix),
            [root, rest @ ..] if root == "oblikovati" => {
                self.api_candidates(&rest[..rest.len() - 1], prefix)
            }
            _ => Vec::new(),
        };
        (candidates, start)
    }
    fn api_candidates(&self, path: &[String], prefix: &str) -> Vec<CompletionCandidate> {
        let mut node = &self.root;
        for segment in path {
            let Some(child) = node.children.get(segment) else {
                return Vec::new();
            };
            node = child;
        }
        node.children
            .iter()
            .filter(|(name, _)| prefix_fold(name, prefix))
            .map(|(name, child)| api_candidate(name, child, path))
            .collect()
    }
}
fn completion_chain(line: &[char], column: usize) -> (Vec<String>, usize) {
    let end = column.min(line.len());
    let mut start = identifier_start(line, end);
    let replace_start = start;
    let mut chain = vec![line[start..end].iter().collect()];
    while start > 0 && line[start - 1] == '.' {
        let dot = start - 1;
        let first = identifier_start(line, dot);
        if first == dot {
            break;
        }
        chain.push(line[first..dot].iter().collect());
        start = first;
    }
    chain.reverse();
    (chain, replace_start)
}
fn prefix_fold(name: &str, prefix: &str) -> bool {
    name.get(..prefix.len())
        .is_some_and(|start| start.eq_ignore_ascii_case(prefix))
}
fn bare_candidates(prefix: &str) -> Vec<CompletionCandidate> {
    let mut candidates: Vec<CompletionCandidate> = [
        (vocabulary::KEYWORDS, CompletionKind::Keyword),
        (vocabulary::BUILTINS, CompletionKind::Builtin),
    ]
    .into_iter()
    .flat_map(|(words, kind)| {
        words
            .iter()
            .filter(move |word| prefix_fold(word, prefix))
            .map(move |word| CompletionCandidate {
                text: (*word).into(),
                kind,
                detail: String::new(),
            })
    })
    .collect();
    candidates.sort_by(|left, right| left.text.cmp(&right.text));
    candidates
}
fn api_candidate(name: &str, node: &MethodNode, path: &[String]) -> CompletionCandidate {
    let kind = if node.method && node.children.is_empty() {
        CompletionKind::Method
    } else {
        CompletionKind::Module
    };
    let parent = path.join(".");
    let detail = if parent.is_empty() {
        format!("oblikovati.{name}")
    } else {
        format!("oblikovati.{parent}.{name}")
    };
    CompletionCandidate {
        text: name.into(),
        kind,
        detail,
    }
}
/// A popup result tied to the editor source and caret that produced it.
pub struct EditorCompletion {
    candidates: Vec<CompletionCandidate>,
    start: EditorPosition,
    caret: EditorPosition,
    epoch: Weak<()>,
}
impl EditorCompletion {
    /// Borrow the sorted popup choices; example: `popup.candidates()[0].text`.
    pub fn candidates(&self) -> &[CompletionCandidate] {
        &self.candidates
    }
    /// Read the prefix's first character position; example: `popup.replace_start()`.
    pub fn replace_start(&self) -> EditorPosition {
        self.start
    }
}
impl ScriptEditor {
    /// Query this caret's source line; example: `let popup = editor.complete(&engine);`.
    pub fn complete(&self, engine: &CompletionEngine) -> EditorCompletion {
        let caret = self.selection.caret;
        let line: String = self.buffer.lines[caret.line].iter().collect();
        let (candidates, column) = engine.suggest(&line, caret.column);
        EditorCompletion {
            candidates,
            start: EditorPosition { column, ..caret },
            caret,
            epoch: Rc::downgrade(&self.source_epoch),
        }
    }
    /// Replace the prefix as one undoable edit; example: `editor.accept_completion(&popup, 0)?;`.
    pub fn accept_completion(
        &mut self,
        popup: &EditorCompletion,
        index: usize,
    ) -> Result<(), String> {
        if !current_epoch(&popup.epoch, &self.source_epoch) || popup.caret != self.selection.caret {
            return Err(format!(
                "completion {:?}..{:?}: expected current source and caret",
                popup.start, popup.caret
            ));
        }
        let candidate = popup.candidates.get(index).ok_or_else(|| {
            format!(
                "completion index {index}: expected index below {}",
                popup.candidates.len()
            )
        })?;
        self.replace(popup.start, popup.caret, &candidate.text);
        Ok(())
    }
}
