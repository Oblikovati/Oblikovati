// SPDX-License-Identifier: GPL-2.0-only
//! Keep unit-test context creation behind the test lease.
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use syn::visit::{self, Visit};

#[derive(Default)]
struct RawContextCalls(usize);
impl<'ast> Visit<'ast> for RawContextCalls {
    fn visit_expr_call(&mut self, call: &'ast syn::ExprCall) {
        if let syn::Expr::Path(path) = call.func.as_ref() {
            let names: Vec<_> = path
                .path
                .segments
                .iter()
                .map(|part| part.ident.to_string())
                .collect();
            let native = names.last().is_some_and(|name| name == "igCreateContext");
            let context = names.len() >= 2 && names[names.len() - 2..] == ["Context", "create"];
            self.0 += usize::from(native || context);
        }
        visit::visit_expr_call(self, call);
    }
}
fn raw_context_calls(source: &str) -> usize {
    let syntax = syn::parse_file(source).expect("expected valid Rust source");
    let mut calls = RawContextCalls::default();
    calls.visit_file(&syntax);
    calls.0
}
fn source_files(folder: &Path, files: &mut Vec<PathBuf>) {
    for entry in std::fs::read_dir(folder).expect("expected a readable source folder") {
        let path = entry.expect("expected a source entry").path();
        if path.is_dir() {
            source_files(&path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.push(path);
        }
    }
}
fn raw_context_inventory(root: &Path) -> BTreeMap<String, usize> {
    let mut files = Vec::new();
    source_files(&root.join("src"), &mut files);
    source_files(&root.join("tests"), &mut files);
    let mut calls = BTreeMap::new();
    for path in files {
        let source = std::fs::read_to_string(&path).expect("expected readable Rust source");
        let count = raw_context_calls(&source);
        if count > 0 {
            let relative = path
                .strip_prefix(root)
                .unwrap()
                .to_string_lossy()
                .replace('\\', "/");
            calls.insert(relative, count);
        }
    }
    calls
}
#[test]
fn unit_test_contexts_use_the_shared_lease() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR"));
    let expected = BTreeMap::from([
        ("src/engine.rs".to_owned(), 1),
        ("src/test_context.rs".to_owned(), 1),
        ("tests/context_ownership.rs".to_owned(), 1),
    ]);
    assert_eq!(
        raw_context_inventory(root),
        expected,
        "raw ImGui constructors: expected the engine, leased test helper, and isolated production probe only"
    );
}
#[test]
fn inventory_detects_safe_and_raw_context_constructors() {
    assert_eq!(
        raw_context_calls(
            "fn probe() { Context::create(); imgui::Context::create(); unsafe { imgui::sys::igCreateContext(std::ptr::null_mut()); } }"
        ),
        3
    );
    assert_eq!(
        raw_context_calls("fn probe() { crate::test_context::create(); }"),
        0
    );
}
