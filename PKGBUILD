pkgname=snobs
pkgver=1.2e45266
pkgrel=1
pkgdesc="add snobs to your pull-requests"
url="https://github.com/reconquest/snobs"
arch=('i686' 'x86_64')
license=('GPL')
makedepends=('go')

source=("git://github.com/reconquest/snobs.git" "snobs.service")
md5sums=('SKIP' 'SKIP')
backup=("etc/snobs/snobs.conf")

pkgver() {
    cd "${pkgname}"
    echo $(git rev-list --count master).$(git rev-parse --short master)
}

build() {
    cd "$srcdir/$pkgname"

    rm -rf "$srcdir/.go/src"

    mkdir -p "$srcdir/.go/src"

    export GOPATH=$srcdir/.go

    mv "$srcdir/$pkgname" "$srcdir/.go/src/"

    cd "$srcdir/.go/src/snobs/"
    ln -sf "$srcdir/.go/src/snobs/" "$srcdir/$pkgname"

    go get
}

package() {
    mkdir -p "$pkgdir/etc/snobs/"
    mkdir -p "$pkgdir/usr/bin"
    mkdir -p "$pkgdir/etc/systemd/system"

    chmod 0600 "$pkgdir/etc/snobs/"

    cp "$srcdir/.go/src/$pkgname/snobs.conf" "$pkgdir/etc/snobs/"
    cp "$srcdir/.go/bin/$pkgname" "$pkgdir/usr/bin"
    cp "$srcdir/snobs.service" "$pkgdir/etc/systemd/system/"
}
