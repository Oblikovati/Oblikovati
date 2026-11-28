// SPDX-License-Identifier: GPL-2.0-only
//! Built-in parametric sample part and empty part packages for browser sessions.

use oblikovati_api::types::{DOCUMENT_PART, USER_PARAM};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    feature::{
        AuthoredFeature, FeatureId,
        sketch_prism::{SketchDirectedCurve, SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    },
    history::{DocumentSnapshot, DocumentSnapshotTarget, SketchReservationMarks},
    identity::{DocumentSaveTarget, FileIdentity, SystemGuidSource},
    parameter::ParameterCollection,
    sketch::{
        DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
        SketchPlane,
    },
    units::UnitsOfMeasure,
};
use oblikovati_persistence::{
    file_access::MemoryDocumentFiles,
    package::{DocumentPackage, ParameterPackageTarget},
};
use std::path::Path;

const SAMPLE_SKETCH: DocumentSketchId = DocumentSketchId(1);

/// Create a valid empty Part package for the browser's New Part action.
/// Example: `let package = empty_part_package("Part2");`.
pub fn empty_part_package(name: &str) -> DocumentPackage {
    DocumentPackage::empty(DOCUMENT_PART, name)
}

/// Build a fully constrained parametric rectangular prism Part package ready for WebGPU display.
/// Example: `let package = sample_prism_package("SampleBlock")?;`.
pub fn sample_prism_package(name: &str) -> Result<DocumentPackage, String> {
    let snapshot = sample_prism_snapshot()?;
    let mut package = DocumentPackage::empty(DOCUMENT_PART, name);
    let mut files = MemoryDocumentFiles::default();
    let path = Path::new("/web/sample.opd");
    let mut guids = SystemGuidSource;
    let identity = FileIdentity::new("0.1.0-web", &mut guids).map_err(|err| err.to_string())?;
    let mut target = ParameterPackageTarget::new(&mut package, path, &mut files)
        .with_display_name(name.to_owned());
    let marks = SketchReservationMarks {
        sketch: 1,
        entity: 104,
        constraints: std::collections::BTreeMap::from([(SAMPLE_SKETCH.0, 7)]),
        dimension_names: Default::default(),
    };
    let digest = target
        .prepare(&snapshot, 0, 3, marks, 1)
        .map_err(|err| format!("sample part prepare: {err}"))?;
    let saved_identity = identity
        .prepare_save(digest, "0.1.0-web", &mut guids)
        .map_err(|err| format!("sample part identity: {err}"))?;
    target
        .persist(&saved_identity)
        .map_err(|err| format!("sample part persist: {err}"))?;
    let bytes = files
        .get(path)
        .ok_or_else(|| "sample part encode: expected persisted bytes".to_owned())?;
    DocumentPackage::decode(&bytes).map_err(|err| format!("sample part decode: {err}"))
}

fn sample_prism_snapshot() -> Result<DocumentSnapshot, String> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "6 cm")
        .map_err(|err| err.to_string())?;
    parameters
        .insert_expression(2, "height", USER_PARAM, "4 cm")
        .map_err(|err| err.to_string())?;
    parameters
        .insert_expression(3, "depth", USER_PARAM, "2.5 cm")
        .map_err(|err| err.to_string())?;

    let mut sketches = DocumentSketches::default();
    sketches
        .add(
            SAMPLE_SKETCH,
            "BaseProfile",
            SketchPlane::xy(),
            rectangle_sketch()?,
        )
        .map_err(|err| err.to_string())?;
    let solved = sketches
        .solve_pending(&parameters, Default::default(), || false)
        .candidate;

    let mut document = DocumentSnapshot {
        sketches: solved,
        parameters,
        work_geometry: Default::default(),
        assets: Default::default(),
        assembly: None,
        drawing: None,
        model_states: None,
        body_names: Default::default(),
        design_views: Default::default(),
        part_design_views: None,
        sketch_settings: None,
        properties: Default::default(),
        attributes: Default::default(),
        attachments: Default::default(),
        interests: Default::default(),
        resources: Default::default(),
    };
    let outer = (101..=104)
        .map(|id| SketchDirectedCurve {
            curve: SketchEntityId(id),
            reversed: false,
        })
        .collect();
    let definition = SketchPrismDefinition::new(
        SAMPLE_SKETCH,
        SketchPrismSelection {
            outer,
            holes: Vec::new(),
        },
        [
            DatumLength::Constant(0.0),
            DatumLength::Constant(0.0),
            DatumLength::Parameter(3),
        ],
    )
    .map_err(|err| err.to_string())?;
    let feature = AuthoredFeature::sketch_prism(FeatureId(1), "Extrusion1", definition, &document)
        .map_err(|err| err.to_string())?;
    document
        .work_geometry
        .append(feature)
        .map_err(|err| err.to_string())?;
    Ok(document)
}

fn rectangle_sketch() -> Result<Sketch2D, String> {
    let mut geometry = Sketch2D::default();
    let corners = [[0.0, 0.0], [6.0, 0.0], [6.0, 4.0], [0.0, 4.0]];
    for (index, position) in corners.into_iter().enumerate() {
        geometry
            .add_point(SketchEntityId(index as u64 + 1), position)
            .map_err(|err| err.to_string())?;
    }
    for index in 0..4u64 {
        let endpoints = [index + 1, (index + 1) % 4 + 1].map(SketchEntityId);
        geometry
            .add_line(SketchEntityId(101 + index), endpoints)
            .map_err(|err| err.to_string())?;
    }
    let ids = [1, 2, 3, 4].map(SketchEntityId);
    let constraints = [
        PlanarConstraint::Fix {
            point: ids[0],
            position: [0.0, 0.0],
        },
        PlanarConstraint::Horizontal {
            points: [ids[0], ids[1]],
        },
        PlanarConstraint::Vertical {
            points: [ids[1], ids[2]],
        },
        PlanarConstraint::Vertical {
            points: [ids[0], ids[3]],
        },
        PlanarConstraint::Horizontal {
            points: [ids[2], ids[3]],
        },
        PlanarConstraint::Distance {
            points: [ids[0], ids[1]],
            parameter: 1,
            measure: DistanceMeasure::Horizontal,
        },
        PlanarConstraint::Distance {
            points: [ids[1], ids[2]],
            parameter: 2,
            measure: DistanceMeasure::Vertical,
        },
    ];
    for (index, constraint) in constraints.into_iter().enumerate() {
        geometry
            .add_constraint(SketchConstraintId(index as u64 + 1), constraint)
            .map_err(|err| err.to_string())?;
    }
    Ok(geometry)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn sample_and_empty_packages_decode_cleanly() {
        let empty = empty_part_package("Part1");
        assert_eq!(empty.manifest().display_name, "Part1");

        let sample = sample_prism_package("SampleBlock").unwrap();
        assert_eq!(sample.manifest().display_name, "SampleBlock");
        let restored = sample.restore_document_parameters().unwrap();
        assert_eq!(restored.parameters.ids().len(), 3);
    }
}
