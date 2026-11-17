// SPDX-License-Identifier: GPL-2.0-only
use super::*;
struct ExampleOwner;
impl crate::dockable_window::DockableWindow for ExampleOwner {}

#[derive(Default)]
struct ClosedWindows(Vec<String>);
impl ClosedWindows {
    fn reconcile(&mut self, registry: &mut DockRegistry) -> bool {
        registry
            .reconcile(|title| self.0.push(title.to_string_lossy().into_owned()))
            .unwrap()
    }
}

#[test]
fn explicit_close_is_reported_once_after_a_suspension() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Example", &state)
        .unwrap();
    state.set_open(true);
    assert!(!closed.reconcile(&mut registry));
    state.set_visibility(DockVisibility::Suspended);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Example"]);
    state.set_open(false);
    assert!(!closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Example"]);
    assert!(!closed.reconcile(&mut registry));
}

#[test]
fn reopening_before_reconciliation_preserves_the_tab() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Example", &state)
        .unwrap();
    state.set_open(true);
    closed.reconcile(&mut registry);
    state.set_open(false);
    state.set_open(true);
    assert!(!closed.reconcile(&mut registry));
    assert!(state.is_open());
}

#[test]
fn dropping_an_open_owner_closes_its_tab() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Example", &state)
        .unwrap();
    state.set_open(true);
    closed.reconcile(&mut registry);
    drop(state);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Example"]);
    assert!(!closed.reconcile(&mut registry));
}

#[test]
fn registration_refuses_a_title_with_a_nul() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let error = registry
        .register::<ExampleOwner>("bad\0title", &state)
        .unwrap_err();
    assert!(error.contains("bad\\0title"));
    assert!(error.contains("NUL-free"));
}

#[test]
fn duplicate_owner_registration_is_rejected() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    registry
        .register::<ExampleOwner>("Example", &state)
        .unwrap();
    let error = registry
        .register::<ExampleOwner>("Again", &state)
        .unwrap_err();
    assert!(error.contains("one registration"));
}

#[test]
fn duplicate_active_imgui_identities_are_rejected() {
    let first = DockableState::default();
    let second = DockableState::default();
    let mut registry = DockRegistry::default();
    registry
        .register::<ExampleOwner>("First###shared", &first)
        .unwrap();
    registry
        .register::<ExampleOwner>("Second###shared", &second)
        .unwrap();
    first.set_open(true);
    second.set_open(true);
    let error = registry
        .reconcile(|_| panic!("duplicate should not close a window"))
        .unwrap_err();
    assert!(error.contains("###shared"));
}

#[test]
fn dynamic_title_keeps_its_stable_identity() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("First###shared", &state)
        .unwrap();
    state.set_open(true);
    assert!(!closed.reconcile(&mut registry));
    state.set_title("Second###shared").unwrap();
    assert!(!closed.reconcile(&mut registry));
    state.set_open(false);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Second###shared"]);
}

#[test]
fn dropped_owner_does_not_close_a_new_owner_with_the_same_identity() {
    let first = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("First###shared", &first)
        .unwrap();
    first.set_open(true);
    closed.reconcile(&mut registry);
    drop(first);
    let second = DockableState::default();
    registry
        .register::<ExampleOwner>("Second###shared", &second)
        .unwrap();
    second.set_open(true);
    assert!(!closed.reconcile(&mut registry));
    assert!(closed.0.is_empty());
}

#[test]
fn closing_a_window_does_not_close_a_different_instance() {
    let first = DockableState::default();
    let second = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Window###one", &first)
        .unwrap();
    registry
        .register::<ExampleOwner>("Window###two", &second)
        .unwrap();
    first.set_open(true);
    second.set_open(true);
    closed.reconcile(&mut registry);
    first.set_open(false);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Window###one"]);
    assert!(second.is_open());
}

#[derive(Default)]
struct DockableTypeInventory(std::collections::BTreeSet<String>);
impl<'ast> syn::visit::Visit<'ast> for DockableTypeInventory {
    fn visit_item_impl(&mut self, item: &'ast syn::ItemImpl) {
        let dockable = item.trait_.as_ref().is_some_and(|(_, path, _)| {
            path.segments
                .last()
                .is_some_and(|part| part.ident == "DockableWindow")
        });
        if dockable
            && let syn::Type::Path(path) = item.self_ty.as_ref()
            && let Some(name) = path.path.segments.last()
        {
            self.0.insert(name.ident.to_string());
        }
        syn::visit::visit_item_impl(self, item);
    }
    fn visit_item_mod(&mut self, item: &'ast syn::ItemMod) {
        if !item.attrs.iter().any(|attr| attr.path().is_ident("cfg") && matches!(&attr.meta, syn::Meta::List(list) if list.tokens.to_string().contains("test"))) {
            syn::visit::visit_item_mod(self, item);
        }
    }
}

fn production_sources(root: &std::path::Path, files: &mut Vec<std::path::PathBuf>) {
    for entry in std::fs::read_dir(root).unwrap() {
        let path = entry.unwrap().path();
        if path.is_dir() {
            production_sources(&path, files);
        } else if path.extension().is_some_and(|extension| extension == "rs") {
            files.push(path);
        }
    }
}

#[test]
fn engine_registers_every_dockable_owner() {
    use syn::visit::Visit;
    let mut paths = Vec::new();
    production_sources(
        &std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("src"),
        &mut paths,
    );
    let mut inventory = DockableTypeInventory::default();
    for path in paths {
        let source = std::fs::read_to_string(path).unwrap();
        inventory.visit_file(&syn::parse_file(&source).unwrap());
    }
    let engine = crate::GuiFrameEngine::new().unwrap();
    let registered: std::collections::BTreeSet<String> =
        crate::engine::dock_registration::owner_types(&engine)
            .into_iter()
            .map(str::to_owned)
            .collect();
    assert_eq!(
        inventory.0, registered,
        "every DockableWindow owner must be registered by the engine"
    );
}

#[test]
fn engine_registers_each_multi_instance_and_multi_title_owner() {
    let engine = crate::GuiFrameEngine::new().unwrap();
    let counts = crate::engine::dock_registration::owner_instance_counts(&engine);
    assert_eq!(counts.get("MaterialBrowser"), Some(&2));
    assert_eq!(counts.get("ProjectsDialog"), Some(&2));
    assert_eq!(counts.get("BlendDialog"), Some(&2));
    assert_eq!(counts.get("GuiDimensionValueEditor"), Some(&2));
    assert_eq!(counts.get("ExternalDockWindow"), Some(&4));
}

#[test]
fn dropping_a_newly_opened_owner_before_reconciliation_closes_its_tab() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Example", &state)
        .unwrap();
    state.set_open(true);
    drop(state);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Example"]);
}

#[test]
fn changing_identity_and_closing_in_the_same_frame_closes_both_tabs() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry.register::<ExampleOwner>("Old", &state).unwrap();
    state.set_open(true);
    closed.reconcile(&mut registry);
    state.set_title("New").unwrap();
    state.set_open(false);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Old", "New"]);
}

#[test]
fn closing_a_new_dynamic_window_uses_its_current_title() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Placeholder", &state)
        .unwrap();
    state.set_open(true);
    state.set_title("Actual").unwrap();
    state.set_open(false);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Actual"]);
}

#[test]
fn suspension_detaches_the_tab_once_and_resume_keeps_its_identity() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Tool###stable", &state)
        .unwrap();
    state.set_open(true);
    assert!(!closed.reconcile(&mut registry));
    state.set_suspended(true);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Tool###stable"]);
    assert!(!closed.reconcile(&mut registry));
    state.set_suspended(false);
    assert!(!closed.reconcile(&mut registry));
    state.set_open(false);
    assert!(closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Tool###stable", "Tool###stable"]);
}

#[test]
fn dropping_a_suspended_owner_does_not_detach_its_tab_twice() {
    let state = DockableState::default();
    let mut registry = DockRegistry::default();
    let mut closed = ClosedWindows::default();
    registry
        .register::<ExampleOwner>("Suspended", &state)
        .unwrap();
    state.set_open(true);
    closed.reconcile(&mut registry);
    state.set_suspended(true);
    assert!(closed.reconcile(&mut registry));
    drop(state);
    assert!(!closed.reconcile(&mut registry));
    assert_eq!(closed.0, ["Suspended"]);
}
