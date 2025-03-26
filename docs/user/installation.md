# MegaTool Installation Guide

This guide provides detailed instructions for installing MegaTool on various platforms.

## Prerequisites

- **Go**: Version 1.24 or later
- **Mise**: For managing the Go toolchain (recommended)
- **Just**: For running tasks (recommended)

## Installation Methods

### Method 1: Using GitHub Releases (Recommended)

The easiest way to install MegaTool is by downloading a pre-built binary from the GitHub releases page:

1. Visit the [GitHub Releases page](https://github.com/yourusername/megatool/releases)
2. Download the appropriate archive for your platform:
   - `megatool-v{VERSION}-linux-amd64.tar.gz` - For Linux (x86_64)
   - `megatool-v{VERSION}-linux-arm64.tar.gz` - For Linux (ARM64)
   - `megatool-v{VERSION}-darwin-arm64.tar.gz` - For macOS (Apple Silicon)
   - `megatool-v{VERSION}-windows-amd64.zip` - For Windows (x86_64)
3. Extract the archive to a directory in your PATH:

```bash
# Linux/macOS (example for v1.0.0 on Linux amd64)
tar -xzf megatool-v1.0.0-linux-amd64.tar.gz -C /usr/local/bin

# Windows
# Extract the ZIP file and add the directory to your PATH
```

Each release includes SHA256 checksums for verifying the integrity of the downloaded files.

### Method 2: Using Go Install

The simplest way to install MegaTool is using Go's install command:

```bash
go install github.com/yourusername/megatool@latest
```

This will download, compile, and install the latest version of MegaTool to your `$GOPATH/bin` directory. Ensure this directory is in your system PATH.

### Method 3: Building from Source

#### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/megatool.git
cd megatool
```

#### 2. Install Prerequisites (if not already installed)

##### Installing Mise

[Mise](https://github.com/jdx/mise) is a tool for managing development tool versions:

```bash
# macOS (using Homebrew)
brew install mise

# Linux/WSL
curl https://mise.run | sh
```

##### Installing Just

[Just](https://github.com/casey/just) is a command runner that simplifies common tasks:

```bash
# macOS (using Homebrew)
brew install just

# Linux/WSL
cargo install just
```

#### 3. Build and Install

Using Just (recommended):

```bash
# Build all binaries
just build

# Install the binaries
just install
```

Manual build (alternative):

```bash
# Build the main binary
go build -o bin/megatool ./cmd/megatool

# Build the server binaries
go build -o bin/megatool-calculator ./cmd/megatool-calculator
go build -o bin/megatool-github ./cmd/megatool-github
go build -o bin/megatool-package-version ./cmd/megatool-package-version

# Install the binaries
cp bin/megatool* $GOPATH/bin/
```

## Verifying Installation

To verify that MegaTool has been installed correctly:

```bash
megatool --version
```

This should display the current version of MegaTool.

## Troubleshooting

### Common Issues

#### "Command not found" Error

If you see a "command not found" error when trying to run MegaTool, ensure that:

1. The installation completed successfully
2. The installation directory is in your PATH

For Go installations, check that `$GOPATH/bin` is in your PATH:

```bash
echo $PATH
```

Add it to your PATH if needed:

```bash
# For bash/zsh
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc  # or ~/.zshrc
source ~/.bashrc  # or ~/.zshrc
```

#### Permission Issues

If you encounter permission issues during installation:

```bash
# For Go install
go install -v github.com/yourusername/megatool@latest

# For manual installation
sudo cp bin/megatool* /usr/local/bin/
```

## Next Steps

After installation, you may want to:

1. [Configure the GitHub server](github.md) if you plan to use GitHub functionality
2. Check out the [general usage guide](usage.md) for more information on using MegaTool
