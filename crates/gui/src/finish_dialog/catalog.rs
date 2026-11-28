// SPDX-License-Identifier: GPL-2.0-only
//! Process presets and defaults for the five Part Modify Finish types (#3918).
use super::GuiFinishType;

/// Preset values populated when a user activates a Finish type or selects a Process.
#[derive(Clone, Copy, Debug)]
pub(super) struct FinishPreset {
    pub process: &'static str,
    pub appearance: &'static str,
    pub thickness: &'static str,
    pub hardness: &'static str,
    pub depth: &'static str,
    pub description: &'static str,
    pub short_description: &'static str,
}

const APPEARANCE_PRESETS: &[FinishPreset] = &[
    FinishPreset {
        process: "Anodized - Teal",
        appearance: "Teal",
        thickness: "10 micron",
        hardness: "",
        depth: "",
        description: "Anodized surface appearance",
        short_description: "Anodized",
    },
    FinishPreset {
        process: "Chrome - Polished",
        appearance: "Chrome - Polished",
        thickness: "5 micron",
        hardness: "",
        depth: "",
        description: "Polished chrome appearance",
        short_description: "Chrome",
    },
    FinishPreset {
        process: "Black Oxide",
        appearance: "Black Oxide",
        thickness: "3 micron",
        hardness: "",
        depth: "",
        description: "Black oxide coating",
        short_description: "BlackOx",
    },
];

const COATING_PRESETS: &[FinishPreset] = &[
    FinishPreset {
        process: "Default Coating",
        appearance: "Silver",
        thickness: "8 micron",
        hardness: "",
        depth: "",
        description: "Standard metallic plating",
        short_description: "Plating",
    },
    FinishPreset {
        process: "Zn12/Cn/T2",
        appearance: "Turquoise",
        thickness: "12 micron",
        hardness: "",
        depth: "",
        description: "Zinc plating chromate passivation",
        short_description: "Zn12/Cn/T2",
    },
    FinishPreset {
        process: "Ni/Cr",
        appearance: "Chrome - Polished",
        thickness: "20 micron",
        hardness: "",
        depth: "",
        description: "Nickel chromium plating",
        short_description: "Ni/Cr",
    },
];

const HEAT_TREATMENT_PRESETS: &[FinishPreset] = &[
    FinishPreset {
        process: "Case Hardening",
        appearance: "Natural",
        thickness: "",
        hardness: "58 HRC",
        depth: "0.5 mm",
        description: "Surface case hardening",
        short_description: "CaseHard",
    },
    FinishPreset {
        process: "Nitriding",
        appearance: "Black",
        thickness: "",
        hardness: "65 HRC",
        depth: "0.3 mm",
        description: "Gas nitriding treatment",
        short_description: "Nitride",
    },
    FinishPreset {
        process: "Annealing",
        appearance: "Natural",
        thickness: "",
        hardness: "20 HRC",
        depth: "1.0 mm",
        description: "Stress relief annealing",
        short_description: "Anneal",
    },
];

const TEXTURE_PRESETS: &[FinishPreset] = &[
    FinishPreset {
        process: "Barrel Finishing",
        appearance: "Barrel Finished",
        thickness: "",
        hardness: "",
        depth: "",
        description: "Tumbled barrel surface finish",
        short_description: "Barrel",
    },
    FinishPreset {
        process: "Sandblasting",
        appearance: "Satin",
        thickness: "",
        hardness: "",
        depth: "",
        description: "Matte sandblasted texture",
        short_description: "Sandblast",
    },
    FinishPreset {
        process: "Polishing",
        appearance: "Chrome - Polished",
        thickness: "",
        hardness: "",
        depth: "",
        description: "Fine mechanical polish",
        short_description: "Polish",
    },
];

const PAINT_PRESETS: &[FinishPreset] = &[
    FinishPreset {
        process: "Prime/Paint RAL 3018",
        appearance: "Red",
        thickness: "50 micron",
        hardness: "",
        depth: "",
        description: "Primer and RAL 3018 strawberry red paint",
        short_description: "RAL 3018",
    },
    FinishPreset {
        process: "Prime/Paint RAL 5010",
        appearance: "Blue",
        thickness: "50 micron",
        hardness: "",
        depth: "",
        description: "Primer and RAL 5010 gentian blue paint",
        short_description: "RAL 5010",
    },
    FinishPreset {
        process: "Prime/Paint RAL 6018",
        appearance: "Green",
        thickness: "50 micron",
        hardness: "",
        depth: "",
        description: "Primer and RAL 6018 yellow green paint",
        short_description: "RAL 6018",
    },
];

pub(super) const APPEARANCE_CHOICES: &[&str] = &[
    "Red",
    "Turquoise",
    "Teal",
    "Blue",
    "Green",
    "Yellow",
    "Black",
    "Black Oxide",
    "Chrome - Polished",
    "Silver",
    "Barrel Finished",
    "Satin",
    "Natural",
];

pub(super) fn presets_for(kind: GuiFinishType) -> &'static [FinishPreset] {
    match kind {
        GuiFinishType::Appearance => APPEARANCE_PRESETS,
        GuiFinishType::MaterialCoating => COATING_PRESETS,
        GuiFinishType::HeatTreatment => HEAT_TREATMENT_PRESETS,
        GuiFinishType::SurfaceTexture => TEXTURE_PRESETS,
        GuiFinishType::Paint => PAINT_PRESETS,
    }
}

pub(super) fn default_preset(kind: GuiFinishType) -> &'static FinishPreset {
    &presets_for(kind)[0]
}
