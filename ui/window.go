// ui/window.go

package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/Brainbeer/gensyn/portage"
)

func StartUI() {
	a := app.New()
	w := a.NewWindow("gensyn")
	w.Resize(fyne.NewSize(800, 600))

	categories, err := portage.GetCategories()
	if err != nil {
		categories = nil
	}

	categoryNames := []string{}
	for _, cat := range categories {
		categoryNames = append(categoryNames, cat.Name)
	}

	tree := widget.NewTree(
		func(uid string) []string {
			if uid == "" {
				return categoryNames
			}
			packages, err := portage.GetPackages(uid)
			if err != nil {
				return []string{}
			}
			ids := []string{}
			for _, pkg := range packages {
				ids = append(ids, uid+"/"+pkg.Name)
			}
			return ids
		},
		func(uid string) bool {
			return !strings.Contains(uid, "/")
		},
		func(branch bool) fyne.CanvasObject {
			if branch {
				return widget.NewLabel("")
			}
			t := canvas.NewText("", nil)
			t.TextSize = 11
			return t
		},
		func(uid string, branch bool, o fyne.CanvasObject) {
			if branch {
				o.(*widget.Label).SetText(uid)
			} else {
				parts := strings.SplitN(uid, "/", 2)
				o.(*canvas.Text).Text = parts[1]
				o.Refresh()
			}
		},
	)

	// Toolbar (top right)
	toolbar := widget.NewLabel("[ Toolbar ]")

	// Package list (center top)
	packageNames := []string{}
	packageList := widget.NewList(
		func() int { return len(packageNames) },
		func() fyne.CanvasObject {
			t := canvas.NewText("", nil)
			t.TextSize = 11
			return t
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			o.(*canvas.Text).Text = packageNames[id]
			o.Refresh()
		},
	)

	tree.OnSelected = func(uid string) {
		if !strings.Contains(uid, "/") {
			return
		}
		parts := strings.SplitN(uid, "/", 2)
		category, pkg := parts[0], parts[1]

		files, err := portage.GetPackageFiles(category, pkg)
		if err != nil {
			packageNames = []string{"Error reading package directory"}
			packageList.Refresh()
			return
		}

		installed := portage.GetInstalledVersion(category, pkg)

		packageNames = []string{}
		if installed != "" {
			packageNames = append(packageNames, "Installed: "+installed)
		} else {
			packageNames = append(packageNames, "Not installed")
		}
		skip := map[string]bool{"Manifest": true, "files": true, "metadata.xml": true}
		for _, f := range files {
			if !skip[f] {
				packageNames = append(packageNames, f)
			}
		}
		packageList.Refresh()
	}

	// Description (center bottom)
	description := widget.NewLabel("[ Description ]")

	// Output terminal (right)
	output := widget.NewLabel("[ Output ]")

	// Center section: package list over description
	centerSection := container.NewVSplit(packageList, description)
	centerSection.SetOffset(0.6)

	// Lower section: center over output terminal
	lowerSection := container.NewHSplit(centerSection, output)
	lowerSection.SetOffset(0.4)

	// Right section: toolbar on top, lower section below
	rightSection := container.NewVSplit(toolbar, lowerSection)
	rightSection.SetOffset(0.2)

	// Main layout: tree on left, right section on right
	main := container.NewHSplit(tree, rightSection)
	main.SetOffset(0.19)

	w.SetContent(main)
	w.ShowAndRun()
}
