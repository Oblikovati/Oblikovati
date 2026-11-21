# Sketch Text 37 height correction

The pinned and live `text-37` scenario at SHA-256
`0e8f627cd0eb991bc18c2615f759ca92be1e1c121eb0c7a819b68f2b08706a3d`
instructs the user to dismiss the zero-height warning, enter `6`, and
press Tab. The next step expects `6.10 mm`.

The [native replay](../receipts/2026-11-21-sketch-text-height-fraction-3907/README.md)
uses the recorded Arial fraction at 6.10 mm and 47 dimensions needed.
A zero input retains the previous valid preview. The explicit new input
`6` produces a 6 mm preview and leaves the font Arial. Cancel restores the
saved 6.10 mm fraction and 47 dimensions needed. The replay includes a
portable native part, captures, and test output.

Check the recording to determine whether the intended action typed 6,
entered 6.10, or restored the previous value. Match the intermediate
height assertion to that action. The final Cancel check can retain 6.10 mm
and 47 dimensions needed for all three alternatives.
