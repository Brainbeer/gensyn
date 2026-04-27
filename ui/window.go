// ui/window.go

package ui

import (
	"bufio"
	"fmt"
	"image/color"
	"os/exec"
	"regexp"
	"strconv"
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

// operationOptions is the single source of truth for the emerge flag radio buttons.
// It is used both in the toolbar and in the Preferences dialog.
var operationOptions = []string{
	"No Flag",
	"-p (pretend)",
	"-f (fetch)",
	"-uvNDU (Update)",
	"-C (Uninstall)",
	"--depclean",
	"Custom",
}

// runCommand streams the output of cmd line by line into the output widget and auto-scrolls.
// onSuccess is called (on the main thread) when the command exits without error; may be nil.
func runCommand(cmd *exec.Cmd, output *widget.RichText, scroll *container.Scroll, syncButton, installButton *widget.Button, onSuccess func()) {
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
				if onSuccess != nil {
					onSuccess()
				}
			}
			output.Refresh()
			scroll.ScrollToBottom()
			syncButton.Enable()
			installButton.Enable()
		})
	}()
}

// promptAndRun shows a sudo password dialog if needed, then fires runCommand.
// onSuccess is passed through to runCommand; may be nil.
func promptAndRun(args []string, needsSudo bool, output *widget.RichText, scroll *container.Scroll, syncButton, installButton *widget.Button, w fyne.Window, onSuccess func()) {
	syncButton.Disable()
	installButton.Disable()
	if Current.ClearOutput {
		output.Segments = []widget.RichTextSegment{}
		output.Refresh()
	}

	if !needsSudo {
		cmd := exec.Command(args[0], args[1:]...)
		runCommand(cmd, output, scroll, syncButton, installButton, onSuccess)
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
		runCommand(cmd, output, scroll, syncButton, installButton, onSuccess)
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

// commitToPortage appends entryText to /etc/portage/<dir>/<pkgname> via sudo tee -a.
// The package name is parsed from the atom at the start of entryText (the part after "/").
func commitToPortage(entryText, dir string, output *widget.RichText, scroll *container.Scroll, commitButton *widget.Button, w fyne.Window) {
	entryText = strings.TrimSpace(entryText)
	if entryText == "" {
		dialog.ShowInformation("Empty entry", "Please enter a package atom.", w)
		return
	}

	fields := strings.Fields(entryText)
	atom := fields[0]
	parts := strings.SplitN(atom, "/", 2)
	if len(parts) != 2 || parts[1] == "" {
		dialog.ShowInformation("Invalid atom", "Entry must begin with category/package.", w)
		return
	}
	pkgName := parts[1]
	targetPath := "/etc/portage/" + dir + "/" + pkgName

	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		output.Segments = append(output.Segments, &widget.TextSegment{Text: "sudo not found. Have you installed app-admin/sudo?\n"})
		output.Refresh()
		return
	}

	commitButton.Disable()
	if Current.ClearOutput {
		output.Segments = []widget.RichTextSegment{}
		output.Refresh()
	}

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("sudo password...")

	var d dialog.Dialog

	confirm := func() {
		d.Hide()
		password := passwordEntry.Text

		cmd := exec.Command(sudoPath, "-S", "--", "tee", "-a", targetPath)

		stdin, err := cmd.StdinPipe()
		if err != nil {
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
				output.Refresh()
				commitButton.Enable()
			})
			return
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
				output.Refresh()
				commitButton.Enable()
			})
			return
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error: " + err.Error() + "\n"})
				output.Refresh()
				commitButton.Enable()
			})
			return
		}

		if err := cmd.Start(); err != nil {
			fyne.Do(func() {
				output.Segments = append(output.Segments, &widget.TextSegment{Text: "Error starting command: " + err.Error() + "\n"})
				output.Refresh()
				commitButton.Enable()
			})
			return
		}

		// Write sudo password then entry text to stdin.
		// sudo -S reads the password line first; tee reads the rest.
		go func() {
			defer stdin.Close()
			stdin.Write([]byte(password + "\n"))
			stdin.Write([]byte(entryText + "\n"))
		}()

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
			scanner := bufio.NewScanner(stderrPipe)
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
					output.Segments = append(output.Segments, &widget.TextSegment{Text: "\n[written to " + targetPath + "]\n"})
				}
				output.Refresh()
				scroll.ScrollToBottom()
				commitButton.Enable()
			})
		}()
	}

	cancel := func() {
		commitButton.Enable()
	}

	passwordEntry.OnSubmitted = func(_ string) {
		confirm()
	}

	d = dialog.NewCustomConfirm("sudo password required", "Commit", "Cancel",
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
	Current = LoadPrefs()
	ApplyTheme(a, Current)

	a.SetIcon(resourceIconPng)
	w := a.NewWindow("gensyn")
	w.Resize(fyne.NewSize(1920, 1080))

	// installedCache tracks which packages have been checked for installation status.
	// Key: "category/package". cacheChecked records whether we've looked it up yet;
	// installedCache holds the version string (empty = not installed).
	installedCache := map[string]string{}
	cacheChecked := map[string]bool{}

	// operationRadio, tree, and packageList are declared here so the Preferences
	// closure (built before they are assigned) can reference them via pointer.
	var operationRadio *widget.RadioGroup
	var tree *widget.Tree
	var packageList *widget.List

	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Quit", func() { a.Quit() }),
	)
	editMenu := fyne.NewMenu("Edit",
		fyne.NewMenuItem("Preferences", func() {
			themeRadio := widget.NewRadioGroup([]string{"Dark", "Light"}, nil)
			themeSelected := "Dark"
			if Current.Theme == "light" {
				themeSelected = "Light"
			}
			themeRadio.SetSelected(themeSelected)
			themeRadio.Horizontal = true

			fontSelect := widget.NewSelect([]string{"9", "10", "11", "12", "13", "14"}, nil)
			fontSelect.SetSelected(fmt.Sprintf("%.0f", Current.FontSize))

			opSelect := widget.NewSelect(operationOptions, nil)
			opSelect.SetSelected(Current.DefaultOperation)

			clearCheck := widget.NewCheck("", nil)
			clearCheck.SetChecked(Current.ClearOutput)

			form := widget.NewForm(
				widget.NewFormItem("Theme", themeRadio),
				widget.NewFormItem("Font size", fontSelect),
				widget.NewFormItem("Default operation", opSelect),
				widget.NewFormItem("Clear output on new command", clearCheck),
			)

			dialog.NewCustomConfirm("Preferences", "Save", "Cancel", form,
				func(confirmed bool) {
					if !confirmed {
						return
					}
					newPrefs := Current
					if themeRadio.Selected == "Light" {
						newPrefs.Theme = "light"
					} else {
						newPrefs.Theme = "dark"
					}
					size, _ := strconv.ParseFloat(fontSelect.Selected, 32)
					newPrefs.FontSize = float32(size)
					newPrefs.DefaultOperation = opSelect.Selected
					newPrefs.ClearOutput = clearCheck.Checked
					Current = newPrefs
					_ = SavePrefs(Current)
					ApplyTheme(a, Current)
					operationRadio.SetSelected(Current.DefaultOperation)
					tree.Refresh()
					packageList.Refresh()
				}, w).Show()
		}),
	)
	aboutMenu := fyne.NewMenu("About",
		fyne.NewMenuItem("About gensyn", func() {
			dialog.ShowInformation("About gensyn",
				"Gensyn - A Synaptic-like program for Gentoo\n\nVersion 1.0\nJarrod McCandless\n\nLicense: GPL 3\nhttps://github.com/Brainbeer/gensyn", w)
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
	packageList = widget.NewList(
		func() int { return len(packageNames) },
		func() fyne.CanvasObject {
			t := canvas.NewText("", nil)
			t.TextSize = Current.FontSize
			return t
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			t := o.(*canvas.Text)
			t.Text = packageNames[id]
			t.TextSize = Current.FontSize
			t.Refresh()
		},
	)

	selectedUID := ""

	selectInTree := func(uid string) {
		parts := strings.SplitN(uid, "/", 2)
		if len(parts) != 2 {
			return
		}
		tree.OpenBranch(parts[0])
		tree.Select(uid)
		tree.ScrollTo(uid)
	}

	// clearInstalledCache wipes the cache and refreshes the tree so installed
	// status is re-checked on the next render of each visible node.
	clearInstalledCache := func() {
		installedCache = map[string]string{}
		cacheChecked = map[string]bool{}
		tree.Refresh()
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
			// Both branches and leaves use widget.Label so we can set TextStyle.Bold.
			return widget.NewLabel("")
		},
		func(uid string, branch bool, o fyne.CanvasObject) {
			lbl := o.(*widget.Label)
			if branch {
				lbl.TextStyle = fyne.TextStyle{}
				lbl.SetText(uid)
				return
			}

			parts := strings.SplitN(uid, "/", 2)
			category, pkg := parts[0], parts[1]

			// Check cache; call GetInstalledVersion only on first encounter.
			if !cacheChecked[uid] {
				cacheChecked[uid] = true
				installedCache[uid] = portage.GetInstalledVersion(category, pkg)
			}

			if installedCache[uid] != "" {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText("✓ " + pkg)
			} else {
				lbl.TextStyle = fyne.TextStyle{}
				lbl.SetText(pkg)
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

	operationRadio = widget.NewRadioGroup(operationOptions, nil)
	operationRadio.SetSelected(Current.DefaultOperation)
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
		// Sync does not change installed packages so no cache clear needed.
		promptAndRun([]string{"emerge", "--sync"}, true, output, outputScroll, syncButton, installButton, w, nil)
	}

	installButton.OnTapped = func() {
		op := operationRadio.Selected

		switch op {

		case "-uvNDU (Update)":
			promptAndRun([]string{"emerge", "-uvNDU", "world"}, true, output, outputScroll, syncButton, installButton, w, clearInstalledCache)

		case "--depclean":
			promptAndRun([]string{"emerge", "--depclean"}, true, output, outputScroll, syncButton, installButton, w, clearInstalledCache)

		case "-C (Uninstall)":
			if selectedUID == "" {
				dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
				return
			}
			promptAndRun([]string{"emerge", "-C", selectedUID}, true, output, outputScroll, syncButton, installButton, w, clearInstalledCache)

		case "-p (pretend)":
			if selectedUID == "" {
				dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
				return
			}
			promptAndRun([]string{"emerge", "-p", selectedUID}, false, output, outputScroll, syncButton, installButton, w, nil)

		case "-f (fetch)":
			if selectedUID == "" {
				dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
				return
			}
			promptAndRun([]string{"emerge", "-f", selectedUID}, true, output, outputScroll, syncButton, installButton, w, nil)

		case "Custom":
			if selectedUID == "" {
				dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
				return
			}
			raw := strings.TrimSpace(customEntry.Text)
			if raw == "" {
				dialog.ShowInformation("No flags", "Please enter custom flags or select a different mode.", w)
				return
			}
			args := append([]string{"emerge"}, strings.Fields(raw)...)
			args = append(args, selectedUID)
			promptAndRun(args, true, output, outputScroll, syncButton, installButton, w, nil)

		default: // "No Flag"
			if selectedUID == "" {
				dialog.ShowInformation("No package selected", "Please select a package from the tree first.", w)
				return
			}
			promptAndRun([]string{"emerge", selectedUID}, true, output, outputScroll, syncButton, installButton, w, clearInstalledCache)
		}
	}

	// Bottom toolbar: portage config writer
	portageRadio := widget.NewRadioGroup([]string{"package.mask", "package.accept_keywords", "package.use"}, nil)
	portageRadio.SetSelected("package.use")
	portageRadio.Horizontal = true

	portageEntry := widget.NewEntry()
	portageEntry.SetPlaceHolder("category/package [flags]")

	commitButton := widget.NewButton("Commit", nil)
	commitButton.OnTapped = func() {
		dir := portageRadio.Selected
		if dir == "" {
			dialog.ShowInformation("No target selected", "Please select package.mask, package.accept_keywords, or package.use.", w)
			return
		}
		commitToPortage(portageEntry.Text, dir, output, outputScroll, commitButton, w)
	}

	commitButtonContainer := container.New(
		layout.NewGridWrapLayout(fyne.NewSize(80, 30)),
		commitButton,
	)

	bottomToolbar := container.NewBorder(nil, nil, portageRadio, commitButtonContainer, portageEntry)

	centerSection := container.NewVSplit(packageList, descriptionScroll)
	centerSection.SetOffset(0.6)

	lowerSection := container.NewHSplit(centerSection, outputScroll)
	lowerSection.SetOffset(0.24)

	rightSection := container.NewBorder(toolbar, nil, nil, nil, lowerSection)

	main := container.NewHSplit(tree, rightSection)
	main.SetOffset(0.12)

	root := container.NewBorder(nil, bottomToolbar, nil, nil, main)

	w.SetContent(root)
	w.ShowAndRun()
}
