// SPDX-License-Identifier: GPL-2.0-only
use super::test_provider::FakeClipboardProvider;
use super::*;
use imgui::ClipboardBackend;
#[test]
fn clipboard_admission_requires_a_positive_budget() {
    assert!(
        GuiClipboard::new(Rc::new(FakeClipboardProvider::default()), 0)
            .err()
            .unwrap()
            .contains("positive")
    );
}
#[test]
fn unicode_and_empty_clipboard_text_round_trip_without_borrowed_storage() {
    let provider = Rc::new(FakeClipboardProvider::default());
    let mut adapter = GuiClipboard::new(provider.clone(), 32).unwrap();
    assert_eq!(adapter.get(), None);
    adapter.set("é界🙂");
    let snapshot = adapter.get().unwrap();
    adapter.set("");
    assert_eq!(snapshot, "é界🙂");
    assert_eq!(adapter.get().as_deref(), Some(""));
    assert_eq!(adapter.issue(), None);
    assert_eq!(provider.writes.borrow().len(), 2);
}
#[test]
fn nul_and_oversized_values_are_declined_before_the_gui_callback_or_native_write() {
    let provider = Rc::new(FakeClipboardProvider::default());
    let mut adapter = GuiClipboard::new(provider.clone(), 4).unwrap();
    for text in ["a\0b", "🙂x"] {
        *provider.text.borrow_mut() = Some(text.into());
        assert_eq!(adapter.get(), None);
        assert!(adapter.issue().unwrap().contains("expected"));
        adapter.set(text);
        assert!(provider.writes.borrow().is_empty());
    }
}
#[test]
fn provider_failures_are_visible_until_a_new_successful_operation() {
    let provider = Rc::new(FakeClipboardProvider::default());
    let mut adapter = GuiClipboard::new(provider.clone(), 32).unwrap();
    *provider.failure.borrow_mut() =
        Some("clipboard peer timeout: expected a complete transfer".into());
    assert_eq!(adapter.get(), None);
    assert!(adapter.issue().unwrap().contains("timeout"));
    adapter.set("new");
    assert!(provider.writes.borrow().is_empty());
    provider.failure.borrow_mut().take();
    adapter.set("new");
    assert_eq!(adapter.issue(), None);
    assert_eq!(adapter.get().as_deref(), Some("new"));
}
