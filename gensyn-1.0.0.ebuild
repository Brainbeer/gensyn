# Copyright 2026 Gentoo Authors
# Distributed under the terms of the GNU General Public License v3

EAPI=8

inherit go-module

DESCRIPTION="A Synaptic-like graphical package manager for Gentoo"
HOMEPAGE="https://github.com/Brainbeer/gensyn"
SRC_URI="https://github.com/Brainbeer/gensyn/archive/refs/tags/v${PV}.tar.gz -> ${P}.tar.gz"

LICENSE="GPL-3"
SLOT="0"
KEYWORDS="~amd64"

RDEPEND="
	app-admin/sudo
	x11-libs/libX11
	x11-libs/libXcursor
	x11-libs/libXrandr
	x11-libs/libXinerama
	media-libs/mesa
"

BDEPEND=">=dev-lang/go-1.21"

src_compile() {
	GOFLAGS="-mod=vendor" ego build -o gensyn .
}

src_install() {
	dobin gensyn
	dodoc README.md CHANGELOG.md CONTRIBUTING.md
}
