// SPDX-License-Identifier: GPL-2.0-only
//! Exact Go glyph sources for the currently supported command set.
macro_rules! glyph_catalog {
    ($($variant:ident => $name:literal),+ $(,)?) => {
        #[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
        pub(crate) enum IconGlyph { $($variant),+ }
        pub(crate) const GLYPHS: &[IconGlyph] = &[$(IconGlyph::$variant),+];
        pub(crate) const SOURCES: &[(&str,&[u8])] = &[$(($name,include_bytes!(concat!("../../assets/icons/",$name,".svg")))),+];
        impl IconGlyph {
            pub(crate) fn index(self) -> usize { self as usize }
            pub(crate) fn source(self) -> &'static [u8] { SOURCES[self.index()].1 }
            pub(crate) fn name(self) -> &'static str { SOURCES[self.index()].0 }
        }
    };
}
glyph_catalog! {
    NewPart => "new-part",
    NewAssembly => "new-assembly",
    NewDrawing => "new-drawing",
    NewSheetMetal => "sheet-metal-new",
    TemplateDocument => "template-document",
    CreateSketch => "create-sketch",
    FinishSketch => "finish-sketch",
    Line => "line",
    Rectangle => "rectangle",
    Circle => "circle",
    Arc => "arc",
    Point => "point",
    Ellipse => "ellipse",
    Dimension => "dimension",
    Coincident => "coincident",
    Collinear => "collinear",
    Parallel => "parallel",
    Perpendicular => "perpendicular",
    Horizontal => "horizontal",
    Vertical => "vertical",
    Tangent => "tangent",
    Concentric => "concentric",
    Equal => "equal",
    Fix => "fix",
    Construction => "construction",
    Centerline => "centerline",
    Parameters => "parameters",
    Home => "home",
    Save => "save",
    Undo => "undo",
    Redo => "redo",
    Extrude => "extrude",
    Revolve => "revolve",
    Hole => "hole",
    Fillet => "fillet",
    Chamfer => "chamfer",
    Move => "move",
    Measure => "measure",
    RectangularPattern => "rectangular-pattern",
    CircularPattern => "circular-pattern",
    Simplify => "simplify",
    WorkPlane => "work-plane-offset",
    WorkAxis => "work-axis",
    WorkPoint => "work-point",
}
