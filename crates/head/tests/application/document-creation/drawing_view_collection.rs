// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    client::{Caller, Client},
    wire::{AddBaseViewArgs, DrawingViewInfo},
};

pub(super) fn verify(client: &Client<impl Caller>, original: DrawingViewInfo) {
    let another = client
        .drawing_views()
        .add_base(&AddBaseViewArgs {
            name: "Top".into(),
            orientation: "top".into(),
            scale: 0.5,
            center_xmm: 180.,
            center_ymm: 80.,
            ..Default::default()
        })
        .unwrap()
        .view;
    let list = || {
        client
            .drawing_views()
            .list(&Default::default())
            .unwrap()
            .views
            .unwrap_or_default()
    };
    assert_eq!(list(), vec![original, another.clone()]);
    let remove = |name: &str| {
        client
            .drawing_views()
            .delete(&oblikovati_api::wire::DeleteViewArgs { name: name.into() })
    };
    assert!(remove("Missing").is_err());
    assert_eq!(remove("VIEW:1").unwrap().views, Some(vec![another.clone()]));
    client.transactions().undo().unwrap();
    assert_eq!(list().len(), 2);
    client.transactions().redo().unwrap();
    assert_eq!(list(), vec![another]);
    assert_eq!(remove("Top").unwrap().views, None);
}
