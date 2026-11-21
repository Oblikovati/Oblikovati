// SPDX-License-Identifier: GPL-2.0-only
//! Keep tool windows on the shared dockable window path.
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use syn::visit::{self, Visit};
use syn::{Expr, ExprCall, ExprMethodCall, ItemFn, ItemImpl, ItemMod, Lit, Meta};

#[path = "dockable_window_guard/infrastructure.rs"]
mod infrastructure;
use infrastructure::INFRASTRUCTURE;

#[derive(Default)]
struct WindowCalls {
    names: Vec<String>,
    direct_constructors: usize,
    disabled_docking: usize,
    overridden_trait: usize,
}
impl<'ast> Visit<'ast> for WindowCalls {
    fn visit_expr_method_call(&mut self, call: &'ast ExprMethodCall) {
        if call.method == "window" {
            self.names.push(window_name(call));
        }
        if call.method == "begin_modal_popup" || call.method == "modal_popup_config" {
            self.names.push(format!("modal:{}", window_name(call)));
        }
        if call.method == "flags" && disables_dockable_window(call) {
            self.disabled_docking += 1;
        }
        if call.method == "title_bar" && hides_dockable_title(call) {
            self.disabled_docking += 1;
        }
        visit::visit_expr_method_call(self, call);
    }
    fn visit_expr_call(&mut self, call: &'ast ExprCall) {
        if let Expr::Path(path) = call.func.as_ref() {
            self.qualified_window(call, &path.path);
        }
        visit::visit_expr_call(self, call);
    }
    fn visit_item_mod(&mut self, item: &'ast ItemMod) {
        if !test_only(&item.attrs) {
            visit::visit_item_mod(self, item);
        }
    }
    fn visit_item_fn(&mut self, item: &'ast ItemFn) {
        if !test_only(&item.attrs) {
            visit::visit_item_fn(self, item);
        }
    }
    fn visit_item_impl(&mut self, item: &'ast ItemImpl) {
        if item.trait_.as_ref().is_some_and(|(_, path, _)| {
            path.segments
                .last()
                .is_some_and(|part| part.ident == "DockableWindow")
        }) && !item.items.is_empty()
        {
            self.overridden_trait += 1;
        }
        visit::visit_item_impl(self, item);
    }
}
impl WindowCalls {
    fn qualified_window(&mut self, call: &ExprCall, path: &syn::Path) {
        let parts: Vec<_> = path
            .segments
            .iter()
            .map(|part| part.ident.to_string())
            .collect();
        if parts.ends_with(&["Window".into(), "window".into()]) {
            self.names.push(window_label(call.args.iter().nth(1)));
        }
        if parts.ends_with(&["Window".into(), "new".into()])
            || parts.ends_with(&["Ui".into(), "window".into()])
            || parts.last().is_some_and(|part| part == "igBegin")
        {
            self.direct_constructors += 1;
        }
    }
}
#[derive(Default)]
struct DockPolicy {
    builder: bool,
    forbidden: bool,
}
impl<'ast> Visit<'ast> for DockPolicy {
    fn visit_path(&mut self, path: &'ast syn::Path) {
        for part in &path.segments {
            self.builder |= is_dockable_constructor(&part.ident);
            self.forbidden |= part.ident == "NO_DOCKING"
                || part.ident == "NO_TITLE_BAR"
                || part.ident == "NO_DECORATION";
        }
        visit::visit_path(self, path);
    }
}
fn disables_dockable_window(call: &ExprMethodCall) -> bool {
    let mut receiver = DockPolicy::default();
    receiver.visit_expr(&call.receiver);
    let mut flags = DockPolicy::default();
    for argument in &call.args {
        flags.visit_expr(argument);
    }
    receiver.builder && flags.forbidden
}
fn hides_dockable_title(call: &ExprMethodCall) -> bool {
    let mut receiver = DockPolicy::default();
    receiver.visit_expr(&call.receiver);
    receiver.builder
        && matches!(call.args.first(), Some(Expr::Lit(expr)) if matches!(&expr.lit, Lit::Bool(value) if !value.value))
}
fn test_only(attributes: &[syn::Attribute]) -> bool {
    attributes.iter().any(|attribute| {
        attribute.path().is_ident("test")
            || matches!(&attribute.meta, Meta::List(list) if list.path.is_ident("cfg") && list.tokens.to_string() == "test")
    })
}
fn window_name(call: &ExprMethodCall) -> String {
    window_label(call.args.first())
}
fn window_label(label: Option<&Expr>) -> String {
    match label {
        Some(Expr::Lit(expr)) => match &expr.lit {
            Lit::Str(text) => text.value(),
            _ => "<non-string>".into(),
        },
        Some(Expr::Path(path)) => path.path.segments.last().unwrap().ident.to_string(),
        Some(Expr::Field(field)) => {
            let syn::Member::Named(member) = &field.member else {
                return "<unknown>".into();
            };
            format!("{}.{member}", window_label(Some(&field.base)))
        }
        Some(Expr::Macro(_)) => "<dynamic>".into(),
        _ => "<unknown>".into(),
    }
}
fn source_files(directory: &Path, files: &mut Vec<PathBuf>) {
    for entry in std::fs::read_dir(directory).expect("expected a readable source directory") {
        let path = entry.expect("expected a source entry").path();
        if path.is_dir() {
            source_files(&path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.push(path);
        }
    }
}
fn calls_in(source: &str) -> WindowCalls {
    let mut calls = WindowCalls::default();
    if let Some(syntax) = production_syntax(source) {
        calls.visit_file(&syntax);
    }
    calls
}
fn production_syntax(source: &str) -> Option<syn::File> {
    let syntax = syn::parse_file(source).expect("expected valid Rust source");
    (!test_only(&syntax.attrs)).then_some(syntax)
}
fn inventory(root: &Path) -> BTreeMap<(String, String), usize> {
    let mut files = Vec::new();
    source_files(root, &mut files);
    let mut found = BTreeMap::new();
    for path in files {
        let relative = path
            .strip_prefix(root)
            .unwrap()
            .to_string_lossy()
            .replace('\\', "/");
        let calls = calls_in(&std::fs::read_to_string(&path).unwrap());
        assert_eq!(
            calls.direct_constructors, 0,
            "{relative}: use DockableWindow instead of a direct window constructor"
        );
        assert_eq!(
            calls.disabled_docking, 0,
            "{relative}: tool windows must keep docking and title bars enabled"
        );
        assert_eq!(
            calls.overridden_trait, 0,
            "{relative}: tool windows must use the shared trait implementation"
        );
        for name in calls.names {
            *found.entry((relative.clone(), name)).or_insert(0) += 1;
        }
    }
    found
}
#[test]
fn tool_windows_use_the_dockable_window_trait() {
    let root = Path::new(env!("CARGO_MANIFEST_DIR")).join("src");
    let mut expected = BTreeMap::new();
    for (path, name) in INFRASTRUCTURE {
        *expected
            .entry((path.to_string(), name.to_string()))
            .or_insert(0) += 1;
    }
    assert_eq!(
        inventory(&root),
        expected,
        "new tool windows must use DockableWindow; review chrome exceptions explicitly"
    );
}
#[test]
fn guard_detects_direct_windows_and_ignores_test_fixtures() {
    let calls = calls_in(
        r#"fn production(ui: &Ui) { ui.window("New Tool"); } #[cfg(test)] mod tests { fn example(ui: &Ui) { ui.window("fixture"); } }"#,
    );
    assert_eq!(calls.names, ["New Tool"]);
    assert_eq!(
        calls_in("fn f(ui: &Ui) { Window::new(ui, \"Tool\"); }").direct_constructors,
        1
    );
    assert_eq!(
        calls_in("fn f(ui: &Ui) { ui.begin_modal_popup(\"New Tool\"); }").names,
        ["modal:New Tool"]
    );
    assert_eq!(calls_in("fn f(ui: &Ui, open: &mut bool) { Tool::tabbed_dockable_window(ui, \"Tool\", \"Group\", open).flags(WindowFlags::NO_DOCKING); }").disabled_docking, 1);
    assert_eq!(calls_in("fn f(ui: &Ui, open: &mut bool) { Tool::dockable_window(ui, \"Tool\", open).title_bar(false); }").disabled_docking, 1);
    assert_eq!(
        calls_in("impl DockableWindow for Tool { fn dockable_window() {} }").overridden_trait,
        1
    );
}

#[test]
fn guard_counts_each_modal_call_for_the_same_title() {
    let calls = calls_in(
        "fn f(ui: &Ui) { ui.begin_modal_popup(REJECTION_POPUP); ui.begin_modal_popup(REJECTION_POPUP); }",
    );
    assert_eq!(
        calls.names,
        ["modal:REJECTION_POPUP", "modal:REJECTION_POPUP"]
    );
}

#[test]
fn guard_handles_file_level_test_fixtures_and_names_modal_fields() {
    let fixture = r#"#![cfg(test)] fn render(ui: &Ui) { ui.window("test frame"); }"#;
    assert!(calls_in(fixture).names.is_empty());
    let production =
        r#"#![cfg(not(test))] fn render(ui: &Ui) { ui.modal_popup_config(spec.title); }"#;
    assert_eq!(calls_in(production).names, ["modal:spec.title"]);
}

fn is_dockable_constructor(ident: &syn::Ident) -> bool {
    ident == "dockable_window" || ident == "tabbed_dockable_window"
}

#[derive(Default)]
struct DockableInventory {
    implementations: std::collections::BTreeSet<String>,
    constructions: std::collections::BTreeSet<String>,
    lifecycle_owners: std::collections::BTreeSet<String>,
    owner: Option<String>,
}
impl<'ast> Visit<'ast> for DockableInventory {
    fn visit_item_impl(&mut self, item: &'ast ItemImpl) {
        let previous = self.owner.take();
        self.owner = match item.self_ty.as_ref() {
            syn::Type::Path(path) => path.path.segments.last().map(|part| part.ident.to_string()),
            _ => None,
        };
        if item.trait_.as_ref().is_some_and(|(_, path, _)| {
            path.segments
                .last()
                .is_some_and(|part| part.ident == "DockableWindow")
        }) && let Some(owner) = &self.owner
        {
            self.implementations.insert(owner.clone());
        }
        visit::visit_item_impl(self, item);
        self.owner = previous;
    }
    fn visit_expr_call(&mut self, call: &'ast ExprCall) {
        if let Expr::Path(path) = call.func.as_ref() {
            let parts = &path.path.segments;
            if parts.len() >= 2
                && parts
                    .last()
                    .is_some_and(|part| is_dockable_constructor(&part.ident))
            {
                let name = parts.iter().nth(parts.len() - 2).unwrap().ident.to_string();
                if let Some(owner) = resolve_owner(&name, self.owner.as_deref()) {
                    self.constructions.insert(owner.to_string());
                }
            }
        }
        visit::visit_expr_call(self, call);
    }
    fn visit_expr_method_call(&mut self, call: &'ast ExprMethodCall) {
        if let Some(owner) = registration_type(call)
            .and_then(|name| resolve_owner(&name, self.owner.as_deref()).map(str::to_owned))
        {
            self.lifecycle_owners.insert(owner);
        }
        visit::visit_expr_method_call(self, call);
    }
    fn visit_item_mod(&mut self, item: &'ast ItemMod) {
        if !test_only(&item.attrs) {
            visit::visit_item_mod(self, item);
        }
    }
    fn visit_item_fn(&mut self, item: &'ast ItemFn) {
        if !test_only(&item.attrs) {
            visit::visit_item_fn(self, item);
        }
    }
}

fn resolve_owner<'a>(name: &'a str, enclosing: Option<&'a str>) -> Option<&'a str> {
    if name == "Self" {
        enclosing
    } else {
        Some(name)
    }
}

fn registration_type(call: &ExprMethodCall) -> Option<String> {
    if call.method != "register"
        || !matches!(call.receiver.as_ref(), Expr::Path(path) if path.path.is_ident("registry"))
    {
        return None;
    }
    let Some(syn::GenericArgument::Type(syn::Type::Path(owner))) =
        call.turbofish.as_ref()?.args.first()
    else {
        return None;
    };
    owner
        .path
        .segments
        .last()
        .map(|part| part.ident.to_string())
}

fn dockable_inventory(root: &Path) -> DockableInventory {
    let mut files = Vec::new();
    source_files(root, &mut files);
    let mut inventory = DockableInventory::default();
    for path in files {
        let source = std::fs::read_to_string(path).unwrap();
        if let Some(syntax) = production_syntax(&source) {
            inventory.visit_file(&syntax);
        }
    }
    inventory
}

#[test]
fn every_dockable_window_uses_the_shared_constructor() {
    let inventory = dockable_inventory(&Path::new(env!("CARGO_MANIFEST_DIR")).join("src"));
    assert!(!inventory.implementations.is_empty());
    assert_eq!(inventory.implementations, inventory.constructions);
}

#[test]
fn every_dockable_window_registers_with_the_lifecycle_owner() {
    let inventory = dockable_inventory(&Path::new(env!("CARGO_MANIFEST_DIR")).join("src"));
    let unregistered: Vec<_> = inventory
        .implementations
        .difference(&inventory.lifecycle_owners)
        .collect();
    assert!(
        unregistered.is_empty(),
        "dockable owners without lifecycle registration: {unregistered:?}"
    );
}
