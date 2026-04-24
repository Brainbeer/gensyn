// models/models.go

package models

// Category represents a Portage Category (app-admin, x11-libs, etc)

type Category struct {
	Name string
}

// Package represents a Portage package

type Package struct {
	Name      string
	Category  string
	Version   string
	Installed bool
}
