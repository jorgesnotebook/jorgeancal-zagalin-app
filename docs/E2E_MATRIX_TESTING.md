# E2E Matrix Testing Guide

This guide explains how to run E2E tests locally against multiple Grafana versions, similar to the CI pipeline.

## Overview

The plugin supports testing across multiple Grafana versions:

- **10.0.0** - Grafana 10.0 (initial 10.x release)
- **10.4.0** - Grafana 10.4 (minimum supported version for this plugin)
- **11.0.0** - Grafana 11.0 (major version upgrade)
- **11.1.0** - Grafana 11.1 (minor updates)
- **12.0.0** - Grafana 12.0 (latest stable, default)

## Prerequisites

1. **Docker** - Running and accessible
2. **Node.js** - Version 22 or higher
3. **Built plugin** - Both frontend and backend compiled
4. **Dependencies installed** - `npm ci` completed

## Quick Start

The easiest way to run matrix tests is using npm scripts:

```bash
# Test all versions (10.0.0, 10.4.0, 11.0.0, 11.1.0, 12.0.0)
npm run e2e:matrix

# Quick test (minimum and latest versions only: 10.4.0, 12.0.0)
npm run e2e:matrix:quick
```

### Advanced Usage

For more control, use the script directly:

```bash
# Run tests against all versions
./scripts/e2e-matrix-local.sh

# Test specific versions only
./scripts/e2e-matrix-local.sh 11.0.0 12.0.0

# Stop on first failure
STOP_ON_FAILURE=true npm run e2e:matrix

# Keep containers running after tests (for debugging)
KEEP_CONTAINERS=true ./scripts/e2e-matrix-local.sh 12.0.0

# Use open-source Grafana instead of Enterprise
GRAFANA_IMAGE=grafana npm run e2e:matrix
```

### What the Script Does

1. **For each Grafana version:**
   - Stops any running Grafana containers
   - Starts Grafana with the specific version
   - Waits for Grafana to be ready (health check)
   - Runs E2E tests (`npm run e2e`)
   - Saves results and logs
   - Stops the container (unless `KEEP_CONTAINERS=true`)

2. **Generates results:**
   - Test output logs: `e2e-matrix-results/test_output_{version}_{timestamp}.log`
   - Grafana logs: `e2e-matrix-results/grafana_{version}_{timestamp}.log` (on failure)
   - Playwright reports: `e2e-matrix-results/playwright-report_{version}_{timestamp}/`
   - Summary: `e2e-matrix-results/results_{timestamp}.txt`

### Script Options

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `STOP_ON_FAILURE` | `false` | Stop testing on first failure |
| `KEEP_CONTAINERS` | `false` | Keep Grafana running after tests |
| `GRAFANA_IMAGE` | `grafana-enterprise` | Grafana image type (`grafana` or `grafana-enterprise`) |

### Example: Testing Specific Versions

```bash
# Test only the minimum and latest versions
./scripts/e2e-matrix-local.sh 10.4.0 12.0.0

# Test with OSS Grafana
GRAFANA_IMAGE=grafana ./scripts/e2e-matrix-local.sh 11.0.0 12.0.0

# Test one version and keep it running for debugging
KEEP_CONTAINERS=true ./scripts/e2e-matrix-local.sh 12.0.0
```

## Manual Testing (Step-by-Step)

If you prefer manual control, follow these steps for each version:

### 1. Build the Plugin

```bash
# Build frontend
npm run build

# Build backend
mage -v buildAll
```

### 2. Test Against Specific Version

```bash
# Stop any running containers
docker compose down

# Start Grafana with specific version
GRAFANA_VERSION=10.4.0 npm run server

# Wait for Grafana to be ready (check health)
curl http://localhost:3000/api/health

# Run E2E tests
npm run e2e

# View results
npx playwright show-report

# Stop Grafana
docker compose down
```

### 3. Repeat for Each Version

```bash
# Grafana 10.0.0
docker compose down
GRAFANA_VERSION=10.0.0 npm run server
# Wait, test, stop...

# Grafana 10.4.0
docker compose down
GRAFANA_VERSION=10.4.0 npm run server
# Wait, test, stop...

# Grafana 11.0.0
docker compose down
GRAFANA_VERSION=11.0.0 npm run server
# Wait, test, stop...

# Grafana 11.1.0
docker compose down
GRAFANA_VERSION=11.1.0 npm run server
# Wait, test, stop...

# Grafana 12.0.0
docker compose down
GRAFANA_VERSION=12.0.0 npm run server
# Wait, test, stop...
```

## Understanding Test Results

### Successful Test Run

```
========================================
E2E Matrix Testing
========================================
Grafana Image: grafana-enterprise
Versions to test: 10.0.0 10.4.0 11.0.0 11.1.0 12.0.0
========================================

[1/5] Testing Grafana 10.0.0
✓ Grafana is ready
✓ Tests passed for Grafana 10.0.0 (45s)

[2/5] Testing Grafana 10.4.0
✓ Grafana is ready
✓ Tests passed for Grafana 10.4.0 (42s)

...

========================================
Test Summary
========================================
Passed (5/5):
  ✓ 10.0.0
  ✓ 10.4.0
  ✓ 11.0.0
  ✓ 11.1.0
  ✓ 12.0.0
```

### Failed Test Run

```
[3/5] Testing Grafana 11.0.0
✓ Grafana is ready
✗ Tests failed for Grafana 11.0.0 (38s)
Saving Grafana logs to e2e-matrix-results/grafana_11.0.0_20260111_143022.log

========================================
Test Summary
========================================
Passed (2/5):
  ✓ 10.0.0
  ✓ 10.4.0

Failed (3/5):
  ✗ 11.0.0
  ✗ 11.1.0
  ✗ 12.0.0
```

### Results Directory Structure

```
e2e-matrix-results/
├── results_20260111_143022.txt                          # Summary
├── test_output_10.0.0_20260111_143022.log              # Test output
├── test_output_10.4.0_20260111_143022.log
├── test_output_11.0.0_20260111_143022.log
├── grafana_11.0.0_20260111_143022.log                  # Grafana logs (failures only)
└── playwright-report_11.0.0_20260111_143022/           # Playwright HTML report
```

## Troubleshooting

### Issue: Script Fails to Start

```bash
# Check Docker is running
docker ps

# Check script is executable
chmod +x e2e-matrix-local.sh

# Run with bash explicitly
bash e2e-matrix-local.sh
```

### Issue: Grafana Won't Start

Check the Grafana logs:

```bash
# While Grafana is starting
docker logs -f jorgeancal-zagalin-app

# Or after failure
cat e2e-matrix-results/grafana_{version}_{timestamp}.log
```

Common issues:
- Port 3000 already in use → Stop other services using port 3000
- Docker out of memory → Increase Docker memory allocation
- Invalid version number → Check version exists on Docker Hub

### Issue: Tests Timeout Waiting for Grafana

Increase the wait timeout in the script:

```bash
# Edit e2e-matrix-local.sh
# Change: local max_attempts=60
# To:     local max_attempts=120  # Wait up to 4 minutes
```

### Issue: Tests Fail for Specific Version

Debug that version specifically:

```bash
# Run just that version with containers kept
KEEP_CONTAINERS=true ./scripts/e2e-matrix-local.sh 11.0.0

# Check Grafana logs
docker logs jorgeancal-zagalin-app

# Check plugin logs
docker exec jorgeancal-zagalin-app cat /var/log/grafana/grafana.log | grep zagalin

# Run tests in headed mode
npm run e2e -- --headed

# Or debug mode
npm run e2e -- --debug
```

### Issue: Out of Disk Space

Clean up old test results and Docker resources:

```bash
# Remove old test results
rm -rf e2e-matrix-results/*

# Clean Docker
docker system prune -af
docker volume prune -f
```

## CI vs Local Testing

### Differences

| Aspect | CI (GitHub Actions) | Local (This Script) |
|--------|-------------------|-------------------|
| Versions | Dynamic (via `grafana/plugin-actions/e2e-version`) | Fixed list (configurable) |
| Parallelization | Runs in parallel (matrix) | Runs sequentially |
| Resources | Fresh VM per version | Single machine, reused |
| Results | GitHub Actions UI + Pages | Local files |
| Speed | Faster (parallel) | Slower (sequential) |

### Why Use Local Testing?

- **Debug failures** before pushing to CI
- **Faster iteration** - no waiting for CI queue
- **Test unreleased versions** - test against any Docker tag
- **Cost savings** - no CI minutes used
- **Offline development** - works without network (with pre-pulled images)

## Best Practices

### Before Pushing Changes

Run a quick smoke test on key versions:

```bash
# Test minimum and latest versions only
./scripts/e2e-matrix-local.sh 10.4.0 12.0.0
```

### Before Release

Run full matrix test:

```bash
# Test all supported versions
./scripts/e2e-matrix-local.sh

# Review all results
cat e2e-matrix-results/results_*.txt
```

### When Debugging Failures

Test one version at a time with debugging:

```bash
# Keep container running
KEEP_CONTAINERS=true ./scripts/e2e-matrix-local.sh 11.0.0

# Access Grafana
open http://localhost:3000

# Check logs
docker logs -f jorgeancal-zagalin-app

# Run tests with UI
npm run e2e -- --headed

# Stop when done
docker compose down
```

## Performance Optimization

### Pre-pull Docker Images

Save time by pre-pulling all images:

```bash
# Pull all versions beforehand
docker pull grafana/grafana-enterprise:10.0.0
docker pull grafana/grafana-enterprise:10.4.0
docker pull grafana/grafana-enterprise:11.0.0
docker pull grafana/grafana-enterprise:11.1.0
docker pull grafana/grafana-enterprise:12.0.0
```

### Parallel Testing (Advanced)

For faster testing, run multiple versions in parallel using different ports:

```bash
# Terminal 1 - Port 3000
GRAFANA_VERSION=10.4.0 docker compose up -d
npm run e2e

# Terminal 2 - Port 3001 (requires modified docker-compose)
# Not recommended - complex setup, risk of resource conflicts
```

**Note**: Parallel testing is complex and not recommended for local use. Let CI handle parallel execution.

## Adding New Versions

To test against a new Grafana version:

1. **Edit the script** (`e2e-matrix-local.sh`):

```bash
GRAFANA_VERSIONS=(
  "10.0.0"
  "10.4.0"
  "11.0.0"
  "11.1.0"
  "12.0.0"
  "12.1.0"  # Add new version
)
```

2. **Verify version exists**:

```bash
# Check Docker Hub for the version
docker pull grafana/grafana-enterprise:12.1.0
```

3. **Run tests**:

```bash
./scripts/e2e-matrix-local.sh
```

## Integration with CI

The local script mimics CI behavior but runs sequentially. The CI pipeline (`.github/workflows/ci.yml`) uses this workflow:

1. **Build job** - Build plugin once
2. **Resolve versions** - Dynamic version matrix
3. **Playwright tests** - Parallel execution across versions
4. **Publish report** - Aggregate results

To ensure local tests match CI:

- Use same `GRAFANA_IMAGE` (default: `grafana-enterprise`)
- Test same version range
- Review CI failures locally before debugging

## Summary

| Method | When to Use | Command |
|--------|------------|---------|
| **Automated script** | Regular testing | `./scripts/e2e-matrix-local.sh` |
| **Specific versions** | Quick smoke test | `./scripts/e2e-matrix-local.sh 10.4.0 12.0.0` |
| **Manual** | Debugging single version | `GRAFANA_VERSION=11.0.0 npm run server` |
| **CI** | Before merge, release | Push to GitHub |

**Recommendation**: Use the automated script for comprehensive local testing, then rely on CI for final validation.

---

**Last Updated**: 2026-01-11
**Script Version**: 1.0.0
