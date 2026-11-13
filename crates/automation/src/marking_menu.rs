// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::marking_menu::MarkingMenus;

pub(crate) const METHODS: [&str; 2] = [METHOD_UI_GET_MARKING_MENU, METHOD_UI_SET_MARKING_MENU];
pub(crate) fn apply(
    menus: &mut MarkingMenus,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_UI_GET_MARKING_MENU => {
            let args: GetMarkingMenuArgs = decode(method, request)?;
            encode(&menus.get(args.environment))
        }
        METHOD_UI_SET_MARKING_MENU => {
            let args: SetMarkingMenuArgs = decode(method, request)?;
            menus.set(args.menu).map_err(Error)?;
            encode(&OKResult { ok: true })
        }
        _ => Err(Error(format!(
            "marking menu method {method:?}: expected one of {METHODS:?}"
        ))),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::{
        client::{Caller, Client},
        types,
    };
    use std::cell::RefCell;
    struct MenuCaller(RefCell<MarkingMenus>);
    impl Caller for MenuCaller {
        fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
            apply(&mut self.0.borrow_mut(), method, request)
        }
    }
    #[test]
    fn typed_api_reads_and_replaces_one_environment() {
        let client = Client::new(MenuCaller(RefCell::new(MarkingMenus::default())));
        let args = GetMarkingMenuArgs {
            environment: types::SKETCH_ENVIRONMENT,
        };
        let mut menu = client.ui().marking_menu(&args).unwrap();
        menu.quadrants = None;
        client
            .ui()
            .set_marking_menu(&SetMarkingMenuArgs { menu: menu.clone() })
            .unwrap();
        assert_eq!(client.ui().marking_menu(&args).unwrap(), menu);
        assert!(
            !client
                .ui()
                .marking_menu(&GetMarkingMenuArgs {
                    environment: types::BASE_ENVIRONMENT
                })
                .unwrap()
                .quadrants
                .unwrap()
                .is_empty()
        );
    }
}
