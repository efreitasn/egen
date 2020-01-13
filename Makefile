all: build install

# build: VERSION=$(shell git describe --abbrev=0 --tags)
build: VERSION="TODO"
build: NAME="ecms"
build: export GOOS?=$(shell go env GOOS)
build: export GOARCH?=$(shell go env GOARCH)
build:
	@echo "Building ecms@${VERSION} for ${GOOS}/${GOARCH}"
	@go build -ldflags="-X github.com/efreitasn/ecms/cmd/ecms/internal/cmds.version=${VERSION}" \
		-o ${NAME} github.com/efreitasn/ecms/cmd/ecms
	@echo "Build completed"

install:
	@sudo cp ecms /usr/local/bin
	@sudo cp completion.sh /usr/share/bash-completion/completions/ecms
	@if [ -f ~/.zshrc ] && ! grep -q "# ecms" ~/.zshrc; then\
  	echo "\n# ecms\nautoload bashcompinit\nbashcompinit\nsource /usr/share/bash-completion/completions/ecms" >> ~/.zshrc;\
	fi
	@echo "Installation is complete"