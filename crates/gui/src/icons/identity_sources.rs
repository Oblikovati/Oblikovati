// SPDX-License-Identifier: GPL-2.0-only
//! Vector artwork from the Oblikovati design handoff.
use super::IconGlyph;

pub(super) fn source(glyph: IconGlyph, light: bool) -> Option<&'static str> {
    Some(match glyph {
        IconGlyph::BrandWordmark if light => {
            include_str!("../../assets/identity/oblikovati-wordmark-light.svg")
        }
        IconGlyph::BrandWordmark => {
            include_str!("../../assets/identity/oblikovati-wordmark-dark.svg")
        }
        IconGlyph::PlaceComponent => {
            include_str!("../../assets/identity/cmd-asm-assembly-component-placement-32.svg")
        }
        IconGlyph::Derive => include_str!("../../assets/identity/cmd-model-derive-32.svg"),
        IconGlyph::Feature => {
            include_str!("../../assets/identity/cmd-model-content-center-feature-32.svg")
        }
        IconGlyph::InsertIFeature => {
            include_str!("../../assets/identity/cmd-model-insert-ifeature-32.svg")
        }
        IconGlyph::Import => include_str!("../../assets/identity/cmd-model-import-32.svg"),
        IconGlyph::SketchVisibility => {
            include_str!("../../assets/identity/cmd-sketch-sketch-visibility-32.svg")
        }
        IconGlyph::RibbonDensity => {
            include_str!("../../assets/identity/cmd-app-ribbon-density-32.svg")
        }
        IconGlyph::NewPart => include_str!("../../assets/identity/doc-part-16.svg"),
        IconGlyph::NewAssembly => include_str!("../../assets/identity/doc-assembly-16.svg"),
        IconGlyph::NewDrawing => include_str!("../../assets/identity/doc-drawing-16.svg"),
        IconGlyph::Open => include_str!("../../assets/identity/cmd-app-document-open-32.svg"),
        IconGlyph::Save => include_str!("../../assets/identity/cmd-app-save-32.svg"),
        IconGlyph::Home => include_str!("../../assets/identity/cmd-app-home-recent-32.svg"),
        IconGlyph::Customize => {
            include_str!("../../assets/identity/cmd-app-ribbon-customize-32.svg")
        }
        IconGlyph::Projects => {
            include_str!("../../assets/identity/cmd-app-project-settings-32.svg")
        }
        IconGlyph::NewWindow => include_str!("../../assets/identity/cmd-app-new-view-32.svg"),
        IconGlyph::UserInterface => {
            include_str!("../../assets/identity/cmd-app-workspace-layout-32.svg")
        }
        IconGlyph::CreateSketch => include_str!("../../assets/identity/cmd-sketch-2d-32.svg"),
        IconGlyph::Extrude => include_str!("../../assets/identity/cmd-extrude-32.svg"),
        IconGlyph::Revolve => include_str!("../../assets/identity/cmd-revolve-32.svg"),
        IconGlyph::Sweep => include_str!("../../assets/identity/cmd-model-sweep-32.svg"),
        IconGlyph::Loft => include_str!("../../assets/identity/cmd-model-loft-32.svg"),
        IconGlyph::Fillet => include_str!("../../assets/identity/cmd-fillet-32.svg"),
        IconGlyph::Hole => include_str!("../../assets/identity/cmd-hole-32.svg"),
        IconGlyph::Shell => include_str!("../../assets/identity/cmd-shell-32.svg"),
        IconGlyph::Split => include_str!("../../assets/identity/cmd-model-split-32.svg"),
        IconGlyph::WorkPlane => include_str!("../../assets/identity/cmd-work-plane-32.svg"),
        IconGlyph::RectangularPattern => {
            include_str!("../../assets/identity/cmd-pattern-rect-32.svg")
        }
        IconGlyph::BrandMark => include_str!("../../assets/identity/oblikovati-mark.svg"),
        IconGlyph::NavPan => include_str!("../../assets/identity/cmd-app-pan-32.svg"),
        IconGlyph::NavOrbit => include_str!("../../assets/identity/cmd-app-orbit-32.svg"),
        IconGlyph::NavZoom => include_str!("../../assets/identity/cmd-app-zoom-32.svg"),
        IconGlyph::NavWheel => include_str!("../../assets/identity/cmd-app-nav-wheel-32.svg"),
        IconGlyph::RecentPin => include_str!("../../assets/identity/cmd-app-pin-16.svg"),
        IconGlyph::RecentPinned => include_str!("../../assets/identity/cmd-app-pinned-16.svg"),
        IconGlyph::Line => include_str!("../../assets/identity/cmd-sketch-line-32.svg"),
        IconGlyph::FinishSketch => include_str!("../../assets/identity/cmd-sketch-finish-32.svg"),
        IconGlyph::MidpointLine => {
            include_str!("../../assets/identity/cmd-sketch-line-midpoint-32.svg")
        }
        IconGlyph::Rectangle => {
            include_str!("../../assets/identity/cmd-sketch-rectangle-two-point-32.svg")
        }
        IconGlyph::Circle => {
            include_str!("../../assets/identity/cmd-sketch-circle-center-point-32.svg")
        }
        IconGlyph::Point => include_str!("../../assets/identity/cmd-sketch-point-32.svg"),
        IconGlyph::Ellipse => include_str!("../../assets/identity/cmd-sketch-ellipse-32.svg"),
        IconGlyph::Coincident => include_str!("../../assets/identity/cmd-sketch-coincident-32.svg"),
        IconGlyph::Collinear => include_str!("../../assets/identity/cmd-sketch-collinear-32.svg"),
        IconGlyph::Parallel => include_str!("../../assets/identity/cmd-sketch-parallel-32.svg"),
        IconGlyph::Perpendicular => {
            include_str!("../../assets/identity/cmd-sketch-perpendicular-32.svg")
        }
        IconGlyph::Horizontal => include_str!("../../assets/identity/cmd-sketch-horizontal-32.svg"),
        IconGlyph::Vertical => include_str!("../../assets/identity/cmd-sketch-vertical-32.svg"),
        IconGlyph::Tangent => include_str!("../../assets/identity/cmd-sketch-tangent-32.svg"),
        IconGlyph::Concentric => include_str!("../../assets/identity/cmd-sketch-concentric-32.svg"),
        IconGlyph::Equal => include_str!("../../assets/identity/cmd-sketch-equal-32.svg"),
        IconGlyph::Fix => include_str!("../../assets/identity/cmd-sketch-fix-32.svg"),
        IconGlyph::Construction => {
            include_str!("../../assets/identity/cmd-sketch-construction-32.svg")
        }
        IconGlyph::Centerline => include_str!("../../assets/identity/cmd-sketch-centerline-32.svg"),
        IconGlyph::Move => include_str!("../../assets/identity/cmd-sketch-move-32.svg"),
        IconGlyph::PreciseInput => {
            include_str!("../../assets/identity/cmd-sketch-precise-input-32.svg")
        }
        IconGlyph::InsertImage => include_str!("../../assets/identity/cmd-sketch-image-32.svg"),
        IconGlyph::ProjectGeometry => {
            include_str!("../../assets/identity/cmd-sketch-project-geometry-32.svg")
        }
        IconGlyph::CreateBlock => {
            include_str!("../../assets/identity/cmd-sketch-create-block-32.svg")
        }
        IconGlyph::Chamfer => include_str!("../../assets/identity/cmd-model-chamfer-32.svg"),
        IconGlyph::MoveBodies => include_str!("../../assets/identity/cmd-model-move-bodies-32.svg"),
        IconGlyph::BendPart => include_str!("../../assets/identity/cmd-model-bend-part-32.svg"),
        IconGlyph::CopyObject => include_str!("../../assets/identity/cmd-model-copy-object-32.svg"),
        IconGlyph::DeleteFace => {
            include_str!("../../assets/identity/cmd-model-delete-face-32.svg")
        }
        IconGlyph::DirectEdit => include_str!("../../assets/identity/cmd-model-direct-edit-32.svg"),
        IconGlyph::Finish => include_str!("../../assets/identity/cmd-model-finish-32.svg"),
        IconGlyph::ExtendSurface => {
            include_str!("../../assets/identity/cmd-model-extend-surface-32.svg")
        }
        IconGlyph::CircularPattern => {
            include_str!("../../assets/identity/cmd-model-circular-pattern-32.svg")
        }
        IconGlyph::Simplify => include_str!("../../assets/identity/cmd-model-simplify-32.svg"),
        IconGlyph::WorkAxis => include_str!("../../assets/identity/cmd-model-work-axis-32.svg"),
        IconGlyph::WorkPoint => include_str!("../../assets/identity/cmd-model-work-point-32.svg"),
        IconGlyph::WorkCoordinateSystem => {
            include_str!("../../assets/identity/cmd-model-user-coordinate-system-32.svg")
        }
        IconGlyph::CenterRectangle => {
            include_str!("../../assets/identity/cmd-sketch-rectangle-center-32.svg")
        }
        IconGlyph::ThreePointRectangle => {
            include_str!("../../assets/identity/cmd-sketch-rectangle-three-point-32.svg")
        }
        IconGlyph::ThreePointCenterRectangle => {
            include_str!("../../assets/identity/cmd-sketch-rectangle-three-point-center-32.svg")
        }
        IconGlyph::CenterPointArc => {
            include_str!("../../assets/identity/cmd-sketch-arc-center-point-32.svg")
        }
        IconGlyph::Arc => include_str!("../../assets/identity/cmd-sketch-arc-three-point-32.svg"),
        IconGlyph::SketchFillet => include_str!("../../assets/identity/cmd-sketch-fillet-32.svg"),
        IconGlyph::SketchText => include_str!("../../assets/identity/cmd-sketch-text-32.svg"),
        IconGlyph::Coil => include_str!("../../assets/identity/cmd-model-coil-32.svg"),
        IconGlyph::Emboss => include_str!("../../assets/identity/cmd-model-emboss-32.svg"),
        IconGlyph::Rib => include_str!("../../assets/identity/cmd-model-rib-32.svg"),
        IconGlyph::Decal => include_str!("../../assets/identity/cmd-model-decal-32.svg"),
        IconGlyph::Unwrap => include_str!("../../assets/identity/cmd-model-unwrap-32.svg"),
        IconGlyph::BoundaryPatch => {
            include_str!("../../assets/identity/cmd-model-boundary-patch-32.svg")
        }
        IconGlyph::Measure => {
            include_str!("../../assets/identity/cmd-sketch-measure-entity-picking-32.svg")
        }
        IconGlyph::MaterialBrowser => {
            include_str!("../../assets/identity/cmd-app-material-browser-32.svg")
        }
        IconGlyph::AppearanceBrowser => {
            include_str!("../../assets/identity/cmd-app-appearance-browser-32.svg")
        }
        IconGlyph::Dimension => {
            include_str!("../../assets/identity/cmd-sketch-dimension-linear-32.svg")
        }
        IconGlyph::Select => {
            include_str!("../../assets/identity/cmd-app-viewport-entity-selection-32.svg")
        }
        _ => return None,
    })
}
