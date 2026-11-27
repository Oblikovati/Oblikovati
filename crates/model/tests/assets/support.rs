// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    types,
    wire::{AppearanceInfo, MaterialInfo},
};
use serde_json::{Value, json};
pub fn material_record(source: types::AssetSource) -> MaterialInfo {
    serde_json::from_value(json!({
        "id":"physical", "displayName":"Physical asset", "source":source,
        "density":7.85, "appearanceId":"visual", "isotropyClass":"orthotropic",
        "mechanical":{"youngsModulus":210.,"poissonsRatio":0.3,"yieldStrength":250.,"ultimateTensileStrength":460.},
        "thermal":{"conductivity":54.,"expansionCoeff":0.000012,"specificHeat":486.},
        "electrical":{"resistivity":0.000000143,"relativePermittivity":1.2},
        "magnetic":{"class":"soft","relativePermeability":4200.,"remanence":0.2,"coercivity":0.08,"saturationFluxDensity":2.1,"coreLoss":0.5},
        "anisotropic":{"e1":143.,"e2":110.,"e3":92.,"g12":42.,"g23":38.,"g13":45.,"nu12":0.22,"nu23":0.19,"nu13":0.18,"alpha1":0.000011,"alpha2":0.000012,"alpha3":0.000013}
    })).unwrap()
}
pub fn appearance_record(source: types::AssetSource) -> AppearanceInfo {
    let mut fields = serde_json::to_value(AppearanceInfo::default()).unwrap();
    populate_visual_fields(&mut fields);
    fields["id"] = json!("visual");
    fields["displayName"] = json!("Visual asset");
    fields["source"] = json!(source);
    for field in ["normal", "tangent", "coatNormal", "coatTangent"] {
        fields["geometry"][field] = json!([0.1, 0.2, 0.3]);
    }
    serde_json::from_value(fields).unwrap()
}
fn populate_visual_fields(fields: &mut Value) {
    match fields {
        Value::Number(_) => *fields = json!(0.625),
        Value::Bool(_) => *fields = json!(true),
        Value::Object(object) => object.values_mut().for_each(populate_visual_fields),
        Value::Array(array) => array.iter_mut().for_each(populate_visual_fields),
        _ => {}
    }
}
