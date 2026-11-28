// SPDX-License-Identifier: GPL-2.0-only
//! Detached B-rep surface trimming against a cutting plane or surface face (#3918).
use super::{
    split_tool::{body_bounds_center_and_extent, build_plane_tool},
    *,
};
use martensite_tkbo::brep_algo_api::BRepAlgoAPISplitter;
use martensite_tkbrep::{
    topexp,
    topods::{Builder, Shape, Shell},
    toptools::IndexedMapOfShape,
};
use martensite_tkernel::ncollection::List;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::{brep_check::BRepCheckAnalyzer, brep_gprop::BRepGProp};
use oblikovati_model::feature::{
    FeatureId,
    surface_trim::{ResolvedSurfaceTrim, ResolvedSurfaceTrimTool, SurfaceTrimSide},
};

/// Tessellated candidate halves and cutting plane frame for interactive Trim Surface preview.
#[derive(Clone, Debug)]
pub struct SurfaceTrimRegionPreview {
    pub positive_triangles: Vec<crate::DisplayTriangle>,
    pub negative_triangles: Vec<crate::DisplayTriangle>,
    pub plane_origin: [f64; 3],
    pub plane_normal: [f64; 3],
    pub positive_area: f64,
    pub negative_area: f64,
}

struct SplitSurfaceHalves {
    positive_shell: Shape,
    negative_shell: Shape,
    plane_origin: [f64; 3],
    plane_normal: [f64; 3],
    positive_area: f64,
    negative_area: f64,
    before_area: f64,
}

struct BuiltTrimTool {
    surface: Shape,
    plane_origin: [f64; 3],
    plane_normal: [f64; 3],
}

impl NamedSolid {
    /// Evaluate a B-rep surface trim, removing `input.remove_side` and retaining the opposite half (#3918).
    pub(in crate::prism::named) fn evaluate_surface_trim_result(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        tool_body: Option<&NamedSolid>,
        input: &ResolvedSurfaceTrim,
    ) -> Result<NamedSolid, String> {
        if !self.is_surface() {
            return Err("surface trim target: expected an open surface body".into());
        }
        let target_shape = self.native_shape(history, document)?;
        let built_tool = build_trim_tool(target_shape, tool_body, history, document, &input.tool)?;
        let halves = split_surface_shell(target_shape, &built_tool)?;
        let (kept_shell, kept_area) = match input.remove_side {
            SurfaceTrimSide::Positive => (halves.negative_shell, halves.negative_area),
            SurfaceTrimSide::Negative => (halves.positive_shell, halves.positive_area),
        };
        if !(kept_area.is_finite() && kept_area > 0.0 && kept_area < halves.before_area - 1e-6) {
            return Err(format!(
                "surface trim kept area {kept_area} from {}: expected strictly reduced positive area",
                halves.before_area
            ));
        }
        let primary_body = self.body_reference_key(history, document)?.to_vec();
        let deleted = self.surface_trim_prior_keys(history, document, &primary_body)?;
        let resolution = match self {
            Self::Prism(body) => body.certification.as_ref().map(|c| c.resolution),
            Self::Derived(body) => body.resolution(),
        };
        let topology = BlendNamedTopology::from_trimmed_shell_preserving_body(
            history,
            document,
            feature,
            primary_body,
            deleted,
            kept_shell,
            resolution,
        )?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }

    /// Split a surface body into both candidate halves for interactive preview and region picking (#3918).
    /// Example: `surface.preview_surface_trim_regions(&history, doc, tool_body, &tool, options)?`.
    pub fn preview_surface_trim_regions(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool_body: Option<&NamedSolid>,
        tool: &ResolvedSurfaceTrimTool,
        options: PrismMeshOptions,
    ) -> Result<SurfaceTrimRegionPreview, String> {
        if !self.is_surface() {
            return Err("surface trim preview: expected an open surface body".into());
        }
        let target_shape = self.native_shape(history, document)?;
        let built_tool = build_trim_tool(target_shape, tool_body, history, document, tool)?;
        let halves = split_surface_shell(target_shape, &built_tool)?;
        let positive_triangles =
            crate::prism::mesh::body_face_display_triangles(&halves.positive_shell, options)?
                .into_iter()
                .flatten()
                .collect();
        let negative_triangles =
            crate::prism::mesh::body_face_display_triangles(&halves.negative_shell, options)?
                .into_iter()
                .flatten()
                .collect();
        Ok(SurfaceTrimRegionPreview {
            positive_triangles,
            negative_triangles,
            plane_origin: halves.plane_origin,
            plane_normal: halves.plane_normal,
            positive_area: halves.positive_area,
            negative_area: halves.negative_area,
        })
    }

    fn surface_trim_prior_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        primary_body: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut keys = self.reference_keys(history, document)?;
        keys.extend(self.subentity_reference_keys(history, document)?);
        if let Self::Derived(body) = self {
            keys.extend(body.live_keys().iter().cloned());
            keys.extend(body.deleted_keys().iter().cloned());
        }
        keys.retain(|key| key.as_slice() != primary_body);
        keys.sort();
        keys.dedup();
        Ok(keys)
    }
}

fn build_trim_tool(
    target_shape: &Shape,
    tool_body: Option<&NamedSolid>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    tool: &ResolvedSurfaceTrimTool,
) -> Result<BuiltTrimTool, String> {
    match tool {
        ResolvedSurfaceTrimTool::Plane(plane) => {
            let (center, extent) = body_bounds_center_and_extent(target_shape)?;
            let built = build_plane_tool(*plane, center, extent)?;
            Ok(BuiltTrimTool {
                surface: built.surface,
                plane_origin: built.plane_origin,
                plane_normal: built.plane_normal,
            })
        }
        ResolvedSurfaceTrimTool::SurfaceFace { face_key } => {
            let body = tool_body.ok_or_else(|| {
                "surface trim tool face: expected the cutting tool body".to_string()
            })?;
            let plane = body.face_plane(history, document, face_key)?;
            let source_face = body.native_face(history, document, face_key)?;
            let mut copier = crate::checked_shape::copy_shape(source_face, "surface trim tool")
                .map_err(|cause| cause.to_string())?;
            let copied = copier.shape().map_err(|cause| cause.to_string())?.clone();
            Ok(BuiltTrimTool {
                surface: copied,
                plane_origin: plane.recipe().origin,
                plane_normal: plane.normal(),
            })
        }
    }
}

fn split_surface_shell(
    target_shape: &Shape,
    built_tool: &BuiltTrimTool,
) -> Result<SplitSurfaceHalves, String> {
    let before_area = shape_surface_area(target_shape, "surface trim input")?;
    if !before_area.is_finite() || before_area <= 0.0 {
        return Err(format!(
            "surface trim input area {before_area}: expected positive finite surface area"
        ));
    }
    let mut copier = crate::checked_shape::copy_shape(target_shape, "surface trim input")
        .map_err(|cause| cause.to_string())?;
    let copied_target = copier.shape().map_err(|cause| cause.to_string())?.clone();

    let mut splitter = BRepAlgoAPISplitter::new();
    let mut arguments = List::default();
    arguments.append(copied_target);
    let mut tools = List::default();
    tools.append(built_tool.surface.clone());
    splitter.set_arguments(&arguments);
    splitter.set_tools(&tools);
    splitter
        .build()
        .map_err(|cause| format!("surface trim split build: {cause}"))?;
    if !splitter.is_done() || splitter.has_errors() {
        return Err("surface trim: BRepAlgoAPISplitter did not complete without errors".into());
    }
    let split_faces = collect_faces(splitter.shape())?;
    if split_faces.len() < 2 {
        return Err(format!(
            "surface trim produced {} faces: expected cutting tool to split surface into two regions",
            split_faces.len()
        ));
    }
    let mut positive_faces = Vec::new();
    let mut negative_faces = Vec::new();
    let mut positive_area = 0.0;
    let mut negative_area = 0.0;
    for face in split_faces {
        let (area, com) = face_area_and_center(&face)?;
        if area <= 1e-12 {
            continue;
        }
        let signed = (com[0] - built_tool.plane_origin[0]) * built_tool.plane_normal[0]
            + (com[1] - built_tool.plane_origin[1]) * built_tool.plane_normal[1]
            + (com[2] - built_tool.plane_origin[2]) * built_tool.plane_normal[2];
        if signed > 0.0 {
            positive_area += area;
            positive_faces.push(face);
        } else if signed < 0.0 {
            negative_area += area;
            negative_faces.push(face);
        } else {
            return Err("surface trim: split face centroid lies on the cutting plane".into());
        }
    }
    if positive_faces.is_empty() || negative_faces.is_empty() {
        return Err(format!(
            "surface trim regions (positive={}, negative={}): expected cutting tool to divide the surface into two sides",
            positive_faces.len(),
            negative_faces.len()
        ));
    }
    let area_tol = 1e-4 * before_area.max(1.0);
    if ((positive_area + negative_area) - before_area).abs() > area_tol {
        return Err(format!(
            "surface trim combined area {} from {}: expected surface area conservation",
            positive_area + negative_area,
            before_area
        ));
    }
    let positive_shell = build_checked_shell(&positive_faces, "surface trim positive region")?;
    let negative_shell = build_checked_shell(&negative_faces, "surface trim negative region")?;
    Ok(SplitSurfaceHalves {
        positive_shell,
        negative_shell,
        plane_origin: built_tool.plane_origin,
        plane_normal: built_tool.plane_normal,
        positive_area,
        negative_area,
        before_area,
    })
}

fn build_checked_shell(faces: &[Shape], label: &str) -> Result<Shape, String> {
    let mut shell = Shell::default();
    let builder = Builder;
    builder.make_shell(&mut shell);
    let mut shell_shape = shell.into_shape();
    for face in faces {
        builder
            .add(&mut shell_shape, face)
            .map_err(|cause| format!("{label} add face: {cause}"))?;
    }
    crate::checked_shape::require_geometry(&shell_shape, label)
        .map_err(|cause| cause.to_string())?;
    let valid = BRepCheckAnalyzer::new(&shell_shape, true, false, false)
        .and_then(|analyzer| analyzer.is_valid())
        .map_err(|cause| format!("{label} BRepCheck failed: {cause}"))?;
    if !valid {
        return Err(format!(
            "{label}: resulting shell failed BRepCheck validation"
        ));
    }
    Ok(shell_shape)
}

fn collect_faces(shape: &Shape) -> Result<Vec<Shape>, String> {
    let mut map = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, ShapeEnum::Face, &mut map)
        .map_err(|cause| cause.to_string())?;
    (1..=map.extent())
        .map(|index| {
            map.find_key(index)
                .cloned()
                .map_err(|cause| cause.to_string())
        })
        .collect()
}

fn shape_surface_area(shape: &Shape, label: &str) -> Result<f64, String> {
    let mut props = GProps::new();
    BRepGProp::surface_properties(shape, &mut props, false, false)
        .map_err(|cause| format!("{label} surface area: {cause}"))?;
    Ok(props.mass())
}

fn face_area_and_center(face: &Shape) -> Result<(f64, [f64; 3]), String> {
    let mut props = GProps::new();
    BRepGProp::surface_properties(face, &mut props, false, false)
        .map_err(|cause| format!("surface trim face properties: {cause}"))?;
    let area = props.mass();
    let com = props.centre_of_mass();
    let center = [com.x(), com.y(), com.z()];
    if !area.is_finite() || center.iter().any(|c| !c.is_finite()) {
        return Err("surface trim face properties: expected finite area and centroid".into());
    }
    Ok((area, center))
}
