// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::keymap::Keymap;

pub(crate) const METHODS: [&str; 7] = [
    METHOD_KEYMAP_LIST,
    METHOD_KEYMAP_SET_CHORD,
    METHOD_KEYMAP_SET_ALIAS,
    METHOD_KEYMAP_RESET,
    METHOD_KEYMAP_RESET_ALL,
    METHOD_KEYMAP_EXPORT,
    METHOD_KEYMAP_IMPORT,
];

pub(crate) fn apply(map: &mut Keymap, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_KEYMAP_LIST => encode(&ListBindingsResult {
            bindings: Some(map.catalog()),
        }),
        METHOD_KEYMAP_EXPORT => encode(&map.export()),
        METHOD_KEYMAP_SET_CHORD => set_chord(map, method, request),
        METHOD_KEYMAP_SET_ALIAS => set_alias(map, method, request),
        METHOD_KEYMAP_RESET => reset(map, method, request),
        METHOD_KEYMAP_RESET_ALL => {
            map.reset_all();
            success()
        }
        METHOD_KEYMAP_IMPORT => {
            map.restore_saved(&decode(method, request)?)
                .map_err(Error)?;
            success()
        }
        _ => Err(Error(format!(
            "keymap method {method:?}: expected one of {METHODS:?}"
        ))),
    }
}
fn set_chord(map: &mut Keymap, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
    let args: SetChordArgs = decode(method, request)?;
    map.set_chord(&args.action_id, &args.chord).map_err(Error)?;
    success()
}
fn set_alias(map: &mut Keymap, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
    let args: SetAliasArgs = decode(method, request)?;
    map.set_alias(&args.action_id, &args.alias).map_err(Error)?;
    success()
}
fn reset(map: &mut Keymap, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
    let args: ResetBindingArgs = decode(method, request)?;
    map.reset(&args.action_id).map_err(Error)?;
    success()
}
fn success() -> Result<Vec<u8>, Error> {
    encode(&OKResult { ok: true })
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::client::{Caller, Client};
    use oblikovati_model::keymap::KeymapAction;
    use std::cell::RefCell;

    struct KeymapCaller(RefCell<Keymap>);
    impl Caller for KeymapCaller {
        fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
            apply(&mut self.0.borrow_mut(), method, request)
        }
    }
    #[test]
    fn typed_client_reads_and_changes_the_same_catalog() {
        let action = KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[]).unwrap();
        let client = Client::new(KeymapCaller(RefCell::new(
            Keymap::new(vec![action]).unwrap(),
        )));
        client
            .keymap()
            .set_chord(&SetChordArgs {
                action_id: "edit.undo".into(),
                chord: "Ctrl+U".into(),
            })
            .unwrap();
        let bindings = client.keymap().list().unwrap().bindings.unwrap();
        assert_eq!(bindings[0].chord, "Ctrl+U");
        let saved = client.keymap().export().unwrap();
        client.keymap().reset_all().unwrap();
        assert_eq!(
            client.keymap().list().unwrap().bindings.unwrap()[0].chord,
            "Ctrl+Z"
        );
        client.keymap().import(&saved).unwrap();
        assert_eq!(
            client.keymap().list().unwrap().bindings.unwrap()[0].chord,
            "Ctrl+U"
        );
    }
}
