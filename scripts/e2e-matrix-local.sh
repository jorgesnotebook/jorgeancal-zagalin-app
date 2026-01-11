#!/bin/bash

# E2E Matrix Testing Script
# Runs E2E tests against multiple Grafana versions locally
#
# Usage:
#   ./e2e-matrix-local.sh                    # Test all versions
#   ./e2e-matrix-local.sh 11.0.0 12.0.0     # Test specific versions
#   STOP_ON_FAILURE=true ./e2e-matrix-local.sh  # Stop on first failure

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GRAFANA_VERSIONS=(
  "10.0.0"
  "10.4.0"
  "11.0.0"
  "11.1.0"
  "12.0.0"
)

# Override versions if provided as arguments
if [ $# -gt 0 ]; then
  GRAFANA_VERSIONS=("$@")
fi

# Options
STOP_ON_FAILURE=${STOP_ON_FAILURE:-false}
GRAFANA_IMAGE=${GRAFANA_IMAGE:-grafana-enterprise}
KEEP_CONTAINERS=${KEEP_CONTAINERS:-false}

# Results tracking
RESULTS_DIR="./e2e-matrix-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_FILE="${RESULTS_DIR}/results_${TIMESTAMP}.txt"
FAILED_VERSIONS=()
PASSED_VERSIONS=()

# Create results directory
mkdir -p "${RESULTS_DIR}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}E2E Matrix Testing${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Grafana Image: ${GRAFANA_IMAGE}"
echo -e "Versions to test: ${GRAFANA_VERSIONS[*]}"
echo -e "Results file: ${RESULTS_FILE}"
echo -e "${BLUE}========================================${NC}\n"

# Initialize results file
echo "E2E Matrix Test Results - ${TIMESTAMP}" > "${RESULTS_FILE}"
echo "Grafana Image: ${GRAFANA_IMAGE}" >> "${RESULTS_FILE}"
echo "Versions: ${GRAFANA_VERSIONS[*]}" >> "${RESULTS_FILE}"
echo "========================================" >> "${RESULTS_FILE}"
echo "" >> "${RESULTS_FILE}"

# Function to wait for Grafana to be ready
wait_for_grafana() {
  local max_attempts=60
  local attempt=0

  echo -e "${YELLOW}Waiting for Grafana to be ready...${NC}"

  while [ $attempt -lt $max_attempts ]; do
    if curl -s http://localhost:3000/api/health > /dev/null 2>&1; then
      echo -e "${GREEN}✓ Grafana is ready${NC}"
      return 0
    fi

    attempt=$((attempt + 1))
    echo -n "."
    sleep 2
  done

  echo -e "\n${RED}✗ Grafana failed to start after ${max_attempts} attempts${NC}"
  return 1
}

# Function to stop Grafana
stop_grafana() {
  echo -e "${YELLOW}Stopping Grafana...${NC}"
  docker compose down > /dev/null 2>&1 || true
  echo -e "${GREEN}✓ Grafana stopped${NC}"
}

# Function to save Grafana logs
save_grafana_logs() {
  local version=$1
  local log_file="${RESULTS_DIR}/grafana_${version}_${TIMESTAMP}.log"

  echo -e "${YELLOW}Saving Grafana logs to ${log_file}${NC}"
  docker logs jorgeancal-zagalin-app > "${log_file}" 2>&1 || true
}

# Function to test a specific version
test_version() {
  local version=$1
  local start_time=$(date +%s)

  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}Testing Grafana ${version}${NC}"
  echo -e "${BLUE}========================================${NC}"

  # Stop any existing containers
  stop_grafana

  # Start Grafana with specific version
  echo -e "${YELLOW}Starting Grafana ${version}...${NC}"
  if ! GRAFANA_VERSION="${version}" GRAFANA_IMAGE="${GRAFANA_IMAGE}" docker compose up -d; then
    echo -e "${RED}✗ Failed to start Grafana ${version}${NC}"
    echo "FAILED: ${version} - Failed to start Grafana" >> "${RESULTS_FILE}"
    FAILED_VERSIONS+=("${version}")
    return 1
  fi

  # Wait for Grafana to be ready
  if ! wait_for_grafana; then
    echo -e "${RED}✗ Grafana ${version} did not become ready${NC}"
    save_grafana_logs "${version}"
    echo "FAILED: ${version} - Grafana did not start" >> "${RESULTS_FILE}"
    FAILED_VERSIONS+=("${version}")

    if [ "${STOP_ON_FAILURE}" = "true" ]; then
      stop_grafana
      exit 1
    fi

    return 1
  fi

  # Run E2E tests
  echo -e "${YELLOW}Running E2E tests...${NC}"
  local test_result=0

  if npm run e2e 2>&1 | tee "${RESULTS_DIR}/test_output_${version}_${TIMESTAMP}.log"; then
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    echo -e "${GREEN}✓ Tests passed for Grafana ${version} (${duration}s)${NC}"
    echo "PASSED: ${version} (${duration}s)" >> "${RESULTS_FILE}"
    PASSED_VERSIONS+=("${version}")
  else
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    test_result=1
    echo -e "${RED}✗ Tests failed for Grafana ${version} (${duration}s)${NC}"
    echo "FAILED: ${version} (${duration}s)" >> "${RESULTS_FILE}"
    FAILED_VERSIONS+=("${version}")

    # Save Grafana logs on test failure
    save_grafana_logs "${version}"

    # Copy Playwright report
    if [ -d "playwright-report" ]; then
      cp -r playwright-report "${RESULTS_DIR}/playwright-report_${version}_${TIMESTAMP}"
    fi

    if [ "${STOP_ON_FAILURE}" = "true" ]; then
      stop_grafana
      exit 1
    fi
  fi

  # Stop Grafana (unless KEEP_CONTAINERS is true)
  if [ "${KEEP_CONTAINERS}" != "true" ]; then
    stop_grafana
  fi

  return $test_result
}

# Main execution
main() {
  local total_versions=${#GRAFANA_VERSIONS[@]}
  local current=0

  for version in "${GRAFANA_VERSIONS[@]}"; do
    current=$((current + 1))
    echo -e "\n${BLUE}[${current}/${total_versions}]${NC}"
    test_version "${version}" || true
  done

  # Summary
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}Test Summary${NC}"
  echo -e "${BLUE}========================================${NC}"

  echo "" >> "${RESULTS_FILE}"
  echo "========================================" >> "${RESULTS_FILE}"
  echo "Summary:" >> "${RESULTS_FILE}"

  if [ ${#PASSED_VERSIONS[@]} -gt 0 ]; then
    echo -e "${GREEN}Passed (${#PASSED_VERSIONS[@]}/${total_versions}):${NC}"
    echo "Passed (${#PASSED_VERSIONS[@]}/${total_versions}):" >> "${RESULTS_FILE}"
    for version in "${PASSED_VERSIONS[@]}"; do
      echo -e "  ${GREEN}✓${NC} ${version}"
      echo "  ✓ ${version}" >> "${RESULTS_FILE}"
    done
  fi

  if [ ${#FAILED_VERSIONS[@]} -gt 0 ]; then
    echo -e "${RED}Failed (${#FAILED_VERSIONS[@]}/${total_versions}):${NC}"
    echo "Failed (${#FAILED_VERSIONS[@]}/${total_versions}):" >> "${RESULTS_FILE}"
    for version in "${FAILED_VERSIONS[@]}"; do
      echo -e "  ${RED}✗${NC} ${version}"
      echo "  ✗ ${version}" >> "${RESULTS_FILE}"
    done
  fi

  echo -e "${BLUE}========================================${NC}"
  echo -e "Results saved to: ${RESULTS_FILE}"
  echo -e "Logs saved in: ${RESULTS_DIR}/"

  # Exit with error if any tests failed
  if [ ${#FAILED_VERSIONS[@]} -gt 0 ]; then
    exit 1
  fi
}

# Trap to ensure cleanup on exit
trap 'stop_grafana' EXIT INT TERM

# Run main
main
