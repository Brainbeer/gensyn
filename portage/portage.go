// portage/portage.go

package portage

import (
	"os"
	"strings"

	"github.com/Brainbeer/gensyn/models"
)

// GetCategories returns a list of Portage categories

func GetCategories() ([]models.Category, error) {
	entries, err := os.ReadDir("/var/db/repos/gentoo/")
	if err != nil {
		return nil, err
	}

	var categories []models.Category
	for _, entry := range entries {
		if entry.IsDir() {
			categories = append(categories, models.Category{Name: entry.Name()})
		}
	}

	return categories, nil
}

// GetPackages returns a list of packages for a given category

func GetPackages(category string) ([]models.Package, error) {
	entries, err := os.ReadDir("/var/db/repos/gentoo/" + category)
	if err != nil {
		return nil, err
	}

	var packages []models.Package
	for _, entry := range entries {
		if entry.IsDir() {
			packages = append(packages, models.Package{
				Name:     entry.Name(),
				Category: category,
			})
		}
	}

	return packages, nil
}

// GetPackageFiles returns the files inside a package directory

func GetPackageFiles(category, pkg string) ([]string, error) {
	entries, err := os.ReadDir("/var/db/repos/gentoo/" + category + "/" + pkg)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		files = append(files, entry.Name())
	}

	return files, nil
}

// GetInstalledVersion checks /var/db/pkg/<category>/ for an installed version of the package
// Returns the version string (e.g. "brasero-3.12.3") or "" if not installed

func GetInstalledVersion(category, pkg string) string {
	entries, err := os.ReadDir("/var/db/pkg/" + category)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		name := entry.Name()
		if len(name) > len(pkg) && name[:len(pkg)] == pkg && name[len(pkg)] == '-' {
			return name
		}
	}

	return ""
}

// GetDescription parses the DESCRIPTION= field from the first .ebuild in the package directory.
// Returns an empty string if not found or on any error.

func GetDescription(category, pkg string) string {
	dir := "/var/db/repos/gentoo/" + category + "/" + pkg
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var ebuildPath string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".ebuild") {
			ebuildPath = dir + "/" + entry.Name()
			break
		}
	}
	if ebuildPath == "" {
		return ""
	}

	data, err := os.ReadFile(ebuildPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "DESCRIPTION=") {
			val := strings.TrimPrefix(line, "DESCRIPTION=")
			val = strings.Trim(val, `"'`)
			return val
		}
	}

	return ""
}
