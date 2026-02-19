package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/filetree"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	berrors "go.etcd.io/bbolt/errors"
)

var (
	app           *core.Body
	panes         *core.Splits
	selectedNode  TreeNode
	bucketButton  *core.Button
	keyButton     *core.Button
	databaseInUse = "Database file is locked. Is the database in use by another application?"
)

func main() { //nolint:funlen
	log.SetFlags(log.Lshortfile | log.Ltime)
	app = core.NewBody("BboltEditor")
	dbfile := "test.db"
	if len(os.Args) == 2 {
		dbfile = os.Args[1]
	}
	if err := openDB(dbfile); err != nil {
		if errors.Is(err, berrors.ErrTimeout) {
			space := core.NewSpace(app)
			core.MessageDialog(space, "Database in use by another application", "Database Error")
		} else {
			log.Fatal(err)
		}
	}
	nodes := getNodes()
	core.NewToolbar(app).Maker(func(p *tree.Plan) {
		tree.Add(p, func(w *core.Button) {
			w.SetText("File").OnClick(func(e events.Event) {
				current, _ := os.Getwd()
				d := core.NewBody("File")
				ft := filetree.NewTree(d).OpenPath(current)
				selected := ""
				ft.OnSelect(func(e events.Event) {
					ft.SelectedFunc(func(n *filetree.Node) {
						selected = string(n.Filepath)
					})
				})
				d.AddBottomBar(func(bar *core.Frame) {
					d.AddCancel(bar)
					core.NewButton(bar).SetText("Open Parent").OnClick(func(e events.Event) {
						current = filepath.Dir(current)
						ft.DeleteChildren()
						ft.OpenPath(current)
						d.Update()
					})
					d.AddOK(bar).OnClick(func(e events.Event) {
						log.Println("open file ", selected)
						if err := loadFile(selected); err != nil {
							if errors.Is(err, berrors.ErrTimeout) {
								core.MessageDialog(d, databaseInUse, "Open Database")
							} else {
								core.ErrorDialog(d, err, "Open File")
							}
						}
					})
				})
				d.RunDialog(w)
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Bucket").SetMenu(bucketContext)
			bucketButton = w
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Key").SetMenu(keyContext).SetEnabled(false)
			keyButton = w
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Settings").OnClick(func(e events.Event) {
				core.SettingsWindow()
			})
		})
		tree.Add(p, func(w *core.Button) {
			w.SetText("Quit").OnClick(func(e events.Event) {
				app.Close()
			})
		})
	})
	core.NewSpace(app)
	panes = core.NewSplits(app).SetSplits(.3, .7) //nolint:mnd
	left := core.NewFrame(panes)
	core.NewFrame(panes)

	tr := core.NewTree(left).SetText(dbfile)
	addNodes(tr, nodes)
	tr.Scene.ContextMenus = nil
	tr.ContextMenus = nil
	tr.ContextMenus = append(tr.ContextMenus, mainContext)
	tr.SetReadOnly(true)

	app.RunMainWindow()
}

func addNodes(t *core.Tree, nodes []*TreeNode) {
	for _, node := range nodes {
		item := core.NewTree(t).SetText(string(node.Name))
		item.SetReadOnly(true)
		item.SetClosed(true)
		item.ContextMenus = nil
		if node.IsBucket {
			item.SetIcon(icons.Colors)
			item.ContextMenus = append(item.ContextMenus, bucketContext)
		} else {
			item.SetIcon(icons.KeyFill)
			item.ContextMenus = append(item.ContextMenus, keyContext)
		}
		name := []string{}
		for _, part := range node.Path {
			name = append(name, string(part))
		}
		item.Name = strings.Join(name, "/")
		item.OnSelect(func(e events.Event) {
			updateDetails(item.Name)
		})
		addNodes(item, node.Children)
	}
}

func mainContext(m *core.Scene) {
	button := core.NewButton(m).SetText("Create Bucket")
	button.OnClick(func(e events.Event) {
		createBucketDialog(TreeNode{}, button)
	})
}

func keyContext(m *core.Scene) {
	button := core.NewButton(m)
	button.SetText("Delete Key").OnClick(func(e events.Event) {
		deleteKeyDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Move Key").OnClick(func(e events.Event) {
		moveKeyDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Rename Key").OnClick(func(e events.Event) {
		renameKeyDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Copy Key").OnClick(func(e events.Event) {
		copyKeyDialog(getNode(m), button)
	})
}

func bucketContext(m *core.Scene) {
	button := core.NewButton(m).SetText("Create Bucket")
	button.OnClick(func(e events.Event) {
		createBucketDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Delete Bucket").OnClick(func(e events.Event) {
		deleteBucketDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Empty Bucket").OnClick(func(e events.Event) {
		emptyBucketDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Add Key").OnClick(func(e events.Event) {
		addKeyDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Move Bucket").OnClick(func(e events.Event) {
		moveBucketDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Rename Bucket").OnClick(func(e events.Event) {
		renameBucketDialog(getNode(m), button)
	})
	core.NewButton(m).SetText("Copy Bucket").OnClick(func(e events.Event) {
		copyBucketDialog(getNode(m), button)
	})
}

func updateDetails(item string) {
	node, ok := nodeMap[item]
	if !ok {
		log.Println("invalid node", item)
	}
	selectedNode = node
	panes.AsFrame().DeleteChildAt(1)
	details := core.NewFrame(panes)
	if node.IsBucket {
		core.NewText(details).SetText("Bucket:")
		bucketButton.SetEnabled(true)
		keyButton.SetEnabled(false)
	} else {
		core.NewText(details).SetText("Key:")
		bucketButton.SetEnabled(false)
		keyButton.SetEnabled(true)
	}
	core.NewSpace(details)
	core.NewText(details).SetText("Path:" + item)
	core.NewText(details).SetText("Name:" + string(node.Name))
	if !node.IsBucket {
		var reset *core.Button
		core.NewSpace(details)
		value := pretty(node.Value)
		te := textcore.NewEditor(details)
		buf := te.Lines.SetText(value)
		te.OnKeyChord(func(e events.Event) {
			log.Println("changed", e.KeyChord())
			if e.KeyCode() == key.CodeReturnEnter {
				reset.SetFocus()
			}
		})
		frame := core.NewFrame(details)
		reset = core.NewButton(frame).SetText("Reset")
		reset.OnClick(func(e events.Event) {
			buf.SetText(node.Value)
		})
		core.NewButton(frame).SetText("Validate Json").OnClick(func(e events.Event) {
			if json.Valid(buf.Text()) {
				core.MessageSnackbar(details, "valid json")
			} else {
				core.MessageSnackbar(details, "not valid json")
			}
		})
		core.NewButton(frame).SetText("Update").OnClick(func(e events.Event) {
			if err := UpdateKey(node, toJSON(buf.Text())); err != nil {
				core.ErrorDialog(details, err, "Update Key")
				return
			}
			reload()
		})
	}
	app.Update()
}

func pretty(s []byte) []byte {
	var data bytes.Buffer
	if err := json.Indent(&data, s, "", "\t"); err != nil {
		return s
	}
	return data.Bytes()
}

func toJSON(orig []byte) []byte { //nolint:varnamelen
	var temp any
	if err := json.Unmarshal(orig, &temp); err != nil {
		return orig
	}
	bytes, err := json.Marshal(temp)
	if err != nil {
		return orig
	}
	return bytes
}
