VERSION  ?= $(shell git describe --tags --always)
BUILD_TIME = $(shell date -u '+%Y-%m-%d %H:%M UTC')
LDFLAGS  = -w -s -X main.version=$(VERSION) -X 'main.buildTime=$(BUILD_TIME)'

# --- macOS -----------------------------------------------------------

.PHONY: mac mac-cli mac-app mac-install mac-bundle mac-clean

mac: mac-cli mac-app mac-install mac-bundle  ## Build and install everything on macOS

mac-cli:  ## Build the CLI
	go build -ldflags "$(LDFLAGS)" -o build/places ./cmd/places

mac-app:  ## Build the desktop app (requires CGO)
	CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
	go build -tags production -ldflags "$(LDFLAGS)" -o build/places-app ./cmd/places-app

mac-install: mac-cli mac-app  ## Install binaries to ~/.local/bin
	@mkdir -p ~/.local/bin
	cp build/places ~/.local/bin/places
	cp build/places-app ~/.local/bin/places-app

mac-bundle: mac-app  ## Create/update Places.app bundle in /Applications
	@mkdir -p /Applications/Places.app/Contents/MacOS
	@mkdir -p /Applications/Places.app/Contents/Resources
	cp build/places-app /Applications/Places.app/Contents/MacOS/places-app
	@echo '<?xml version="1.0" encoding="UTF-8"?>' > /Applications/Places.app/Contents/Info.plist
	@echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> /Applications/Places.app/Contents/Info.plist
	@echo '<plist version="1.0"><dict>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleExecutable</key><string>places-app</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleIdentifier</key><string>com.mavwarf.places</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleName</key><string>Places</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleDisplayName</key><string>Places</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleVersion</key><string>$(VERSION)</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleShortVersionString</key><string>$(VERSION)</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundlePackageType</key><string>APPL</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>CFBundleIconFile</key><string>AppIcon</string>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>NSHighResolutionCapable</key><true/>' >> /Applications/Places.app/Contents/Info.plist
	@echo '<key>NSSupportsAutomaticGraphicsSwitching</key><true/>' >> /Applications/Places.app/Contents/Info.plist
	@echo '</dict></plist>' >> /Applications/Places.app/Contents/Info.plist
	@if [ ! -f /Applications/Places.app/Contents/Resources/AppIcon.icns ]; then \
		iconset=$$(mktemp -d)/AppIcon.iconset && mkdir -p "$$iconset" && \
		for size in 16 32 128 256 512; do \
			sips -z $$size $$size cmd/places-app/appicon.png --out "$$iconset/icon_$${size}x$${size}.png" >/dev/null 2>&1; \
			double=$$((size * 2)); \
			if [ $$double -le 1024 ]; then \
				sips -z $$double $$double cmd/places-app/appicon.png --out "$$iconset/icon_$${size}x$${size}@2x.png" >/dev/null 2>&1; \
			fi; \
		done && \
		iconutil -c icns "$$iconset" -o /Applications/Places.app/Contents/Resources/AppIcon.icns && \
		rm -rf "$$(dirname $$iconset)"; \
	fi
	@echo "Places.app updated — launch from Spotlight"

mac-clean:  ## Remove build artifacts
	rm -rf build/

# --- Windows ----------------------------------------------------------

.PHONY: windows windows-cli windows-app

windows: windows-cli windows-app  ## Build everything on Windows

windows-cli:  ## Build the CLI
	go build -ldflags "$(LDFLAGS)" -o build/places.exe ./cmd/places

windows-app:  ## Build the desktop app
	go build -tags production -ldflags "$(LDFLAGS) -H windowsgui" -o build/places-app.exe ./cmd/places-app

# --- Common -----------------------------------------------------------

.PHONY: clean

clean:  ## Remove all build artifacts
	rm -rf build/
