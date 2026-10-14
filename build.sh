#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building RFPIO-Go...${NC}"

# Build the core library (just verify it compiles)
echo -e "${YELLOW}  Checking core library...${NC}"
go build ./internal/core/...
echo -e "${GREEN}  Core library: OK${NC}"

# Build the CLI tool
echo -e "${YELLOW}  Building rfpio CLI...${NC}"
go build -o bin/rfpio ./cmd/rfpio/
echo -e "${GREEN}  rfpio CLI: OK${NC}"

# Build the MCP server
echo -e "${YELLOW}  Building rfpio-mcp server...${NC}"
go build -o bin/rfpio-mcp ./cmd/rfpio-mcp/
echo -e "${GREEN}  rfpio-mcp: OK${NC}"

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo "  CLI:      bin/rfpio"
echo "  MCP:      bin/rfpio-mcp"
echo ""
echo "Run 'go test ./...' to run tests."
