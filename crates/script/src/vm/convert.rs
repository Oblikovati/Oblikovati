// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use serde_json::{Map, Number, Value as JsonValue};
const MAX_DEPTH: usize = 64;
const MAX_SAFE_INTEGER: i64 = 9_007_199_254_740_991;
pub(super) fn arguments(table: Table) -> mlua::Result<Vec<u8>> {
    let converted = to_json(Value::Table(table), 0)?;
    serde_json::to_vec(&converted).map_err(failure)
}
pub(super) fn reply(lua: &Lua, bytes: &[u8]) -> mlua::Result<Value> {
    if bytes.is_empty() {
        return lua.create_table().map(Value::Table);
    }
    let parsed: JsonValue = serde_json::from_slice(bytes).map_err(failure)?;
    from_json(lua, parsed, 0)
}
fn require_depth(depth: usize) -> mlua::Result<()> {
    if depth > MAX_DEPTH {
        return Err(failure(format!(
            "table depth {depth}: expected at most {MAX_DEPTH}, with no cycles"
        )));
    }
    Ok(())
}
fn to_json(value: Value, depth: usize) -> mlua::Result<JsonValue> {
    require_depth(depth)?;
    match value {
        Value::Nil => Ok(JsonValue::Null),
        Value::Boolean(boolean) => Ok(JsonValue::Bool(boolean)),
        Value::String(text) => Ok(JsonValue::String(text.to_string_lossy())),
        Value::Integer(integer) => number_to_json(integer as f64),
        Value::Number(number) => number_to_json(number),
        Value::Table(table) => table_to_json(table, depth),
        other => Err(failure(format!(
            "Lua {}: expected nil, boolean, number, string or table",
            other.type_name()
        ))),
    }
}
fn number_to_json(number: f64) -> mlua::Result<JsonValue> {
    if !number.is_finite() {
        return Err(failure(format!(
            "Lua number {number}: expected finite JSON number"
        )));
    }
    if number.fract() == 0.0 {
        if number.abs() > MAX_SAFE_INTEGER as f64 {
            return Err(failure(format!(
                "Lua integer {number}: expected exact identifier within ±{MAX_SAFE_INTEGER}"
            )));
        }
        return Ok(JsonValue::Number(Number::from(number as i64)));
    }
    Number::from_f64(number)
        .map(JsonValue::Number)
        .ok_or_else(|| failure(format!("number {number}: expected finite JSON number")))
}
fn table_to_json(table: Table, depth: usize) -> mlua::Result<JsonValue> {
    let entries: Vec<(Value, Value)> = table.pairs().collect::<mlua::Result<_>>()?;
    if !entries.is_empty()
        && entries
            .iter()
            .all(|(key, _)| sequence_index(key, entries.len()))
    {
        return sequence_to_json(&table, entries.len(), depth);
    }
    object_to_json(entries, depth)
}
fn sequence_index(key: &Value, length: usize) -> bool {
    match key {
        Value::Integer(index) => *index >= 1 && (*index as u64) <= length as u64,
        Value::Number(index) => index.fract() == 0.0 && *index >= 1.0 && *index <= length as f64,
        _ => false,
    }
}
fn sequence_to_json(table: &Table, length: usize, depth: usize) -> mlua::Result<JsonValue> {
    let mut values = Vec::with_capacity(length);
    for index in 1..=length {
        values.push(to_json(table.raw_get(index)?, depth + 1)?);
    }
    Ok(JsonValue::Array(values))
}
fn object_to_json(entries: Vec<(Value, Value)>, depth: usize) -> mlua::Result<JsonValue> {
    let mut object = Map::new();
    for (key, value) in entries {
        let key = object_key(key)?;
        if object.contains_key(&key) {
            return Err(failure(format!(
                "table key {key:?}: expected unique JSON object keys"
            )));
        }
        object.insert(key, to_json(value, depth + 1)?);
    }
    Ok(JsonValue::Object(object))
}
fn object_key(key: Value) -> mlua::Result<String> {
    match key {
        Value::String(text) => Ok(text.to_string_lossy()),
        Value::Integer(integer) => Ok(integer.to_string()),
        Value::Number(number) if number.is_finite() => Ok(number.to_string()),
        Value::Boolean(boolean) => Ok(boolean.to_string()),
        other => Err(failure(format!(
            "Lua key {}: expected string, finite number or boolean",
            other.type_name()
        ))),
    }
}
fn from_json(lua: &Lua, value: JsonValue, depth: usize) -> mlua::Result<Value> {
    require_depth(depth)?;
    match value {
        JsonValue::Null => Ok(Value::Nil),
        JsonValue::Bool(boolean) => Ok(Value::Boolean(boolean)),
        JsonValue::String(text) => lua.create_string(text).map(Value::String),
        JsonValue::Number(number) => json_number(number),
        JsonValue::Array(values) => json_array(lua, values, depth),
        JsonValue::Object(fields) => json_object(lua, fields, depth),
    }
}
fn json_number(number: Number) -> mlua::Result<Value> {
    let floating = number
        .as_f64()
        .ok_or_else(|| failure(format!("JSON number {number}: expected finite Lua number")))?;
    if floating.fract() == 0.0 && floating.abs() > MAX_SAFE_INTEGER as f64 {
        return Err(failure(format!(
            "JSON integer {number}: expected exact Lua identifier within ±{MAX_SAFE_INTEGER}"
        )));
    }
    Ok(Value::Number(floating))
}
fn json_array(lua: &Lua, values: Vec<JsonValue>, depth: usize) -> mlua::Result<Value> {
    let table = lua.create_table()?;
    for (index, value) in values.into_iter().enumerate() {
        table.raw_set(index + 1, from_json(lua, value, depth + 1)?)?;
    }
    Ok(Value::Table(table))
}
fn json_object(lua: &Lua, fields: Map<String, JsonValue>, depth: usize) -> mlua::Result<Value> {
    let table = lua.create_table()?;
    for (key, value) in fields {
        table.raw_set(key, from_json(lua, value, depth + 1)?)?;
    }
    Ok(Value::Table(table))
}
