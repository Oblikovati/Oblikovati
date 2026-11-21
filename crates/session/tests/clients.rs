// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_session::clients::{ClientRegistrationError, ConnectedClients};

#[test]
fn registration_order_duplicate_names_and_nonreused_ids_match_go() {
    let mut clients = ConnectedClients::default();
    assert!(clients.entries().is_empty());
    let first = clients.register("cam").unwrap();
    let second = clients.register("cam").unwrap();
    assert_eq!((first, second), (1, 2));
    let owned = clients.entries().to_vec();
    assert!(clients.contains_name("cam"));
    assert!(!clients.contains_name("CAM"));
    clients.unregister(first).unwrap();
    assert!(clients.contains_name("cam"));
    assert_eq!(owned.len(), 2);
    assert_eq!(clients.register("fea").unwrap(), 3);
    assert_eq!(
        clients
            .entries()
            .iter()
            .map(|entry| (entry.id, entry.name.as_str()))
            .collect::<Vec<_>>(),
        [(2, "cam"), (3, "fea")]
    );
    clients.unregister(second).unwrap();
    assert!(!clients.contains_name("cam"));
}

#[test]
fn invalid_registration_and_removal_preserve_entries_and_id_sequence() {
    let mut clients = ConnectedClients::default();
    let error = clients.register("").unwrap_err();
    assert_eq!(error, ClientRegistrationError::EmptyName);
    assert!(error.to_string().contains("nonempty"));
    assert_eq!(clients.register(" ").unwrap(), 1);
    for id in [-1, 0, 99] {
        let error = clients.unregister(id).unwrap_err();
        assert_eq!(error, ClientRegistrationError::Unknown(id));
        assert!(error.to_string().contains(&id.to_string()));
        assert_eq!(clients.entries().len(), 1);
    }
    assert_eq!(clients.register("next").unwrap(), 2);
}
