# 5-Minute Quick Start Guide

Get up and running with Zagalin plugin development in 5 minutes.

##  Prerequisites Check (30 seconds)

```bash
# Required versions
node --version    # Should be >= 22
go version        # Should be >= 1.21
docker --version  # Any recent version
git --version     # Any recent version
```

**Missing something?**
- Node: https://nodejs.org/ (use LTS version)
- Go: https://go.dev/dl/
- Docker: https://docs.docker.com/get-docker/

##  Initial Setup (2 minutes)

```bash
# 1. Navigate to project
cd /home/j/repos/jorgesnotebook/jorgeancal-zagalin-app

# 2. Install dependencies
npm ci

# 3. Verify installation
npm run typecheck
npm run lint
```

**If errors occur**: Check Node version is 22+

##  Start Development (1 minute)

Open **3 terminals**:

**Terminal 1 - Frontend Watch:**
```bash
npm run dev
```
 You should see: "webpack compiled successfully"

**Terminal 2 - Grafana Server:**
```bash
npm run server
```
 Wait for: "HTTP Server Listen" in logs

**Terminal 3 - Tests:**
```bash
npm run test
```
 Tests should pass

##  Access the Plugin (30 seconds)

1. Open browser: http://localhost:3000
2. Login: `admin` / `admin`
3. Navigate to: **Administration → Plugins → Zagalin**
4. Click **"Enable"** if not already enabled

##  Make Your First Change (1 minute)

**Edit a file:**
```bash
# Open src/pages/ChatPage.tsx
# Find line with "Welcome" text
# Change it to "Hello World!"
# Save the file
```

**See the change:**
1. Refresh browser (http://localhost:3000/a/jorgeancal-zagalin-app)
2. You should see "Hello World!"

**Congratulations! ** Your development environment is working!

##  Next Steps

Now that you're set up, explore based on your role:

### Frontend Developer
1. Read: `.claude/rules/00-getting-started/frontend-tour.md`
2. Try: Add a new page (see `.claude/rules/00-getting-started/common-tasks.md`)
3. Learn: React patterns in `src/components/`

### Backend Developer
1. Read: `.claude/rules/00-getting-started/backend-tour.md`
2. Try: Add a new endpoint (see `.claude/rules/00-getting-started/common-tasks.md`)
3. Learn: Go patterns in `pkg/plugin/`

### LLM/AI Developer
1. Read: `.claude/rules/03-integrations/llm-official.md`
2. Try: Add a new chat tool/function
3. Learn: `pkg/plugin/assistant.go` and `src/services/assistantService.ts`

### DevOps/SRE
1. Read: `.claude/rules/04-quality/ci-cd.md`
2. Try: Run full CI locally: `./ci-local.sh`
3. Learn: `.github/workflows/`

##  Quick Troubleshooting

### Port 3000 already in use
```bash
# Find process using port 3000
lsof -ti:3000
# Kill it
kill -9 $(lsof -ti:3000)
# Or use different port
GRAFANA_URL=http://localhost:3001 npm run server
```

### Plugin not loading in Grafana
```bash
# Rebuild and restart
npm run build
docker restart jorgeancal-zagalin-app
```

### Tests failing
```bash
# Clean install
rm -rf node_modules package-lock.json
npm ci
npm run test:ci
```

### Backend not starting
```bash
# Rebuild backend
mage -v buildAll
# Check binary exists
ls -la dist/gpx_*
# Check logs
docker logs jorgeancal-zagalin-app
```

##  Full Documentation

- **Architecture Overview**: `.claude/rules/00-getting-started/architecture-tour.md`
- **Common Tasks**: `.claude/rules/00-getting-started/common-tasks.md`
- **Decision Trees**: `.claude/DECISION_TREES.md`
- **Complete Index**: `.claude/DOCUMENTATION_INDEX.md`

##  Learning Resources

**Today (Day 1)**:
-  Complete this quick start
-  Make a small change
-  Read architecture tour

**This Week**:
- Read KISS principles: `.claude/rules/02-development/clean-code.md`
- Read your role-specific tour
- Build a simple feature

**This Month**:
- Deep dive into relevant standards
- Contribute to a real feature
- Review testing guide

##  Pro Tips

**Speed up development:**
```bash
# Use aliases
alias dev="npm run dev"
alias test="npm run test"
alias build="npm run build && mage -v buildAll"
alias serve="npm run server"
```

**Pre-commit checks:**
```bash
# Before committing
npm run typecheck && npm run lint && npm run test:ci
```

**Full validation:**
```bash
# Run complete CI pipeline
./ci-local.sh
```

##  Daily Development Checklist

Before starting work:
- [ ] Pull latest: `git pull`
- [ ] Install deps: `npm ci` (if package.json changed)
- [ ] Start dev servers (3 terminals)
- [ ] Check tests pass

Before committing:
- [ ] Code follows KISS principles
- [ ] Tests written and passing
- [ ] No console.log statements
- [ ] Run: `npm run typecheck && npm run lint`

Before pushing:
- [ ] Run: `./ci-local.sh`
- [ ] All tests pass
- [ ] Commit message follows convention

---

**Time to complete**: 5 minutes
**Last updated**: 2026-01-10
**Questions?** Check `.claude/DECISION_TREES.md` or ask your team!
