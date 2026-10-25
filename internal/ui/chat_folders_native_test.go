package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Exercise actual GTK modifier handling and native DnD when an X11 test display
// and XTest are available. Other environments still run the store/layout tests.
func TestNativeChatMultiSelectAndFolderDrag(t *testing.T) {
	if os.Getenv("WHATSDOWN_NATIVE_INPUT_TESTS") != "1" {
		t.Skip("set WHATSDOWN_NATIVE_INPUT_TESTS=1 with an isolated X11 display")
	}
	if os.Getenv("DISPLAY") == "" || (os.Getenv("WAYLAND_DISPLAY") != "" && os.Getenv("GDK_BACKEND") != "x11") {
		t.Skip("requires X11")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("requires python3 and XTest")
	}
	c := newFolderSidebar(t)
	folder, err := c.store.CreateChatFolder(context.Background(), "Friends", "#3584e4", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	c.currentChat = &c.chats[0]
	window := gtk.NewWindow()
	window.SetTitle("WhatsDown sidebar native test")
	window.SetDefaultSize(400, 600)
	window.SetChild(c.chatsList)
	window.Present()
	t.Cleanup(window.Destroy)
	pumpSidebarEvents(150 * time.Millisecond)
	point := func(index int) []float64 {
		row := c.chatsList.RowAtIndex(index)
		x, y, ok := row.TranslateCoordinates(window, 24, float64(row.Height())/2)
		if !ok {
			t.Fatal("row has no window coordinates")
		}
		return []float64{x, y}
	}
	a, b, header := point(2), point(3), point(0)
	nativeSidebarActions(t, []any{[]any{"move", a[0], a[1]}, []any{"click", 1, false}})
	nativeSidebarActions(t, []any{[]any{"move", b[0], b[1]}, []any{"click", 1, true}})
	if len(c.selectedChatJIDs) != 2 {
		t.Fatalf("Ctrl-click selection = %v, want two chats", c.selectedChatJIDs)
	}
	if c.currentChat.JID != "0@a.test" {
		t.Fatal("Ctrl-click activated a different chat")
	}
	// A context menu on a selected chat must keep the entire selection.
	nativeSidebarActions(t, []any{[]any{"move", b[0], b[1]}, []any{"click", 3, false}})
	if len(c.selectedChatJIDs) != 2 {
		t.Fatal("right-click discarded multi-selection")
	}
	nativeSidebarActions(t, []any{[]any{"escape"}})
	// Drag the selected block onto the folder header.
	nativeSidebarActions(t, []any{[]any{"move", a[0], a[1]}, []any{"down"}, []any{"move", a[0] + 30, a[1]}, []any{"move", header[0], header[1]}, []any{"up"}})
	folders, err := c.store.ListChatFolders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].ID != folder.ID || len(folders[0].ChatJIDs) != 2 {
		t.Fatalf("native multi-chat drop did not move both chats: %+v", folders)
	}
	// Non-selectable folder headers must still respond to ordinary clicks.
	nativeSidebarActions(t, []any{[]any{"move", header[0], header[1]}, []any{"click", 1, false}})
	if !c.chatFolders[0].Collapsed {
		t.Fatal("clicking folder header did not collapse it")
	}
	nativeSidebarActions(t, []any{[]any{"move", header[0], header[1]}, []any{"click", 1, false}})
	if c.chatFolders[0].Collapsed {
		t.Fatal("clicking folder header did not expand it")
	}

	// Move only the second member above the first, then out to Other chats.
	c.chatsList.UnselectAll()
	first, second := point(1), point(2)
	first[1] -= float64(c.chatsList.RowAtIndex(1).Height()) / 4
	nativeSidebarActions(t, []any{[]any{"move", second[0], second[1]}, []any{"down"}, []any{"move", second[0] + 30, second[1]}, []any{"move", first[0], first[1]}, []any{"up"}})
	folders, _ = c.store.ListChatFolders(context.Background())
	if folders[0].ChatJIDs[0] != "0@b.test" {
		t.Fatalf("native member reorder failed: %v", folders[0].ChatJIDs)
	}
	member, outside := point(1), point(3)
	nativeSidebarActions(t, []any{[]any{"move", member[0], member[1]}, []any{"down"}, []any{"move", member[0] + 30, member[1]}, []any{"move", outside[0], outside[1]}, []any{"up"}})
	folders, _ = c.store.ListChatFolders(context.Background())
	if len(folders[0].ChatJIDs) != 1 || folders[0].ChatJIDs[0] != "0@a.test" {
		t.Fatalf("native ungroup drop failed: %v", folders[0].ChatJIDs)
	}

	other, err := c.store.CreateChatFolder(context.Background(), "Work", "#ff0000", nil)
	if err != nil {
		t.Fatal(err)
	}
	c.LoadChats()
	pumpSidebarEvents(100 * time.Millisecond)
	from, to := point(2), point(0)
	to[1] -= float64(c.chatsList.RowAtIndex(0).Height()) / 4
	nativeSidebarActions(t, []any{[]any{"move", from[0], from[1]}, []any{"down"}, []any{"move", from[0] + 30, from[1]}, []any{"move", to[0], to[1]}, []any{"up"}})
	folders, _ = c.store.ListChatFolders(context.Background())
	if folders[0].ID != other.ID || len(folders[0].ChatJIDs) != 0 {
		t.Fatalf("native folder reorder failed or nested a folder: %+v", folders)
	}
}

func pumpSidebarEvents(duration time.Duration) {
	ctx := glib.MainContextDefault()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		for ctx.Iteration(false) {
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func nativeSidebarActions(t *testing.T, actions []any) {
	nativeWindowActions(t, "WhatsDown sidebar native test", actions)
}

func nativeWindowActions(t *testing.T, title string, actions []any) {
	t.Helper()
	encoded, _ := json.Marshal(actions)
	cmd := exec.Command("python3", "-c", sidebarXTest, string(encoded), title)
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		pumpSidebarEvents(10 * time.Millisecond)
		select {
		case err := <-done:
			if err != nil {
				if cmd.ProcessState.ExitCode() == 77 {
					t.Skip("XTest unavailable: " + output.String())
				}
				t.Fatalf("native sidebar input: %v: %s", err, output.String())
			}
			pumpSidebarEvents(150 * time.Millisecond)
			return
		default:
			if time.Now().After(deadline) {
				cmd.Process.Kill()
				t.Fatal("native sidebar input timed out")
			}
		}
	}
}

const sidebarXTest = `
import ctypes as C, ctypes.util, json, sys, time
libx=ctypes.util.find_library('X11'); libxt=ctypes.util.find_library('Xtst')
if not libx or not libxt: sys.exit(77)
try:
 x=C.CDLL(libx); xt=C.CDLL(libxt)
except OSError:
 sys.exit(77)
P=C.c_void_p; U=C.c_ulong; I=C.c_int
x.XOpenDisplay.argtypes=[C.c_char_p]; x.XOpenDisplay.restype=P
d=x.XOpenDisplay(None)
if not d: sys.exit(77)
x.XDefaultRootWindow.argtypes=[P]; x.XDefaultRootWindow.restype=U
root=x.XDefaultRootWindow(d)
x.XQueryTree.argtypes=[P,U,C.POINTER(U),C.POINTER(U),C.POINTER(C.POINTER(U)),C.POINTER(C.c_uint)]
x.XFetchName.argtypes=[P,U,C.POINTER(C.c_char_p)]
x.XFree.argtypes=[P]
def find(w):
 name=C.c_char_p()
 if x.XFetchName(d,w,C.byref(name)) and name.value:
  match=name.value==sys.argv[2].encode(); x.XFree(name)
  if match: return w
 r=U(); p=U(); kids=C.POINTER(U)(); n=C.c_uint()
 if not x.XQueryTree(d,w,C.byref(r),C.byref(p),C.byref(kids),C.byref(n)): return 0
 children=[kids[i] for i in range(n.value)]
 if kids: x.XFree(kids)
 for child in children:
  found=find(child)
  if found: return found
 return 0
w=find(root)
if not w: raise RuntimeError('test window not found')
x.XTranslateCoordinates.argtypes=[P,U,U,I,I,C.POINTER(I),C.POINTER(I),C.POINTER(U)]
ox=I(); oy=I(); child=U(); x.XTranslateCoordinates(d,w,root,0,0,C.byref(ox),C.byref(oy),C.byref(child))
xt.XTestFakeMotionEvent.argtypes=[P,I,I,I,U]
xt.XTestFakeButtonEvent.argtypes=[P,C.c_uint,I,U]
xt.XTestFakeKeyEvent.argtypes=[P,C.c_uint,I,U]
x.XStringToKeysym.argtypes=[C.c_char_p]; x.XStringToKeysym.restype=U
x.XKeysymToKeycode.argtypes=[P,U]; x.XKeysymToKeycode.restype=C.c_uint
x.XSync.argtypes=[P,I]
ctrl=x.XKeysymToKeycode(d,x.XStringToKeysym(b'Control_L'))
esc=x.XKeysymToKeycode(d,x.XStringToKeysym(b'Escape'))
for action in json.loads(sys.argv[1]):
 kind=action[0]
 if kind=='move': xt.XTestFakeMotionEvent(d,-1,ox.value+int(action[1]),oy.value+int(action[2]),0)
 elif kind=='down': xt.XTestFakeButtonEvent(d,1,1,0)
 elif kind=='up': xt.XTestFakeButtonEvent(d,1,0,0)
 elif kind=='escape':
  xt.XTestFakeKeyEvent(d,esc,1,0); xt.XTestFakeKeyEvent(d,esc,0,0)
 elif kind=='click':
  if action[2]: xt.XTestFakeKeyEvent(d,ctrl,1,0)
  xt.XTestFakeButtonEvent(d,action[1],1,0); xt.XTestFakeButtonEvent(d,action[1],0,0)
  if action[2]: xt.XTestFakeKeyEvent(d,ctrl,0,0)
 x.XSync(d,0); time.sleep(0.12)
`
