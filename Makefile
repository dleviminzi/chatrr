OSNAME=$(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(shell uname -m)
VSSOS=$(OSNAME)
VSS_VERSION=v0.1.1-alpha.19
DBNAME?=db.sqlite

ifeq ($(OSNAME),darwin)
	VSSOS=macos
	ifeq ($(ARCH),arm64)
		ARCH=aarch64
		CGO_LDFLAGS=-L$(HOMEBREW_PREFIX)/opt/libomp/lib -L./extensions -Wl,-undefined,dynamic_lookup -lomp
	else
		CGO_LDFLAGS=-L./extensions -Wl,-undefined,dynamic_lookup -lomp
	endif
else ifeq ($(OSNAME),linux)
	CGO_LDFLAGS=-L./extensions -Wl,-undefined,dynamic_lookup -lstdc++
endif

build: extensions
	@echo "Building for $(OSNAME) $(ARCH) with CGO_LDFLAGS=$(CGO_LDFLAGS)"
	@CGO_LDFLAGS="$(CGO_LDFLAGS)" go build -o bin/chatrr ./cmd/main.go

clean:
	@echo "Cleaning"
	@rm -rf b

extensions:
	@echo "Downloading sqlite-vss $(VSS_VERSION)"
	@mkdir -p extensions
	@curl -sL "https://github.com/asg017/sqlite-vss/releases/download/$(VSS_VERSION)/sqlite-vss-$(VSS_VERSION)-static-$(VSSOS)-$(ARCH).tar.gz" | tar zx -C extensions
