#!/bin/bash

# HyprKnot Installation Script
# Lightweight HTTP API wrapper for KnotDNS

set -e

BINARY_NAME="hyprknot"
VERSION="1.0.0"
CONFIG_DIR="/etc/hyprknot"
SYSTEMD_DIR="/etc/systemd/system"
BINARY_DIR="/usr/local/bin"
LOG_DIR="/var/log/hyprknot"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check if KnotDNS is installed
check_knot() {
    if ! command -v knotc &> /dev/null; then
        print_warning "knotc command not found. Please install KnotDNS first."
        print_status "On Ubuntu/Debian: apt install knot"
        print_status "On CentOS/RHEL: yum install knot"
        exit 1
    fi

    print_success "KnotDNS found: $(knotc --version | head -n1)"
}

# Check if binary exists
check_binary() {
    if [[ ! -f "build/$BINARY_NAME" ]]; then
        print_error "Binary not found. Please run 'make build' first."
        exit 1
    fi
}

# Create user and directories
setup_user() {
    print_status "Setting up user and directories..."

    # Create hyprknot user if it doesn't exist
    if ! id "hyprknot" &>/dev/null; then
        useradd --system --shell /bin/false --no-create-home hyprknot
        print_success "Created hyprknot user"
    fi

    # Add hyprknot user to knot group for socket access
    if getent group knot >/dev/null 2>&1; then
        usermod -a -G knot hyprknot
        print_success "Added hyprknot user to knot group"
    else
        print_warning "knot group not found. Ensure KnotDNS is properly installed."
    fi

    # Create directories
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"

    # Set ownership
    chown hyprknot:hyprknot "$LOG_DIR"
    chown hyprknot:hyprknot "$CONFIG_DIR"

    print_success "Directories created"
}

# Install binary
install_binary() {
    print_status "Installing binary..."

    cp "build/$BINARY_NAME" "$BINARY_DIR/"
    chmod +x "$BINARY_DIR/$BINARY_NAME"

    print_success "Binary installed to $BINARY_DIR/$BINARY_NAME"
}

# Install configuration
install_config() {
    print_status "Installing configuration..."

    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        print_warning "Configuration file already exists, backing up..."
        cp "$CONFIG_DIR/config.yaml" "$CONFIG_DIR/config.yaml.backup.$(date +%Y%m%d-%H%M%S)"
    fi

    cp config.yaml "$CONFIG_DIR/"
    chown hyprknot:hyprknot "$CONFIG_DIR/config.yaml"
    chmod 640 "$CONFIG_DIR/config.yaml"

    print_success "Configuration installed to $CONFIG_DIR/config.yaml"
}

# Install systemd service
install_service() {
    print_status "Installing systemd service..."

    cp hyprknot.service "$SYSTEMD_DIR/"
    systemctl daemon-reload

    print_success "Systemd service installed"
}

# Main installation
main() {
    echo "🚀 HyprKnot Installation Script"
    echo "================================"
    echo

    check_root
    check_knot
    check_binary

    setup_user
    install_binary
    install_config
    install_service

    echo
    print_success "Installation completed successfully!"
    echo
    print_status "Next steps:"
    echo "  1. Edit the configuration: nano $CONFIG_DIR/config.yaml"
    echo "  2. Add your API keys and configure allowed zones"
    echo "  3. Start the service: systemctl enable --now hyprknot"
    echo "  4. Check status: systemctl status hyprknot"
    echo "  5. View logs: journalctl -u hyprknot -f"
    echo
    print_status "API will be available at: http://localhost:8080"
    print_status "Health check: curl http://localhost:8080/health"
    print_status "API docs: curl http://localhost:8080/api/v1/docs"
    echo
}

# Run installation
main "$@"
