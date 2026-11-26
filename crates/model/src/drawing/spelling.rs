// SPDX-License-Identifier: GPL-2.0-only
//! Native Go recipe spellings for the canonical API enums, not a second enum namespace.
use oblikovati_api::types::*;
macro_rules! recipe_spelling {
    ($module:ident, $kind:ty, {$($variant:path => $text:literal),+ $(,)?}) => {
        pub(super) mod $module {
            use super::*;
            const VALUES: &[($kind, &str)] = &[$(($variant, $text)),+];
            pub fn serialize<S: serde::Serializer>(value: &$kind, output: S) -> Result<S::Ok, S::Error> {
                let name = name(value)
                    .ok_or_else(|| serde::ser::Error::custom(format!("drawing enum {value:?}: expected a canonical recipe spelling")))?;
                output.serialize_str(name)
            }
            pub fn name(value: &$kind) -> Option<&'static str> {
                VALUES.iter().find(|(kind, _)| kind == value).map(|(_, text)| *text)
            }
            pub fn parse(name: &str) -> Option<$kind> {
                VALUES.iter().find(|(_, text)| *text == name).map(|(kind, _)| *kind)
            }
            pub fn deserialize<'de, D: serde::Deserializer<'de>>(input: D) -> Result<$kind, D::Error> {
                let name = <String as serde::Deserialize>::deserialize(input)?;
                parse(&name)
                    .ok_or_else(|| serde::de::Error::custom(format!("drawing enum {name:?}: expected one of {VALUES:?}")))
            }
        }
    };
}
recipe_spelling!(sheet_size, SheetSize, {
    SHEET_SIZE_CUSTOM => "custom", SHEET_SIZE_A0 => "a0", SHEET_SIZE_A1 => "a1", SHEET_SIZE_A2 => "a2",
    SHEET_SIZE_A3 => "a3", SHEET_SIZE_A4 => "a4", SHEET_SIZE_ANSI_A => "ansiA", SHEET_SIZE_ANSI_B => "ansiB",
    SHEET_SIZE_ANSI_C => "ansiC", SHEET_SIZE_ANSI_D => "ansiD", SHEET_SIZE_ANSI_E => "ansiE"
});
recipe_spelling!(sheet_orientation, SheetOrientation, { SHEET_PORTRAIT => "portrait", SHEET_LANDSCAPE => "landscape" });
recipe_spelling!(base_orientation, BaseViewOrientation, {
    BASE_VIEW_FRONT => "front", BASE_VIEW_TOP => "top", BASE_VIEW_RIGHT => "right", BASE_VIEW_BACK => "back",
    BASE_VIEW_LEFT => "left", BASE_VIEW_BOTTOM => "bottom", BASE_VIEW_ISO => "iso"
});
recipe_spelling!(view_style, DrawingViewStyle, {
    HIDDEN_LINE_VIEW_STYLE => "hiddenLine", WIREFRAME_VIEW_STYLE => "wireframe", SHADED_VIEW_STYLE => "shaded",
    HIDDEN_LINE_REMOVED_VIEW_STYLE => "hiddenLineRemoved", FROM_BASE_VIEW_STYLE => "fromBase",
    SHADED_HIDDEN_LINE_VIEW_STYLE => "shadedHiddenLine"
});

recipe_spelling!(drafting_standard, DraftingStandard, { DRAFTING_ISO => "ISO", DRAFTING_ANSI => "ANSI" });

recipe_spelling!(view_kind, DrawingViewType, { DRAWING_VIEW_BASE => "base" });

recipe_spelling!(title_block_location, TitleBlockLocation, {
    BOTTOM_RIGHT_TITLE_BLOCK => "bottomRight", BOTTOM_LEFT_TITLE_BLOCK => "bottomLeft",
    TOP_LEFT_TITLE_BLOCK => "topLeft", TOP_RIGHT_TITLE_BLOCK => "topRight", BOTTOM_RIGHT_TITLE_BLOCK => ""
});
