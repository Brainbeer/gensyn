// ui/window.go

package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/Brainbeer/gensyn/portage"
)

// compactHBox lays out children horizontally with a 2px gap instead of the default theme padding
type compactHBox struct{}

func (c *compactHBox) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	for _, o := range objects {
		w := o.MinSize().Width
		o.Resize(fyne.NewSize(w, size.Height))
		o.Move(fyne.NewPos(x, 0))
		x += w + 2
	}
}

func (c *compactHBox) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w := float32(0)
	h := float32(0)
	for i, o := range objects {
		ms := o.MinSize()
		if i > 0 {
			w += 2
		}
		w += ms.Width
		if ms.Height > h {
			h = ms.Height
		}
	}
	return fyne.NewSize(w, h)
}

func StartUI() {
	a := app.New()
	w := a.NewWindow("gensyn")
	w.Resize(fyne.NewSize(1920, 1080))

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

	// Toolbar
	// Keyword search entry fills available width, same height as buttons
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search...")

	// 10px gap between search entry and buttons
	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(10, 1))

	// Sync and Install buttons stacked vertically on the far right (56px = ~7% of 800px)
	syncButton := widget.NewButton("Sync", func() {})
	installButton := widget.NewButton("Install", func() {})

	buttonStack := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(56, 30)),
		syncButton,
		installButton,
	)

	// Toggle for search mode, with "|" separator to distinguish from operation radio buttons
	searchToggle := widget.NewRadioGroup([]string{"Package", "Command"}, nil)
	searchToggle.SetSelected("Package")
	searchToggle.Horizontal = true

	separator := widget.NewLabel("|")

	// Operation radio buttons
	operationRadio := widget.NewRadioGroup([]string{"No Flag", "-p (pretend)", "-f (fetch)", "-uvNDU (Update)", "Custom"}, nil)
	operationRadio.SetSelected("No Flag")
	operationRadio.Horizontal = true

	// Custom flags entry — only enabled when Custom is selected
	customEntry := widget.NewEntry()
	customEntry.SetPlaceHolder("flags...")
	customEntry.Disable()
	customEntryContainer := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(130, 30)),
		customEntry,
	)

	operationRadio.OnChanged = func(val string) {
		if val == "Custom" {
			customEntry.Enable()
		} else {
			customEntry.Disable()
		}
	}

	toggleRow := container.New(&compactHBox{}, searchToggle, separator, operationRadio, customEntryContainer)

	leftSection := container.NewVBox(searchEntry, toggleRow)
	toolbar := container.NewBorder(nil, nil, nil, container.NewHBox(gap, buttonStack), leftSection)

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
	rightSection := container.NewBorder(toolbar, nil, nil, nil, lowerSection)

	// Main layout: tree on left, right section on right
	main := container.NewHSplit(tree, rightSection)
	main.SetOffset(0.12)

	w.SetContent(main)
	w.ShowAndRun()
}
