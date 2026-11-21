// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn desktop_starts_with_an_existing_single_key_customization() {
    let store = FakeKeymapStore::default();
    store.saved.lock().unwrap().chords = Some(std::collections::BTreeMap::from([(
        "sketch-line".into(),
        "E".into(),
    )]));
    let (desktop, _) = owner();
    let mut desktop = desktop
        .with_keymap_preferences(Box::new(store.clone()))
        .unwrap();
    let bindings = Client::new(desktop.document_host())
        .keymap()
        .list()
        .unwrap()
        .bindings
        .unwrap();
    let chord = |id: &str| {
        bindings
            .iter()
            .find(|item| item.action_id == id)
            .unwrap()
            .chord
            .as_str()
    };
    assert_eq!(chord("sketch-line"), "E");
    assert_eq!(chord("Create.Extrude"), "");
    assert_eq!(
        store.saved.lock().unwrap().chords.as_ref().unwrap()["sketch-line"],
        "E"
    );
}
