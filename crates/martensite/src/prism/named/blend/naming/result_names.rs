// SPDX-License-Identifier: GPL-2.0-only
//! Distinguish the descendants of a split by exact topology history.
use super::*;

type Entry = (Vec<u8>, Shape);
type Entries = Vec<Entry>;

pub(super) struct ResultNames {
    pub(super) entries: Entries,
    pub(super) splits: SplitTopologyNames,
}

impl BlendCandidate {
    pub(super) fn result_names(
        &self,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<ResultNames, String> {
        let mut entries = Vec::new();
        let mut upper = Vec::new();
        let mut splits = SplitTopologyNames::new();
        for (kind, shape_kind) in [
            (1, ShapeEnum::Face),
            (2, ShapeEnum::Edge),
            (3, ShapeEnum::Vertex),
        ] {
            let names = self.names_of_kind(document, feature, kind, shape_kind, &upper)?;
            upper = split_names(
                names,
                &self.history,
                &upper,
                document,
                feature,
                kind,
                &mut splits,
            )?;
            entries.extend(upper.iter().cloned());
        }
        Ok(ResultNames { entries, splits })
    }

    fn names_of_kind(
        &self,
        document: DocumentGuid,
        feature: FeatureId,
        kind: u32,
        shape_kind: ShapeEnum,
        upper: &[Entry],
    ) -> Result<Entries, String> {
        shapes(&self.shape, shape_kind)?
            .into_iter()
            .map(|shape| {
                let name = match direct_name(&self.history, &shape, kind, document, feature)? {
                    Some(key) => key,
                    None if kind != 1 => {
                        incident_name(document, feature, kind, shape_kind, &shape, upper)?
                    }
                    None => return Err("blend face: expected kernel history".into()),
                };
                Ok((name, shape))
            })
            .collect()
    }
}

fn split_names(
    names: Entries,
    history: &[BlendHistoryRecord],
    upper: &[Entry],
    document: DocumentGuid,
    feature: FeatureId,
    kind: u32,
    splits: &mut SplitTopologyNames,
) -> Result<Entries, String> {
    let mut grouped = BTreeMap::<Vec<u8>, Vec<Shape>>::new();
    for (name, shape) in names {
        grouped.entry(name).or_default().push(shape);
    }
    let mut entries = Vec::new();
    for (parent, descendants) in grouped {
        if let [shape] = descendants.as_slice() {
            entries.push((parent, shape.clone()));
            continue;
        }
        let witnesses = descendants
            .iter()
            .map(|shape| incident_sources(shape, history, upper, kind))
            .collect::<Result<Vec<_>, _>>()?;
        let mut child_names = encode_split_names(&parent, &witnesses, document, feature, kind)?;
        entries.extend(child_names.iter().cloned().zip(descendants));
        child_names.sort();
        splits.insert(parent, child_names);
    }
    require_unique(&entries)?;
    Ok(entries)
}

fn encode_split_names(
    parent: &[u8],
    witnesses: &[BTreeSet<Vec<u8>>],
    document: DocumentGuid,
    feature: FeatureId,
    kind: u32,
) -> Result<Vec<Vec<u8>>, String> {
    let encode = |signature: &BTreeSet<Vec<u8>>| {
        let mut parents = signature.clone();
        parents.insert(parent.to_vec());
        let parents: Vec<_> = parents.into_iter().collect();
        DerivedReferenceName::new(document, feature.0, kind, "split", &parents)?.encode()
    };
    match witnesses.iter().map(encode).collect::<Result<_, _>>() {
        Ok(names) => Ok(names),
        Err(limit) => {
            // Use a bounded exact signature when the complete ancestry does
            // not fit. A witness is useful only if it separates siblings.
            let signatures = distinguishing_signatures(parent, witnesses).ok_or(limit)?;
            signatures.iter().map(encode).collect()
        }
    }
}

fn distinguishing_signatures(
    parent: &[u8],
    witnesses: &[BTreeSet<Vec<u8>>],
) -> Option<Vec<BTreeSet<Vec<u8>>>> {
    let candidates: BTreeSet<_> = witnesses
        .iter()
        .flatten()
        .filter(|key| key.as_slice() != parent)
        .map(|key| (key.len(), key))
        .collect();
    let mut unresolved: Vec<_> = (0..witnesses.len())
        .flat_map(|first| ((first + 1)..witnesses.len()).map(move |second| (first, second)))
        .collect();
    let mut signatures = vec![BTreeSet::new(); witnesses.len()];
    for (_, key) in candidates {
        let remaining = unresolved.len();
        unresolved.retain(|&(first, second)| {
            witnesses[first].contains(key) == witnesses[second].contains(key)
        });
        if unresolved.len() != remaining {
            for (witnesses, signature) in witnesses.iter().zip(&mut signatures) {
                if witnesses.contains(key) {
                    signature.insert(key.clone());
                }
            }
        }
        if unresolved.is_empty() {
            return Some(signatures);
        }
    }
    None
}

fn incident_sources(
    shape: &Shape,
    history: &[BlendHistoryRecord],
    upper: &[Entry],
    kind: u32,
) -> Result<BTreeSet<Vec<u8>>, String> {
    let mut entities = vec![shape.clone()];
    if kind == 1 {
        entities.extend(shapes(shape, ShapeEnum::Edge)?);
    }
    if kind <= 2 {
        entities.extend(shapes(shape, ShapeEnum::Vertex)?);
    }
    let mut parents: BTreeSet<_> = history
        .iter()
        .filter(|record| entities.iter().any(|entity| contributes(record, entity)))
        .map(|record| record.name.clone())
        .collect();
    let shape_kind = if kind == 2 {
        ShapeEnum::Edge
    } else {
        ShapeEnum::Vertex
    };
    for (name, incident) in upper {
        if shapes(incident, shape_kind)?
            .iter()
            .any(|item| item.is_same(shape))
        {
            parents.insert(name.clone());
        }
    }
    Ok(parents)
}

fn contributes(record: &BlendHistoryRecord, entity: &Shape) -> bool {
    record.source.is_same(entity)
        || record
            .modified
            .iter()
            .chain(&record.generated)
            .any(|child| child.is_same(entity))
}

pub(super) fn incident_name(
    document: DocumentGuid,
    feature: FeatureId,
    target_kind: u32,
    kind: ShapeEnum,
    shape: &Shape,
    upper: &[(Vec<u8>, Shape)],
) -> Result<Vec<u8>, String> {
    let mut parents = Vec::new();
    for (name, entity) in upper {
        if shapes(entity, kind)?
            .iter()
            .any(|candidate| candidate.is_same(shape))
        {
            parents.push(name.clone());
        }
    }
    if parents.is_empty() {
        return Err(format!(
            "blend result kind {target_kind}: expected named incident entities"
        ));
    }
    DerivedReferenceName::new(document, feature.0, target_kind, "incidence", &parents)?.encode()
}

pub(super) fn require_unique(names: &[(Vec<u8>, Shape)]) -> Result<(), String> {
    let mut seen = BTreeMap::new();
    for (name, shape) in names {
        if seen.insert(name, shape).is_some() {
            return Err(format!("blend name {name:?}: expected one result entity"));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
    use oblikovati_model::identity::SemanticReferenceName;

    #[test]
    fn split_descendants_use_exact_boundary_history_independent_of_result_order() {
        let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let name = |id, kind| {
            SemanticReferenceName::new(document, id, kind, "fixture", "source", &[1])
                .unwrap()
                .encode()
                .unwrap()
        };
        let shape = martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox::from_sizes(1., 1., 1.)
            .unwrap()
            .shape()
            .unwrap()
            .clone();
        let faces = shapes(&shape, ShapeEnum::Face).unwrap();
        let entries = vec![
            (name(1, 1), faces[0].clone()),
            (name(1, 1), faces[1].clone()),
        ];
        let other_edges = shapes(&faces[1], ShapeEnum::Edge).unwrap();
        let edge = shapes(&faces[0], ShapeEnum::Edge)
            .unwrap()
            .into_iter()
            .find(|edge| !other_edges.iter().any(|other| edge.is_same(other)))
            .unwrap();
        let witness = BlendHistoryRecord {
            name: name(2, 2),
            source: edge,
            generated: Vec::new(),
            modified: Vec::new(),
            deleted: false,
            transient: true,
        };
        let feature = FeatureId(3);
        let mut first = SplitTopologyNames::new();
        let bound = split_names(
            entries.clone(),
            std::slice::from_ref(&witness),
            &[],
            document,
            feature,
            1,
            &mut first,
        )
        .unwrap();
        let mut reversed = entries;
        reversed.reverse();
        let mut second = SplitTopologyNames::new();
        let rebound = split_names(
            reversed.clone(),
            &[witness],
            &[],
            document,
            feature,
            1,
            &mut second,
        )
        .unwrap();
        assert_eq!(first, second);
        assert_eq!(first[&name(1, 1)].len(), 2);
        for (key, face) in bound {
            assert!(
                rebound
                    .iter()
                    .any(|(other, other_face)| other == &key && other_face.is_same(&face))
            );
            assert_ne!(key, name(1, 1));
        }
        let ambiguous = split_names(
            reversed,
            &[],
            &[],
            document,
            feature,
            1,
            &mut SplitTopologyNames::new(),
        );
        assert!(ambiguous.is_err());
    }
    #[test]
    fn oversized_split_ancestry_uses_a_stable_distinguishing_witness() {
        let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let name = |id, kind| {
            SemanticReferenceName::new(document, id, kind, "fixture", "source", &[1])
                .unwrap()
                .encode()
                .unwrap()
        };
        let first: BTreeSet<_> = (2..22).map(|id| name(id, 2)).collect();
        let mut second = first.clone();
        second.insert(name(22, 2));
        let parent = name(1, 1);
        let feature = FeatureId(23);
        let keys = encode_split_names(
            &parent,
            &[first.clone(), second.clone()],
            document,
            feature,
            1,
        )
        .unwrap();
        let reversed =
            encode_split_names(&parent, &[second, first.clone()], document, feature, 1).unwrap();
        assert_eq!(keys[0], reversed[1]);
        assert_eq!(keys[1], reversed[0]);
        assert_ne!(keys[0], keys[1]);
        assert!(
            encode_split_names(&parent, &[first.clone(), first], document, feature, 1).is_err()
        );
    }
}
