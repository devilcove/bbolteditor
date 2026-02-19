package main

import (
	"encoding/json"
	"errors"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/textcore"
)

func createBucketDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Create Bucket")
	core.NewText(d).SetText("Parent Bucket")
	parent := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("Bucket Name")
	name := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			path := Path{}
			if parent.Text() != "" {
				path = stringToPath(parent.Text())
			}
			path = append(path, []byte(name.Text()))
			if _, err := CreateBucket(path); err != nil {
				core.ErrorDialog(button, err, "Create Bucket")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func deleteBucketDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Delete Bucket")
	core.NewText(d).SetText("Path")
	core.NewTextField(d).SetText(pathToString(node.Path))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := DeleteBucket(node.Path); err != nil {
				core.ErrorDialog(button, err, "Delete Bucket")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func emptyBucketDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Empty Bucket")
	core.NewText(d).SetText("Path")
	core.NewTextField(d).SetText(pathToString(node.Path))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := EmptyBucket(node.Path); err != nil {
				core.ErrorDialog(button, err, "Empty Bucket")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func addKeyDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Add Key")
	core.NewText(d).SetText("Parent Bucket")
	parent := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("Key Name")
	name := core.NewTextField(d)
	core.NewText(d).SetText("Key Value")
	value := textcore.NewEditor(d).Lines
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		core.NewButton(bar).SetText("validate json").OnClick(func(e events.Event) {
			if json.Valid(value.Text()) {
				core.MessageSnackbar(bar, "valid json")
			} else {
				core.MessageSnackbar(bar, "not valid json")
			}
		})
		d.AddOK(bar).OnClick(func(e events.Event) {
			path := stringToPath(parent.Text())
			if err := CreateKey(name.Text(), toJSON(value.Text()), path); err != nil {
				core.ErrorDialog(button, err, "Add Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func moveBucketDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Move Bucket")
	core.NewText(d).SetText("Current Path")
	current := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Path")
	newPath := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := MoveBucket(stringToPath(current.Text()),
				stringToPath(newPath.Text())); err != nil {
				core.ErrorDialog(button, err, "Add Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func moveKeyDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Move Key")
	core.NewText(d).SetText("Current Path")
	current := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Path")
	newPath := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := MoveKey(stringToPath(current.Text()),
				stringToPath(newPath.Text())); err != nil {
				core.ErrorDialog(button, err, "Add Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func deleteKeyDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Delete Key")
	core.NewText(d).SetText("Path")
	core.NewText(d).SetText(pathToString(node.Path))
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := DeleteKey(node.Path); err != nil {
				core.ErrorDialog(button, err, "Delete Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func renameKeyDialog(node TreeNode, button *core.Button) { //nolint:dupl
	d := core.NewBody("Rename Key")
	core.NewText(d).SetText("Path")
	currentPath := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Name")
	newName := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if strings.Contains(newName.Text(), " ") {
				core.ErrorDialog(button, errors.New("key name cannot contain spaces"), "Rename Key")
				return
			}
			if err := RenameKey(stringToPath(currentPath.Text()), newName.Text()); err != nil {
				core.ErrorDialog(button, err, "Rename Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func renameBucketDialog(node TreeNode, button *core.Button) { //nolint: dupl
	d := core.NewBody("Rename Bucket")
	core.NewText(d).SetText("Path")
	currentPath := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Name")
	newName := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if strings.Contains(newName.Text(), " ") {
				core.ErrorDialog(button, errors.New("bucket name cannot contain spaces"), "Rename Bucket")
				return
			}
			if err := RenameBucket(stringToPath(currentPath.Text()), newName.Text()); err != nil {
				core.ErrorDialog(button, err, "Rename Bucket")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func copyKeyDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Copy Key")
	core.NewText(d).SetText("Key Path")
	currentPath := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Path")
	newPath := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if err := CopyKey(stringToPath(currentPath.Text()), stringToPath(newPath.Text())); err != nil {
				core.ErrorDialog(button, err, "Copy Key")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}

func copyBucketDialog(node TreeNode, button *core.Button) {
	d := core.NewBody("Copy Bucket")
	core.NewText(d).SetText("Bucket Path")
	currentPath := core.NewTextField(d).SetText(pathToString(node.Path))
	core.NewText(d).SetText("New Path")
	newPath := core.NewTextField(d)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			node.Path = stringToPath(currentPath.Text())
			if err := CopyBucket(stringToPath(currentPath.Text()), stringToPath(newPath.Text())); err != nil {
				core.ErrorDialog(button, err, "Copy Bucket")
				return
			}
			reload()
		})
	})
	d.RunDialog(button)
}
