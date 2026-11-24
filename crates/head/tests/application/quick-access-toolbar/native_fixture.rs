// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[given("`fixtures/cad/blank-metric-part.opd` is open and active in the MCAD Software")]
fn open_blank_metric_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/blank-metric-part.opd");
    world.open_fixture(&path);
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    world.previous_document = Some(world.active_id());
}
