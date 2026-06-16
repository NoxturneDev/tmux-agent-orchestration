#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0;37m' # No Color

echo -e "${YELLOW}Step 1: Building Svelte 5 frontend...${NC}"
cd web/frontend
bun run build
cd ../..

echo -e "\n${YELLOW}Step 2: Compiling Go binary with embedded assets...${NC}"
go build -o tmux-ai-orchestrator .

# Guard check for TMUX
if [ -z "$TMUX" ]; then
    echo -e "\n${RED}Error: This application must be run inside an active tmux session.${NC}"
    echo -e "Please start or attach to a tmux session, then run: ${YELLOW}./run_dashboard.sh${NC}"
    exit 1
fi

echo -e "\n${GREEN}Step 3: Starting Tmux AI Orchestrator Web Server on http://localhost:8069${NC}"
exec ./tmux-ai-orchestrator web --port=8069
