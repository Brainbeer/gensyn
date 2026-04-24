// ui/window.go

package ui

import (
	"bufio"
	"image/color"
	"os/exec"
	"regexp"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/Brainbeer/gensyn/portage"
)

// ansiEscape strips ANSI terminal escape sequences from a string
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

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

// stripVersion removes the version suffix from a Portage package atom.
// e.g. "git-2.45.1" -> "git", "lib-foo-1.2.3-r1" -> "lib-foo"
func stripVersion(pkgWithVersion string) string {
	parts := strings.Split(pkgWithVersion, "-")
	end := len(parts)
	for end > 1 {
		seg := parts[end-1]
		if seg == "" {
			end--
			continue
		}
		if seg[0] == 'r' && len(seg) > 1 && unicode.IsDigit(rune(seg[1])) {
			end--
			continue
		}
		if unicode.IsDigit(rune(seg[0])) {
			end--
			continue
		}
		break
	}
	return strings.Join(parts[:end], "-")
}

// runCommand streams the output of cmd line by line into the output widget and auto-scrolls.
func runCommand(cmd *exec.Cmd, output *widget.RichText, scroll *container.Scroll, syncButton, installButton *widget.Button) {
	pipe, err := cmd.StderrPipe()
	if err != nil {
		fyne.Do(func() {
			output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
			output.Refresh()
		})
		return
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fyne.Do(func() {
			output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
			output.Refresh()
		})
		return
	}

	if err := cmd.Start(); err != nil {
		fyne.Do(func() {
			output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error starting command: " + err.Error() + "\n"})
			output.Refresh()
			syncButton.Enable()
			installButton.Enable()
		})
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := stripANSI(scanner.Text())
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: line + "\n"})
				output.Refresh()
				scroll.ScrollToBottom()
			})
		}
	}()

	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := stripANSI(scanner.Text())
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: line + "\n"})
				output.Refresh()
				scroll.ScrollToBottom()
			})
		}
	}()

	go func() {
		err := cmd.Wait()
		fyne.Do(func() {
			if err != nil {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "\n[exited with error: " + err.Error() + "]\n"})
			} else {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "\n[done]\n"})
			}
			output.Refresh()
			scroll.ScrollToBottom()
			syncButton.Enable()
			installButton.Enable()
		})
	}()
}

// promptAndRun shows a sudo password dialog if needed, then fires runCommand.
func promptAndRun(args []string, needsSudo bool, output *widget.RichText, scroll *container.Scroll, syncButton, installButton *widget.Button, w fyne.Window) {
	syncButton.Disable()
	installButton.Disable()
	output.Segments = []widget.RichTextSegment{}
	output.Refresh()

	if !needsSudo {
		cmd := exec.Command(args[0], args[1:]...)
		runCommand(cmd, output, scroll, syncButton, installButton)
		return
	}

	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		output.Segments = append(output.Segments, &widget.TextSegment{Text: "sudo not found. Have you installed app-admin/sudo?\n"})
		output.Refresh()
		syncButton.Enable()
		installButton.Enable()
		return
	}

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("sudo password...")

	var d dialog.Dialog

	confirm := func() {
		d.Hide()
		password := passwordEntry.Text
		sudoArgs := append([]string{"-S", "--"}, args...)
		cmd := exec.Command(sudoPath, sudoArgs...)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
			output.Refresh()
			syncButton.Enable()
			installButton.Enable()
			return
		}
		go func() {
			defer stdin.Close()
			stdin.Write([]byte(password + "\n"))
		}()
		runCommand(cmd, output, scroll, syncButton, installButton)
	}

	cancel := func() {
		syncButton.Enable()
		installButton.Enable()
	}

	passwordEntry.OnSubmitted = func(_ string) {
		confirm()
	}

	d = dialog.NewCustomConfirm("sudo password required", "Run", "Cancel",
		passwordEntry,
		func(confirmed bool) {
			if confirmed {
				confirm()
			} else {
				cancel()
			}
		},
		w,
	)
	d.Show()
}

func StartUI() {
	a := app.New()
	a.SetIcon(resourceIconPng)
	w := a.NewWindow("gensyn")
	w.Resize(fyne.NewSize(1920, 1080))

	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Quit", func() { a.Quit() }),
	)
	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Preferences", func() {
			dialog.ShowInformation("Preferences", "Preferences coming soon.", w)
		}),
	)
	aboutMenu := fyne.NewMenu("About",
		fyne.NewMenuItem("About gensyn", func() {
			dialog.ShowInformation("About", "gensyn\nA Synaptic-like package manager for Gentoo.", w)
		}),
	)
	w.SetMainMenu(fyne.NewMainMenu(fileMenu, editMenu, aboutMenu))

	categories, err := portage.GetCategories()
	if err != nil {
		categories = nil
	}

	categoryNames := []string{}
	for _, cat := range categories {
		categoryNames = append(categoryNames, cat.Name)
	}

	description := widget.NewRichTextFromMarkdown("")
	descriptionScroll := container.NewVScroll(description)

	output := widget.NewRichText()
	output.Wrapping = fyne.TextWrapWord
	outputScroll := container.NewVScroll(output)

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

	selectedUID := ""

	var tree *widget.Tree

	selectInTree := func(uid string) {
		parts := strings.SplitN(uid, "/", 2)
		if len(parts) != 2 {
			return
		}
		tree.OpenBranch(parts[0])
		tree.Select(uid)
		tree.ScrollTo(uid)
	}

	tree = widget.NewTree(
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

	tree.OnSelected = func(uid string) {
		if !strings.Contains(uid, "/") {
			return
		}
		selectedUID = uid
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

		desc := portage.GetDescription(category, pkg)
		if desc == "" {
			desc = "_No description available._"
		}
		description.ParseMarkdown(desc)
	}

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search...")

	gap := canvas.NewRectangle(color.Transparent)
	gap.SetMinSize(fyne.NewSize(10, 1))

	syncButton := widget.NewButton("Sync", nil)
	installButton := widget.NewButton("Install", nil)

	buttonStack := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(56, 30)),
		syncButton,
		installButton,
	)

	searchToggle := widget.NewRadioGroup([]string{"Package", "Command"}, nil)
	searchToggle.SetSelected("Package")
	searchToggle.Horizontal = true

	separator := widget.NewLabel("|")

	operationRadio := widget.NewRadioGroup([]string{"No Flag", "-p (pretend)", "-f (fetch)", "-uvNDU (Update)", "Custom"}, nil)
	operationRadio.SetSelected("No Flag")
	operationRadio.Horizontal = true

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

	searchEntry.OnSubmitted = func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}

		if searchToggle.Selected == "Command" {
			eqPath, err := exec.LookPath("equery")
			if err != nil {
				dialog.ShowInformation("gentoolkit required",
					"equery not found. Have you installed app-portage/gentoolkit?", w)
				return
			}
			searchEntry.Disable()
			go func() {
				out, err := exec.Command(eqPath, "belongs", query).Output()
				fyne.Do(func() {
					searchEntry.Enable()
					if err != nil || strings.TrimSpace(string(out)) == "" {
						dialog.ShowInformation("Not found",
							"No package found owning: "+query, w)
						return
					}
					seen := map[string]bool{}
					var uids []string
					for _, line := range strings.Split(string(out), "\n") {
						line = strings.TrimSpace(line)
						if line == "" || strings.HasPrefix(line, "*") {
							continue
						}
						atom := strings.SplitN(line, " ", 2)[0]
						parts := strings.SplitN(atom, "/", 2)
						if len(parts) != 2 {
							continue
						}
						uid := parts[0] + "/" + stripVersion(parts[1])
						if !seen[uid] {
							seen[uid] = true
							uids = append(uids, uid)
						}
					}
					if len(uids) == 0 {
						dialog.ShowInformation("Not found",
							"No package found owning: "+query, w)
						return
					}
					if len(uids) == 1 {
						selectInTree(uids[0])
						return
					}
					var d dialog.Dialog
					list := widget.NewList(
						func() int { return len(uids) },
						func() fyne.CanvasObject { return widget.NewLabel("") },
						func(id widget.ListItemID, o fyne.CanvasObject) {
							o.(*widget.Label).SetText(uids[id])
						},
					)
					list.OnSelected = func(id widget.ListItemID) {
						d.Hide()
						selectInTree(uids[id])
					}
					lc := container.NewVScroll(list)
					lc.SetMinSize(fyne.NewSize(400, 300))
					d = dialog.NewCustom("Owning packages", "Cancel", lc, w)
					d.Show()
				})
			}()
			return
		}

		searchEntry.Disable()
		go func() {
			lower := strings.ToLower(query)
			var matches []string
			for _, cat := range categoryNames {
				pkgs, err := portage.GetPackages(cat)
				if err != nil {
					continue
				}
				for _, pkg := range pkgs {
					if strings.Contains(strings.ToLower(pkg.Name), lower) {
						matches = append(matches, cat+"/"+pkg.Name)
					}
				}
			}

			fyne.Do(func() {
				searchEntry.Enable()
				if len(matches) == 0 {
					dialog.ShowInformation("No results",
						"No packages found matching: "+query, w)
					return
				}
				if len(matches) == 1 {
					selectInTree(matches[0])
					return
				}
				var d dialog.Dialog
				list := widget.NewList(
					func() int { return len(matches) },
					func() fyne.CanvasObject { return widget.NewLabel("") },
					func(id widget.ListItemID, o fyne.CanvasObject) {
						o.(*widget.Label).SetText(matches[id])
					},
				)
				list.OnSelected = func(id widget.ListItemID) {
					d.Hide()
					selectInTree(matches[id])
				}
				listContainer := container.NewVScroll(list)
				listContainer.SetMinSize(fyne.NewSize(400, 400))
				d = dialog.NewCustom("Search results", "Cancel", listContainer, w)
				d.Show()
			})
		}()
	}

	toggleRow := container.New(&compactHBox{}, searchToggle, separator, operationRadio, customEntryContainer)
	leftSection := container.NewVBox(searchEntry, toggleRow)
	toolbar := container.NewBorder(nil, nil, nil, container.NewHBox(gap, buttonStack), leftSection)

	syncButton.OnTapped = func() {
		promptAndRun([]string{"emerge", "--sync"}, true, output, outputScroll, syncButton, installButton, w)
	}

	installButton.OnTapped = func() {
		op := operationRadio.Selected

		if op == "-uvNDU (Update)" {
			promptAndRun([]string{"emerge", "-uvNDU", "world"}, true, output, outputScroll, syncButton, installButton, w)
			return
		}

		if selectedUID == "" {
			dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
			return
		}

		needsSudo := op != "-p (pretend)"

		var flags []string
		switch op {
		case "No Flag":
			flags = []string{}
		case "-p (pretend)":
			flags = []string{"-p"}
		case "-f (fetch)":
			flags = []string{"-f"}
		case "Custom":
			raw := strings.TrimSpace(customEntry.Text)
			if raw == "" {
				dialog.ShowInformation("No flags", "Please enter custom flags or select a different mode.", w)
				return
			}
			flags = strings.Fields(raw)
		}

		args := append([]string{"emerge"}, flags...)
		args = append(args, selectedUID)
		promptAndRun(args, needsSudo, output, outputScroll, syncButton, installButton, w)
	}

	centerSection := container.NewVSplit(packageList, descriptionScroll)
	centerSection.SetOffset(0.6)

	lowerSection := container.NewHSplit(centerSection, outputScroll)
	lowerSection.SetOffset(0.24)

	rightSection := container.NewBorder(toolbar, nil, nil, nil, lowerSection)

	main := container.NewHSplit(tree, rightSection)
	main.SetOffset(0.12)

	w.SetContent(main)
	w.ShowAndRun()
}
