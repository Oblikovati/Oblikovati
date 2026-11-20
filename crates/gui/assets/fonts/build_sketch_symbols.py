# SPDX-License-Identifier: GPL-2.0-only
"""Build the original sketch symbol fallback font for the ImGui atlas."""
from pathlib import Path

from fontTools.fontBuilder import FontBuilder
from fontTools.pens.ttGlyphPen import TTGlyphPen


def diameter_glyph():
    """Draw a circular ring and an ascending stroke as a single font glyph."""
    pen = TTGlyphPen(None)
    pen.moveTo((70, 350))
    pen.qCurveTo((70, 660), (380, 660))
    pen.qCurveTo((690, 660), (690, 350))
    pen.qCurveTo((690, 40), (380, 40))
    pen.qCurveTo((70, 40), (70, 350))
    pen.closePath()
    pen.moveTo((145, 350))
    pen.qCurveTo((145, 115), (380, 115))
    pen.qCurveTo((615, 115), (615, 350))
    pen.qCurveTo((615, 585), (380, 585))
    pen.qCurveTo((145, 585), (145, 350))
    pen.closePath()
    pen.moveTo((100, 39))
    pen.lineTo((619, 708))
    pen.lineTo((677, 663))
    pen.lineTo((158, -6))
    pen.closePath()
    return pen.glyph()


def build_font(destination: Path) -> None:
    """Write a repeatable TrueType font with one U+2300 glyph."""
    font = FontBuilder(1000, isTTF=True)
    font.setupGlyphOrder([".notdef", "diameter"])
    font.setupCharacterMap({0x2300: "diameter"})
    font.setupGlyf({".notdef": TTGlyphPen(None).glyph(), "diameter": diameter_glyph()})
    font.setupHorizontalMetrics({".notdef": (760, 0), "diameter": (760, 70)})
    font.setupHorizontalHeader(ascent=800, descent=-200)
    font.setupNameTable({"familyName": "Oblikovati Sketch Symbols", "styleName": "Regular",
        "uniqueFontIdentifier": "OblikovatiSketchSymbols-Regular", "fullName": "Oblikovati Sketch Symbols",
        "psName": "OblikovatiSketchSymbols-Regular"})
    font.setupOS2(sTypoAscender=800, sTypoDescender=-200, usWinAscent=800, usWinDescent=200)
    font.setupPost()
    font.font.recalcTimestamp = False
    font.font["head"].created = font.font["head"].modified = 3856896000
    font.save(destination)


if __name__ == "__main__":
    build_font(Path(__file__).with_name("OblikovatiSketchSymbols.ttf"))
