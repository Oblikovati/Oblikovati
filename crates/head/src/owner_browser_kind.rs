// SPDX-License-Identifier: GPL-2.0-only
//! Browser display kind labels for authored Part features.

/// What a search calls this kind of feature.
pub(super) fn feature_kind_name(
    definition: &oblikovati_model::feature::part::PartFeatureDefinition,
) -> &'static str {
    use oblikovati_model::feature::part::PartFeatureDefinition::{
        Chamfer, DeleteBody, DerivedPart, Fillet, ImportedModel, SketchCoil, SketchDecal,
        SketchEmboss, SketchLoft, SketchPrism, SketchRevolve, SketchRib, SketchSweep, Unwrap,
    };
    match definition {
        SketchPrism(_) => "Extrusion",
        SketchRevolve(_) => "Revolution",
        SketchCoil(_) => "Coil",
        SketchEmboss(_) => "Emboss",
        SketchRib(_) => "Rib",
        SketchDecal(_) => "Decal",
        DerivedPart(_) => "Derived Part",
        ImportedModel(def) => match def.import_mode() {
            oblikovati_model::feature::imported_model::ImportMode::ReferenceModel => {
                "Imported Reference"
            }
            oblikovati_model::feature::imported_model::ImportMode::ConvertModel => "Imported Model",
        },
        Unwrap(_) => "Unwrap",
        SketchSweep(_) => "Sweep",
        SketchLoft(_) => "Loft",
        Fillet(_) => "Fillet",
        Chamfer(_) => "Chamfer",
        oblikovati_model::feature::part::PartFeatureDefinition::Hole(_) => "Hole",
        oblikovati_model::feature::part::PartFeatureDefinition::Slot(_) => "Slot",
        oblikovati_model::feature::part::PartFeatureDefinition::Combine(_) => "Combine",
        oblikovati_model::feature::part::PartFeatureDefinition::DirectEdit(_) => "Direct Edit",
        oblikovati_model::feature::part::PartFeatureDefinition::Draft(_) => "Draft",
        oblikovati_model::feature::part::PartFeatureDefinition::Split(_) => "Split",
        oblikovati_model::feature::part::PartFeatureDefinition::DeleteFace(_) => "Delete Face",
        oblikovati_model::feature::part::PartFeatureDefinition::ReplaceFace(_) => "Replace Face",
        oblikovati_model::feature::part::PartFeatureDefinition::Shell(_) => "Shell",
        oblikovati_model::feature::part::PartFeatureDefinition::Thread(_) => "Thread",
        oblikovati_model::feature::part::PartFeatureDefinition::Mark(_) => "Mark",
        oblikovati_model::feature::part::PartFeatureDefinition::Finish(_) => "Finish",
        oblikovati_model::feature::part::PartFeatureDefinition::RuledSurface(_) => "Ruled Surface",
        oblikovati_model::feature::part::PartFeatureDefinition::Ucs(_) => "User Coordinate System",
        oblikovati_model::feature::part::PartFeatureDefinition::ExtendSurface(_) => {
            "Extend Surface"
        }
        oblikovati_model::feature::part::PartFeatureDefinition::BoundaryPatch(_) => {
            "Boundary Patch"
        }
        oblikovati_model::feature::part::PartFeatureDefinition::ThickenOffset(def) => {
            if def.output().is_surface() {
                "Offset Surface"
            } else {
                "Thicken"
            }
        }
        oblikovati_model::feature::part::PartFeatureDefinition::Stitch(_) => "Stitch",
        DeleteBody(_) => "Delete Body",
        oblikovati_model::feature::part::PartFeatureDefinition::Fixed(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::Offset { .. }
        | oblikovati_model::feature::part::PartFeatureDefinition::ParallelThroughPoint(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::ThroughThreePoints(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::LineThroughPoint(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::NormalToAxis(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::AtAngleAboutAxis(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::ThroughTwoAxes(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::WorkPlaneExtended(_) => {
            "Work Plane"
        }
        oblikovati_model::feature::part::PartFeatureDefinition::Axis(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::AxisThroughPoints(_) => {
            "Work Axis"
        }
        oblikovati_model::feature::part::PartFeatureDefinition::Point(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::PointAtReference(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::WorkPoint(_) => "Work Point",
        _ => "Work Feature",
    }
}
