// portage/portage.go

package portage

import (
	"os"

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
