import gi,time
gi.require_version('Atspi','2.0')
from gi.repository import Atspi
Atspi.init()
def controls():
    desktop=Atspi.get_desktop(0)
    for i in range(desktop.get_child_count()):
        app=desktop.get_child_at_index(i)
        for j in range(app.get_child_count()):
            window=app.get_child_at_index(j)
            children=[window.get_child_at_index(k) for k in range(window.get_child_count())]
            names=[c.get_name() for c in children]
            if 'Minimize' in names and 'Close' in names:
                return window,children
    return None,None
def await_controls(name):
    for _ in range(600):
        window,buttons=controls()
        if buttons and any(b.get_name()==name for b in buttons): return window,buttons
        time.sleep(.1)
    raise AssertionError('missing button '+name)
window,buttons=await_controls('Maximize')
for button in buttons:
    bounds=button.get_component_iface().get_extents(Atspi.CoordType.SCREEN)
    print(button.get_name(),button.get_role_name(),bounds.x,bounds.y,bounds.width,bounds.height,flush=True)
    assert button.get_role_name()=='button' and bounds.width>0 and bounds.height>0
max_button=next(b for b in buttons if b.get_name()=='Maximize')
assert max_button.get_action_iface().do_action(0)
window,buttons=await_controls('Restore')
print('Maximize action produced Restore',flush=True)
restore=next(b for b in buttons if b.get_name()=='Restore')
assert restore.get_action_iface().do_action(0)
window,buttons=await_controls('Maximize')
close=next(b for b in buttons if b.get_name()=='Close')
assert close.get_action_iface().do_action(0)
print('Close action sent',flush=True)
