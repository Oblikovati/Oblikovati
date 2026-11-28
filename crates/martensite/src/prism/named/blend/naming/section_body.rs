// SPDX-License-Identifier: GPL-2.0-only
use super::{
    result_names::{incident_name, require_unique},
    *,
};

pub(super) fn bind_section_body(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    body: Vec<u8>,
    shape: Shape,
    resolution: PrismResolution,
    achieved_tolerance: Option<PrismAchievedTolerance>,
    subentities: Vec<(Vec<u8>, Shape)>,
) -> Result<BlendNamedTopology, String> {
    let entries: Vec<_> = std::iter::once((body.clone(), shape.clone()))
        .chain(subentities)
        .collect();
    let bindings = ExactReferenceBindings::from_entries(
        document,
        history.evaluation_scope(),
        entries.clone(),
    )?;
    verify_round_trip(&bindings, history, document, &entries)?;
    let live = bindings
        .encoded_keys(history, document)?
        .map(<[u8]>::to_vec)
        .collect();
    Ok(BlendNamedTopology {
        resolution: Some(resolution),
        achieved_tolerance,
        live,
        deleted: Vec::new(),
        splits: BTreeMap::new(),
        bindings,
        active_body: body.clone(),
        primary_body: body,
        shape,
        thread_overlays: Vec::new(),
    })
}

pub(super) fn incident_names(
    document: DocumentGuid,
    feature: FeatureId,
    solid: &Shape,
    kind: ShapeEnum,
    reference_kind: u32,
    parents: &[(Vec<u8>, Shape)],
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let names = shapes(solid, kind)?
        .into_iter()
        .map(|shape| {
            let name = incident_name(document, feature, reference_kind, kind, &shape, parents)?;
            Ok((name, shape))
        })
        .collect::<Result<Vec<_>, String>>()?;
    require_unique(&names)?;
    Ok(names)
}
pub(super) fn name_open_shell_edges(
    shell: &Shape,
    trims: [(Vec<u8>, Shape); 2],
    name: &impl Fn(u32, &str) -> Result<Vec<u8>, String>,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    let mut other = shapes(shell, ShapeEnum::Edge)?
        .into_iter()
        .filter(|edge| trims.iter().all(|(_, trim)| !edge.is_same(trim)));
    let seam = other.next();
    if other.next().is_some() {
        return Err("section shell has more than one unnamed seam".into());
    }
    let seam = seam
        .map(|shape| Ok::<_, String>((name(2, "seam")?, shape)))
        .transpose()?;
    Ok(trims.into_iter().chain(seam).collect())
}
