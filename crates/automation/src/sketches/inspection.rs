// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::{PlacedSketch, PlacedSketchHealth, SketchEvaluation};

pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<Vec<u8>, Error> {
    let document = owner::active_sketch_host(files)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    if method == METHOD_SKETCH_GET {
        let args: SketchArgs = crate::decode(method, request)?;
        let id = sketch_at(session.snapshot(), args.sketch_index)?;
        return encode(&inspect(session, document, args.sketch_index, id, editing)?);
    }
    encode(&ListSketchesResult {
        sketches: Some(list_sketches(session, document, editing)?),
    })
}

fn list_sketches<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    document: u64,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<Vec<SketchInfo>, Error> {
    session
        .snapshot()
        .sketches
        .sketches()
        .iter()
        .enumerate()
        .map(|(index, sketch)| inspect(session, document, wire_count(index)?, sketch.id(), editing))
        .collect()
}

fn inspect<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    document: u64,
    index: i64,
    id: DocumentSketchId,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<SketchInfo, Error> {
    let snapshot = session.snapshot();
    let sketch = snapshot.sketches.get(id).map_err(api_error)?;
    let structure = session.analyze_sketch(id).map_err(api_error)?;
    let mut summary = sketch_summary(snapshot, sketch, index)?;
    summary.dof = wire_count(structure.degrees_of_freedom)?;
    summary.editing = editing == Some((document, id));
    summary.healthy = current_health(snapshot, id)?;
    Ok(summary)
}

fn sketch_summary(
    snapshot: &DocumentSnapshot,
    sketch: &PlacedSketch,
    index: i64,
) -> Result<SketchInfo, Error> {
    let plane = snapshot
        .resolve_sketch_plane(sketch.id())
        .map_err(api_error)?;
    let owned_by = consuming_feature(snapshot, sketch.id());
    Ok(SketchInfo {
        index,
        name: sketch.name().into(),
        plane: plane_label(plane.normal()).into(),
        // All authored sketches are displayed; visibility/share/style edits are not yet exposed (#3563).
        visible: true,
        entity_count: wire_count(sketch.geometry().entity_ids().len())?,
        consumed: owned_by.is_some(),
        owned_by: owned_by.unwrap_or_default().into(),
        ..Default::default()
    })
}

fn consuming_feature(snapshot: &DocumentSnapshot, sketch: DocumentSketchId) -> Option<&str> {
    snapshot
        .work_geometry
        .sketch_consumers(sketch)
        .next()
        .map(|feature| feature.name())
}

fn current_health(snapshot: &DocumentSnapshot, id: DocumentSketchId) -> Result<bool, Error> {
    let health = snapshot.document_sketch_health(id).map_err(api_error)?;
    Ok(
        matches!(health, PlacedSketchHealth::Current(SketchEvaluation::Numerical(report))
        if report.termination == SolveTermination::Converged && report.structure.status != ConstraintStatus::OverConstrained),
    )
}

fn plane_label(normal: [f64; 3]) -> &'static str {
    match normal.map(f64::abs) {
        [0.0, 0.0, 1.0] => "XY",
        [0.0, 1.0, 0.0] => "XZ",
        [1.0, 0.0, 0.0] => "YZ",
        _ => "custom",
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::feature::part::PartFeatureDefinition;
    use oblikovati_model::{
        feature::{
            AuthoredFeature, FeatureId, FeatureReadFootprint,
            sketch_prism::{SketchDirectedLine, SketchPrismDefinition, SketchPrismSelection},
            work_geometry::DatumLength,
        },
        sketch::SketchEntityId,
    };

    #[test]
    fn plane_names_accept_reversed_normals_and_keep_oblique_planes_custom() {
        for (normal, name) in [
            ([0., 0., -1.], "XY"),
            ([0., -1., 0.], "XZ"),
            ([-1., 0., 0.], "YZ"),
            ([0.6, 0.8, 0.], "custom"),
        ] {
            assert_eq!(plane_label(normal), name);
        }
    }

    #[test]
    fn consumer_names_come_from_authored_feature_incidence_in_program_order() {
        let mut snapshot = DocumentSnapshot {
            work_geometry: Default::default(),
            sketches: Default::default(),
            attributes: Default::default(),
            attachments: Default::default(),
            interests: Default::default(),
            resources: Default::default(),
            assets: Default::default(),
            body_names: Default::default(),
            sketch_settings: None,
            parameters: oblikovati_model::parameter::ParameterCollection::new(Default::default()),
            properties: Default::default(),
        };
        assert_eq!(consuming_feature(&snapshot, DocumentSketchId(1)), None);
        for (id, name) in [(1, "First extrusion"), (2, "Second extrusion")] {
            let definition = consumer_definition(DocumentSketchId(1));
            snapshot
                .work_geometry
                .append(AuthoredFeature::new(
                    FeatureId(id),
                    name,
                    PartFeatureDefinition::SketchPrism(Box::new(definition)),
                    FeatureReadFootprint::new(vec![], false),
                ))
                .unwrap();
        }
        assert_eq!(
            consuming_feature(&snapshot, DocumentSketchId(1)),
            Some("First extrusion")
        );
        assert_eq!(
            snapshot
                .work_geometry
                .sketch_consumers(DocumentSketchId(1))
                .map(|feature| feature.id().0)
                .collect::<Vec<_>>(),
            [1, 2]
        );
        assert_eq!(consuming_feature(&snapshot, DocumentSketchId(2)), None);
    }

    fn consumer_definition(sketch: DocumentSketchId) -> SketchPrismDefinition {
        let outer = (1..=3)
            .map(|id| SketchDirectedLine {
                line: SketchEntityId(id),
                reversed: false,
            })
            .collect();
        SketchPrismDefinition::new(
            sketch,
            SketchPrismSelection {
                outer,
                holes: vec![],
            },
            [
                DatumLength::Constant(0.),
                DatumLength::Constant(0.),
                DatumLength::Constant(1.),
            ],
        )
        .unwrap()
    }
}
