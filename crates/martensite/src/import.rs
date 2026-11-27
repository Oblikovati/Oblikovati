// SPDX-License-Identifier: GPL-2.0-only
//! Certified STL and STEP exchange inspection and Martensite shape import.
use crate::{
    BlendNamedTopology, CertifiedNamedSolidEvaluator, EvaluationEntity, EvaluationFailure,
    EvaluationFailureKind, EvaluationPhase, NamedSolid,
};
use martensite_tkbrep::topexp::Explorer;
use martensite_tkbrep::topods::Shape;
use martensite_tkdestep::step_control::StepControlReader;
use martensite_tkdestl::{rw_stl::RwStl, stl_api::StlApiReader};
use martensite_tkernel::{message::ProgressRange, tcollection::AsciiString};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Point3, Transform3};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIMakeShape, BRepBuilderAPITransform};
use martensite_tkxsbase::{if_select::IFSelectReturnStatus, xs_control::XSControlReaderOps};
use oblikovati_model::feature::{
    FeatureId,
    imported_model::{ImportedFileKind, ResolvedImportedModel},
    part::PartFeatureFailure,
};
use std::path::Path;

const SCALE_EPSILON: f64 = 1e-12;

/// Summary extracted from an STL file before or during import.
#[derive(Clone, Debug, PartialEq)]
pub struct StlInspection {
    pub solid_name: String,
    pub triangle_count: usize,
    pub bounding_box: [f64; 3],
}

/// Summary extracted from a STEP file before or during import.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct StepInspection {
    pub solid_names: Vec<String>,
    pub solid_count: usize,
}

/// Inspect an ASCII or binary STL file via Martensite `RwStl`.
pub fn inspect_stl(path: &Path) -> Result<StlInspection, String> {
    let path_str = path
        .to_str()
        .ok_or_else(|| format!("STL path {path:?}: expected valid UTF-8"))?;
    let mesh = RwStl::read_file(&AsciiString::from(path_str), &ProgressRange::default())
        .map_err(|cause| format!("STL read {path_str:?}: {cause}"))?;
    let Some(triangulation) = mesh.get() else {
        return Err(format!("STL read {path_str:?}: expected non-empty mesh"));
    };
    let nodes = triangulation.nb_nodes();
    let triangles = triangulation.nb_triangles();
    if nodes <= 0 || triangles <= 0 {
        return Err(format!(
            "STL read {path_str:?}: expected triangles (found {triangles})"
        ));
    }
    let mut min = [f64::INFINITY; 3];
    let mut max = [f64::NEG_INFINITY; 3];
    for idx in 1..=nodes {
        let p = triangulation
            .node(idx)
            .map_err(|cause| format!("STL node {idx}: {cause}"))?;
        let coords = [p.x(), p.y(), p.z()];
        for axis in 0..3 {
            min[axis] = min[axis].min(coords[axis]);
            max[axis] = max[axis].max(coords[axis]);
        }
    }
    let bounding_box = std::array::from_fn(|axis| (max[axis] - min[axis]).max(0.0));
    let solid_name = parse_ascii_stl_name(path).unwrap_or_else(|| {
        path.file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or("ImportedSTL")
            .to_owned()
    });
    Ok(StlInspection {
        solid_name,
        triangle_count: usize::try_from(triangles).unwrap_or(0),
        bounding_box,
    })
}

/// Inspect a STEP file via Martensite `StepControlReader` and extract its solid names.
pub fn inspect_step(path: &Path) -> Result<StepInspection, String> {
    let shape = read_step_shape(path)?;
    let mut count = 0usize;
    let mut explorer = Explorer::new(&shape, ShapeEnum::Solid, ShapeEnum::Shape)
        .map_err(|cause| format!("STEP solid traversal: {cause}"))?;
    while explorer.more() {
        count += 1;
        explorer
            .next()
            .map_err(|cause| format!("STEP solid traversal: {cause}"))?;
    }
    if count == 0 {
        return Err(format!(
            "STEP read {:?}: expected at least one solid body",
            path
        ));
    }
    let mut names = parse_step_solid_names(path);
    while names.len() < count {
        names.push(format!("Solid{}", names.len() + 1));
    }
    names.truncate(count);
    Ok(StepInspection {
        solid_names: names,
        solid_count: count,
    })
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(crate) fn evaluate_named_imported_model(
        &mut self,
        feature: FeatureId,
        input: ResolvedImportedModel,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let failure = |kind: EvaluationFailureKind, message: String| {
            PartFeatureFailure::Kernel(
                EvaluationPhase::Prism
                    .refused(kind, EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let path = Path::new(&input.source_path);
        let raw = match input.file_kind {
            ImportedFileKind::Stl => read_stl_shape(path)
                .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?,
            ImportedFileKind::Step => read_step_shape(path)
                .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?,
        };
        let shape = scale_imported_shape(&raw, input.unit_scale)
            .map_err(|msg| failure(EvaluationFailureKind::InvalidResult, msg))?;
        let topology =
            BlendNamedTopology::from_derived(self.history, self.document, feature, shape, None)
                .map_err(|cause| {
                    failure(
                        EvaluationFailureKind::InvalidReference,
                        format!("import topology: {cause}"),
                    )
                })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }
}

fn read_stl_shape(path: &Path) -> Result<Shape, String> {
    let path_str = path
        .to_str()
        .ok_or_else(|| format!("STL path {path:?}: expected valid UTF-8"))?;
    let mut shape = Shape::default();
    let ok = StlApiReader
        .read_file(&mut shape, &AsciiString::from(path_str))
        .map_err(|cause| format!("STL read {path_str:?}: {cause}"))?;
    if !ok || shape.is_null() {
        return Err(format!("STL read {path_str:?}: produced an empty shape"));
    }
    Ok(shape)
}

fn read_step_shape(path: &Path) -> Result<Shape, String> {
    let path_str = path
        .to_str()
        .ok_or_else(|| format!("STEP path {path:?}: expected valid UTF-8"))?;
    let mut reader =
        StepControlReader::new().map_err(|cause| format!("STEP reader init: {cause}"))?;
    let status = reader
        .read_file(path_str)
        .map_err(|cause| format!("STEP read {path_str:?}: {cause}"))?;
    if status != IFSelectReturnStatus::RetDone {
        return Err(format!("STEP read {path_str:?}: returned {status:?}"));
    }
    reader
        .transfer_roots(&ProgressRange::default())
        .map_err(|cause| format!("STEP transfer {path_str:?}: {cause}"))?;
    let shape = reader.xs_reader().one_shape();
    if shape.is_null() {
        return Err(format!("STEP transfer {path_str:?}: produced no shape"));
    }
    Ok(shape)
}

fn scale_imported_shape(shape: &Shape, scale: f64) -> Result<Shape, String> {
    if (scale - 1.0).abs() <= SCALE_EPSILON {
        return Ok(shape.clone());
    }
    let mut trsf = Transform3::default();
    trsf.set_scale(Point3::default(), scale)
        .map_err(|cause| cause.to_string())?;
    let mut scaled = BRepBuilderAPITransform::with_shape(shape, trsf, true, false)
        .map_err(|cause| cause.to_string())?;
    Ok(scaled.shape().map_err(|cause| cause.to_string())?.clone())
}

fn parse_ascii_stl_name(path: &Path) -> Option<String> {
    let content = std::fs::read_to_string(path).ok()?;
    let first = content.lines().next()?.trim();
    let rest = first.strip_prefix("solid")?.trim();
    (!rest.is_empty()).then(|| rest.to_owned())
}

fn parse_step_solid_names(path: &Path) -> Vec<String> {
    let Ok(content) = std::fs::read_to_string(path) else {
        return Vec::new();
    };
    let mut names = Vec::new();
    let needle = "MANIFOLD_SOLID_BREP(";
    let mut rest = content.as_str();
    while let Some(pos) = rest.find(needle) {
        let after = &rest[pos + needle.len()..];
        if let Some(after_quote) = after.trim_start().strip_prefix('\'')
            && let Some(end_quote) = after_quote.find('\'')
        {
            let name = after_quote[..end_quote].trim();
            if !name.is_empty() {
                names.push(name.to_owned());
            }
        }
        rest = after;
    }
    names
}
