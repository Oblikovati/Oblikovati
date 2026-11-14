// SPDX-License-Identifier: GPL-2.0-only
//! Real parameter-driven part and a drawing that references it.
#[path = "../../../../test-utilities/rust/selected_sketch_prism.rs"]
#[allow(dead_code)]
mod prism;
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
use oblikovati_persistence::package::DocumentPackage;
use std::path::{Path, PathBuf};

pub fn saved_drawing(folder: &Path) -> PathBuf {
    let source = folder.join("model.opd");
    let snapshot = prism::selected_prism_document();
    let part = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap();
    std::fs::write(&source, part.encode().unwrap()).unwrap();
    let mut drawing = DrawingDocument::new();
    drawing.set_model_reference("model.opd").unwrap();
    drawing
        .add_base_view_definition(
            0,
            DrawingBaseViewDefinition::new(
                "Front",
                oblikovati_api::types::BASE_VIEW_FRONT,
                1.,
                [210., 148.5],
            )
            .unwrap(),
        )
        .unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap();
    let file = folder.join("drawing.odd");
    std::fs::write(&file, package.encode().unwrap()).unwrap();
    file
}

pub fn assert_saved_view(path: &Path) {
    let package = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let saved = package
        .restore_saved_drawing_views()
        .unwrap()
        .expect("expected saved view geometry");
    assert_eq!(saved.views.len(), 1);
    assert!(!saved.views[0].lines.is_empty(), "expected projected lines");
}
