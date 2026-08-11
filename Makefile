# JABARI Makefile

BINARY ?= jabari
ALIAS  ?= androidsec
PREFIX ?= /usr/local
DESTDIR ?=

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
USER    ?= $(shell id -u -n)

LDFLAGS := -s -w \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Version=$(VERSION) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Commit=$(COMMIT) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.Date=$(DATE) \
	-X github.com/anomalyco/qyvora-jabari/internal/version.BuildUser=$(USER)

# --- install layout ------------------------------------------------------
# System-wide install (default PREFIX=/usr/local, typically needs root):
#   /usr/local/bin/jabari            command
#   /usr/local/bin/androidsec        alias symlink
#   /usr/local/share/applications/   desktop entry (searchable in the app menu)
#   /usr/local/share/icons/hicolor/512x512/apps/jabari.png
#   /usr/local/share/pixmaps/jabari.png
# User install (make install-user) mirrors the same layout under ~/.local.

ICON    := assets/jabari.png
DESKTOP := assets/jabari.desktop

BINDIR    := $(DESTDIR)$(PREFIX)/bin
ICONDIR   := $(DESTDIR)$(PREFIX)/share/icons/hicolor/512x512/apps
PIXMAPDIR := $(DESTDIR)$(PREFIX)/share/pixmaps
APPDIR    := $(DESTDIR)$(PREFIX)/share/applications

USERBIN    := $(HOME)/.local/bin
USERICON   := $(HOME)/.local/share/icons/hicolor/512x512/apps
USERPIXMAP := $(HOME)/.local/share/pixmaps
USERAPP    := $(HOME)/.local/share/applications

.PHONY: all build test vet fmt check install install-user uninstall uninstall-user clean

all: build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/jabari
	ln -sf $(BINARY) bin/$(ALIAS)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal pkg

check: fmt vet test

install: build
	install -d $(BINDIR)
	install -m 0755 bin/$(BINARY) $(BINDIR)/$(BINARY)
	ln -sf $(BINARY) $(BINDIR)/$(ALIAS)
	$(MAKE) install-data

install-data:
	install -d $(ICONDIR) $(PIXMAPDIR) $(APPDIR)
	install -m 0644 $(ICON) $(ICONDIR)/jabari.png
	install -m 0644 $(ICON) $(PIXMAPDIR)/jabari.png
	sed -e 's|@PREFIX@|$(PREFIX)|g' $(DESKTOP) > $(APPDIR)/jabari.desktop
	chmod 0644 $(APPDIR)/jabari.desktop
	update-desktop-database $(APPDIR) 2>/dev/null || true
	gtk-update-icon-cache -f $(DESTDIR)$(PREFIX)/share/icons/hicolor 2>/dev/null || true
	@echo "jabari installed to $(BINDIR) with icon and desktop entry."

install-user: build
	install -d $(USERBIN)
	install -m 0755 bin/$(BINARY) $(USERBIN)/$(BINARY)
	ln -sf $(BINARY) $(USERBIN)/$(ALIAS)
	install -d $(USERICON) $(USERPIXMAP) $(USERAPP)
	install -m 0644 $(ICON) $(USERICON)/jabari.png
	install -m 0644 $(ICON) $(USERPIXMAP)/jabari.png
	sed -e 's|@PREFIX@|$(HOME)/.local|g' $(DESKTOP) > $(USERAPP)/jabari.desktop
	chmod 0644 $(USERAPP)/jabari.desktop
	update-desktop-database $(USERAPP) 2>/dev/null || true
	gtk-update-icon-cache -f $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@echo "jabari installed to $(USERBIN) with icon and desktop entry."
	@echo "Add $$HOME/.local/bin to your PATH if it is not already there."

uninstall:
	rm -f $(BINDIR)/$(BINARY) $(BINDIR)/$(ALIAS)
	rm -f $(ICONDIR)/jabari.png $(PIXMAPDIR)/jabari.png $(APPDIR)/jabari.desktop
	update-desktop-database $(APPDIR) 2>/dev/null || true
	gtk-update-icon-cache -f $(DESTDIR)$(PREFIX)/share/icons/hicolor 2>/dev/null || true

uninstall-user:
	rm -f $(USERBIN)/$(BINARY) $(USERBIN)/$(ALIAS)
	rm -f $(USERICON)/jabari.png $(USERPIXMAP)/jabari.png $(USERAPP)/jabari.desktop
	update-desktop-database $(USERAPP) 2>/dev/null || true
	gtk-update-icon-cache -f $(HOME)/.local/share/icons/hicolor 2>/dev/null || true

clean:
	rm -rf bin
