// SPDX-License-Identifier: GPL-2.0-only
//! Keep tool windows on the shared dockable window path.
use std::collections::BTreeMap;
use std::path::{Path, PathBuf};
use syn::visit::{self, Visit};
use syn::{Expr, ExprCall, ExprMethodCall, ItemFn, ItemImpl, ItemMod, Lit, Meta};

// Fixed chrome and transient modal file pickers, wizards, and confirmations
// have a separate owner. Every entry is exact, so another window fails the guard.
const INFRASTRUCTURE: &[(&str, &str)] = &[
    (
        "application/document-creation/assembly_place_dialog.rs",
        "modal:FILE_CHOOSER",
    ),
    ("application/document-creation/workspaces.rs", "title"),
    (
        "application/document-tabs/mod.rs",
        "##document-list-control",
    ),
    ("application/document-tabs/previews.rs", "##tab-preview"),
    (
        "application/home-recent-documents/mod.rs",
        "##home-workspace",
    ),
    (
        "application/home-recent-documents/mod.rs",
        "##home-side-panel",
    ),
    (
        "application/quick-access-toolbar/mod.rs",
        "##quick-access-below",
    ),
    (
        "application/quick-access-toolbar/plane_selection.rs",
        "##plane-selection",
    ),
    ("application/viewport-orientation/mod.rs", "CONTROL_WINDOW"),
    (
        "application/viewport-orientation/mod.rs",
        "NAVIGATION_WINDOW",
    ),
    (
        "application/viewport-orientation/navigation_wheel.rs",
        "WINDOW",
    ),
    (
        "application/workspace-layout/browser_feature.rs",
        "SELECTION_TOOLBAR",
    ),
    ("application/workspace-layout/mod.rs", "Model"),
    ("application/workspace-layout/mod.rs", "##panel-plus"),
    ("application/workspace-layout/mod.rs", "##browser-boundary"),
    ("application/workspace-layout/status_bar.rs", "##status-bar"),
    ("chrome/side.rs", "title"),
    ("chrome.rs", "##application-ribbon"),
    ("document_tabs.rs", "<dynamic>"),
    ("document_tabs.rs", "##open-document-tabs"),
    ("ribbon_panel_additional.rs", "<dynamic>"),
    ("ribbon_panels.rs", "##ribbon-temporary-panel"),
    ("sketch/command_alias.rs", "##command-alias"),
    ("sketch/line/mod.rs", "##sketch-hud"),
    ("dockable_window.rs", "title"),
    ("ribbon_reset.rs", "modal:RESET_TITLE"),
    ("application/document-creation/mod.rs", "modal:New document"),
    (
        "application/document-creation/component_dialog.rs",
        "modal:TITLE",
    ),
    (
        "application/document-creation/component_dialog.rs",
        "modal:FILE_CHOOSER",
    ),
    (
        "application/document-creation/workspaces.rs",
        "modal:INSERT",
    ),
    (
        "application/document-saving/close_prompt.rs",
        "modal:EXIT_ENVIRONMENT",
    ),
    (
        "application/document-saving/close_prompt.rs",
        "modal:SAVE_CHANGES",
    ),
    ("application/document-saving/save_as.rs", "modal:TITLE"),
    (
        "application/document-saving/save_as.rs",
        "modal:IMAGE_CHOOSER",
    ),
    (
        "application/document-saving/save_as.rs",
        "modal:CONFIRM_REPLACE",
    ),
    ("application/document-tabs/mod.rs", "modal:SELECT_WINDOW"),
    ("application/document-opening/mod.rs", "modal:TITLE"),
    ("application/document-opening/mod.rs", "modal:DEFER_TITLE"),
    (
        "application/home-recent-documents/mod.rs",
        "modal:NOTICE_TITLE",
    ),
    ("application/model-browser/color_dialog.rs", "modal:TITLE"),
    (
        "application/model-browser/find_prompts.rs",
        "modal:GROUP_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:NO_MATCH_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:SAVE_TITLE",
    ),
    (
        "application/model-browser/find_prompts.rs",
        "modal:OPEN_TITLE",
    ),
    (
        "application/project-dialog/new_project_wizard.rs",
        "modal:TITLE",
    ),
    (
        "application/project-dialog/new_project_wizard.rs",
        "modal:BROWSER_TITLE",
    ),
    (
        "application/ribbon-customization/mod.rs",
        "modal:Ribbon Settings Error",
    ),
    (
        "application/ribbon-customization/settings_dialog.rs",
        "modal:<unknown>",
    ),
    (
        "application/ribbon-customization/settings_dialog.rs",
        "modal:MISSING_FILE",
    ),
    ("sketch/dimension-edit/mod.rs", "modal:WARNING"),
];

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
            let segments: Vec<_> = path
                .path
                .segments
                .iter()
                .map(|part| part.ident.to_string())
                .collect();
            if segments.ends_with(&["Window".into(), "new".into()])
                || segments.ends_with(&["Ui".into(), "window".into()])
                || segments.last().is_some_and(|part| part == "igBegin")
            {
                self.direct_constructors += 1;
            }
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
#[derive(Default)]
struct DockPolicy {
    builder: bool,
    forbidden: bool,
}
impl<'ast> Visit<'ast> for DockPolicy {
    fn visit_path(&mut self, path: &'ast syn::Path) {
        for part in &path.segments {
            self.builder |= part.ident == "dockable_window";
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
    match call.args.first() {
        Some(Expr::Lit(expr)) => match &expr.lit {
            Lit::Str(text) => text.value(),
            _ => "<non-string>".into(),
        },
        Some(Expr::Path(path)) => path.path.segments.last().unwrap().ident.to_string(),
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
    let syntax = syn::parse_file(source).expect("expected valid Rust source");
    let mut calls = WindowCalls::default();
    calls.visit_file(&syntax);
    calls
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
    let expected: BTreeMap<_, _> = INFRASTRUCTURE
        .iter()
        .map(|(path, name)| ((path.to_string(), name.to_string()), 1))
        .collect();
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
    assert_eq!(calls_in("fn f(ui: &Ui, open: &mut bool) { Tool::dockable_window(ui, \"Tool\", open).flags(WindowFlags::NO_DOCKING); }").disabled_docking, 1);
    assert_eq!(calls_in("fn f(ui: &Ui, open: &mut bool) { Tool::dockable_window(ui, \"Tool\", open).title_bar(false); }").disabled_docking, 1);
    assert_eq!(
        calls_in("impl DockableWindow for Tool { fn dockable_window() {} }").overridden_trait,
        1
    );
}
