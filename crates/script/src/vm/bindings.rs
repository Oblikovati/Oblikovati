// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn install(
    lua: &Lua,
    pipe: Rc<RefCell<LuaHostPipe>>,
    methods: &[String],
) -> mlua::Result<()> {
    let root = lua.create_table()?;
    for method in methods {
        install_group(lua, &root, pipe.clone(), method)?;
    }
    let generic = pipe.clone();
    root.set(
        "call",
        lua.create_function(move |lua, (method, arguments): (String, Option<Table>)| {
            invoke(lua, &generic, method, arguments)
        })?,
    )?;
    let methods = methods.to_vec();
    root.set(
        "methods",
        lua.create_function(move |lua, ()| lua.create_sequence_from(methods.clone()))?,
    )?;
    lua.globals().set("oblikovati", root)?;
    install_print(lua, pipe)
}
fn install_group(
    lua: &Lua,
    root: &Table,
    pipe: Rc<RefCell<LuaHostPipe>>,
    method: &str,
) -> mlua::Result<()> {
    let segments: Vec<&str> = method.split('.').collect();
    if segments.len() < 2 {
        return Ok(());
    }
    let mut parent = root.clone();
    for segment in &segments[..segments.len() - 1] {
        parent = group_table(lua, &parent, segment)?;
    }
    let method = method.to_owned();
    parent.set(
        segments[segments.len() - 1],
        lua.create_function(move |lua, arguments: Option<Table>| {
            invoke(lua, &pipe, method.clone(), arguments)
        })?,
    )
}
fn group_table(lua: &Lua, parent: &Table, segment: &str) -> mlua::Result<Table> {
    if let Value::Table(table) = parent.get::<Value>(segment)? {
        return Ok(table);
    }
    let table = lua.create_table()?;
    parent.set(segment, table.clone())?;
    Ok(table)
}
fn invoke(
    lua: &Lua,
    pipe: &RefCell<LuaHostPipe>,
    method: String,
    arguments: Option<Table>,
) -> mlua::Result<Value> {
    let arguments = arguments.unwrap_or(lua.create_table()?);
    let encoded = convert::arguments(arguments)?;
    let reply = pipe
        .borrow_mut()
        .call(method.clone(), encoded)
        .map_err(|error| failure(format!("oblikovati.call({method:?}): {error}")))?;
    convert::reply(lua, &reply)
}
fn install_print(lua: &Lua, pipe: Rc<RefCell<LuaHostPipe>>) -> mlua::Result<()> {
    lua.globals().set(
        "print",
        lua.create_function(move |lua, arguments: mlua::MultiValue| {
            let stringify: Function = lua.globals().get("tostring")?;
            let mut parts = Vec::new();
            for argument in arguments {
                parts.push(
                    stringify
                        .call::<mlua::LuaString>(argument)?
                        .to_string_lossy(),
                );
            }
            pipe.borrow_mut().print(parts.join("\t"))
        })?,
    )
}
