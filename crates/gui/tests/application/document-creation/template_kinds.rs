// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::SUB_TYPE_SHEET_METAL_PART;
fn template(kind: GuiTemplateKind, units: GuiTemplateUnits) -> GuiDocumentTemplate {
    GuiDocumentTemplate { kind, units }
}
#[test]
fn units_answer_their_folder_and_length_unit() {
    let metric = template(GuiTemplateKind::Part, GuiTemplateUnits::Metric);
    let inch = template(GuiTemplateKind::Part, GuiTemplateUnits::Inch);
    assert_eq!(
        (metric.length_unit(), metric.units.folder()),
        ("mm", "Metric")
    );
    assert_eq!((inch.length_unit(), inch.units.folder()), ("in", "English"));
}
#[test]
fn every_kind_has_its_title_type_and_name() {
    use GuiTemplateKind::*;
    use GuiTemplateUnits::*;
    for (kind, units, title, type_name, prefix) in [
        (Part, Metric, "Standard metric part", "part", "Part"),
        (Part, Inch, "Standard inch part", "part", "Part"),
        (
            SheetMetal,
            Metric,
            "Metric sheet-metal part",
            "part",
            "Part",
        ),
        (SheetMetal, Inch, "Inch sheet-metal part", "part", "Part"),
        (
            Assembly,
            Metric,
            "Standard metric assembly",
            "assembly",
            "Assembly",
        ),
        (
            Drawing,
            Metric,
            "Metric drawing with title block",
            "drawing",
            "Drawing",
        ),
        (
            Presentation,
            Metric,
            "Standard metric presentation",
            "presentation",
            "Presentation",
        ),
    ] {
        let tile = template(kind, units);
        assert_eq!(tile.title(), title);
        assert_eq!(tile.type_name(), type_name);
        assert_eq!(tile.name_prefix(), prefix);
    }
}
#[test]
fn only_sheet_metal_has_a_flavor_and_it_is_the_registered_one() {
    let sheet = template(GuiTemplateKind::SheetMetal, GuiTemplateUnits::Metric);
    assert_eq!(sheet.sub_type(), SUB_TYPE_SHEET_METAL_PART.0);
    let part = template(GuiTemplateKind::Part, GuiTemplateUnits::Metric);
    assert_eq!(part.sub_type(), "");
}
#[test]
fn every_kind_appears_in_exactly_one_group() {
    let tiles: Vec<GuiTemplateKind> = TEMPLATE_GROUPS
        .iter()
        .flat_map(|(_, kinds)| kinds.iter().copied())
        .collect();
    assert_eq!(tiles.len(), 5);
    for (index, tile) in tiles.iter().enumerate() {
        assert!(
            !tiles[index + 1..].contains(tile),
            "tile {tile:?}: expected one group"
        );
    }
}
