// SPDX-License-Identifier: GPL-3.0-or-later

package repo

import (
	"path/filepath"

	"github.com/samlzz/situ/internal/situfile"
)

// SituFileName is the name of the manifest of a module.
const SituFileName = "Situfile"

// Rel returns the repository-relative directory of id.
func Rel(id situfile.ModuleID) string {
	if id.IsProfile() {
		return situfile.HostsDir + "/" + id.Component
	}
	return situfile.ComponentsDir + "/" + id.String()
}

// SituFile returns the repository-relative path of the Situfile of id.
func SituFile(id situfile.ModuleID) string { return Rel(id) + "/" + SituFileName }

// ModuleDir returns the directory of id on disk.
func (r *Repo) ModuleDir(id situfile.ModuleID) string {
	return filepath.Join(r.Dir, filepath.FromSlash(Rel(id)))
}
