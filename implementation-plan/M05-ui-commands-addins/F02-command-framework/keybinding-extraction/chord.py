#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-2.0-only
"""Normalize vendor shortcut strings into one canonical chord syntax.

The target keybinding model is a per-profile chord map (modifiers + key), so every
vendor's spelling ("Ctrl + C", "Ctrl+Shift+C", "Shift+F3", "SpaceBar") collapses to
a single form: modifiers in fixed order Ctrl, Alt, Shift, then the key, joined by
'+'. Mouse/drag gestures are not keyboard chords and are flagged so they can be
excluded from a keyboard-binding profile.
"""
import re

_MODS = {
    "ctrl": "Ctrl",
    "control": "Ctrl",
    "alt": "Alt",
    "shift": "Shift",
    "cmd": "Cmd",
}
_MOD_ORDER = {"Ctrl": 0, "Alt": 1, "Shift": 2, "Cmd": 3}

# Vendor key spellings → canonical key token.
_KEY_ALIASES = {
    "spacebar": "Space",
    "space": "Space",
    "singlequote": "'",
    "esc": "Esc",
    "escape": "Esc",
    "del": "Delete",
    "ins": "Insert",
    "enter": "Enter",
    "return": "Enter",
    "prior": "PageUp",
    "next": "PageDown",
    "pageup": "PageUp",
    "pagedown": "PageDown",
}
_GESTURE = re.compile(r"\b(drag|click|mouse|wheel|scroll)\b", re.IGNORECASE)


def normalize(raw: str) -> dict | None:
    """Return {chord, mods, key, gesture} for a vendor shortcut, or None if empty.

    >>> normalize("Ctrl + Shift + C")["chord"]
    'Ctrl+Shift+C'
    >>> normalize("SpaceBar")["chord"]
    'Space'
    >>> normalize("Shift + Drag")["gesture"]
    True
    """
    if raw is None:
        return None
    text = raw.strip()
    if not text:
        return None
    gesture = bool(_GESTURE.search(text))
    parts = [p.strip() for p in re.split(r"\s*\+\s*", text) if p.strip()]
    mods, keys = [], []
    for part in parts:
        low = part.lower()
        if low in _MODS:
            mod = _MODS[low]
            if mod not in mods:
                mods.append(mod)
        elif _GESTURE.match(part):
            continue  # the gesture verb itself isn't a key
        else:
            keys.append(_canon_key(part))
    mods.sort(key=lambda m: _MOD_ORDER[m])
    key = keys[-1] if keys else ""  # last token is the activating key
    chord = "+".join(mods + ([key] if key else []))
    return {"chord": chord, "mods": mods, "key": key, "gesture": gesture}


def _canon_key(token: str) -> str:
    """Canonicalize a single key token (function keys upper, letters upper, named keys mapped)."""
    low = token.lower()
    if low in _KEY_ALIASES:
        return _KEY_ALIASES[low]
    if re.fullmatch(r"f\d{1,2}", low):  # F1..F12
        return low.upper()
    if len(token) == 1:
        return token.upper()
    return token  # multi-char alias (Inventor) or named key kept verbatim


if __name__ == "__main__":
    import doctest

    doctest.testmod(verbose=False)
    print("chord.py self-test ok")
