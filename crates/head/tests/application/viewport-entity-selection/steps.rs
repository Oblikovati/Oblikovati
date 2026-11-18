// SPDX-License-Identifier: GPL-2.0-only
//! Viewport selection against the scenario's native Inventor fixture (#1997).
use super::ApplicationWorld;
use cucumber::given;
use oblikovati_api::{client::Client, wire::OpenDocumentArgs};

#[given(
    "the MCAD Software displays fixture \"viewport-selection-solid.ipt\" in the part modeling workspace"
)]
fn open_viewport_selection_fixture(world: &mut ApplicationWorld) {
    let path = super::scenario_root().join("fixtures/cad/viewport-selection-solid.ipt");
    let request = OpenDocumentArgs {
        full_document_name: path.to_string_lossy().into_owned(),
        visible: true,
        ..Default::default()
    };
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&request)
        .unwrap_or_else(|error| {
            panic!("fixture {path:?}: expected an open Inventor part: {error}")
        });
    world.pump(Vec::new());
    assert!(
        world.state().document.is_some(),
        "fixture {path:?}: expected an active part"
    );
}
