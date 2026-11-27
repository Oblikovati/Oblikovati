// SPDX-License-Identifier: GPL-2.0-only
// Lua 5.1 keywords and the globals available after vm::strip_globals and host binding.
// tests/editor_completion.rs checks every builtin against the isolated worker.
pub(crate) const KEYWORDS: &[&str] = &[
    "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "if", "in", "local",
    "nil", "not", "or", "repeat", "return", "then", "true", "until", "while",
];
pub(crate) const BUILTINS: &[&str] = &[
    "_G",
    "_VERSION",
    "assert",
    "coroutine",
    "error",
    "gcinfo",
    "getfenv",
    "getmetatable",
    "ipairs",
    "math",
    "next",
    "oblikovati",
    "pairs",
    "pcall",
    "print",
    "rawequal",
    "rawget",
    "rawset",
    "select",
    "setfenv",
    "setmetatable",
    "string",
    "table",
    "tonumber",
    "tostring",
    "type",
    "unpack",
    "xpcall",
];

pub(crate) fn is_identifier(character: char) -> bool {
    character.is_ascii_alphanumeric() || character == '_'
}
