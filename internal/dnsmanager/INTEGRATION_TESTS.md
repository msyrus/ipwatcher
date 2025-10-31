# DNS Manager Integration Tests

This document describes the integration tests for the dnsmanager package and how to run them.

## Overview

The integration tests in `dnsmanager_integration_test.go` test the actual Cloudflare API integration to ensure the DNS manager works correctly with real Cloudflare zones and DNS records.

## Prerequisites

To run the integration tests, you need:

1. A Cloudflare account with API access
2. A zone (domain) configured in Cloudflare
3. An API token with the following permissions:
   - Zone:Read
   - DNS:Read
   - DNS:Edit

## Environment Variables

Set the following environment variables before running the integration tests:

```bash
export CLOUDFLARE_API_TOKEN="your-api-token-here"
export CLOUDFLARE_TEST_ZONE_ID="your-zone-id-here"
export CLOUDFLARE_TEST_ZONE_NAME="example.com"  # Your actual domain
```

### Getting Your Zone ID

You can get your zone ID from:

1. Cloudflare Dashboard → Select your domain → Overview → Zone ID (in the right sidebar)
2. Or use the Cloudflare API:

   ```bash
   curl -X GET "https://api.cloudflare.com/client/v4/zones?name=example.com" \
     -H "Authorization: Bearer YOUR_API_TOKEN" \
     -H "Content-Type: application/json"
   ```

### Creating an API Token

1. Go to [Cloudflare Dashboard](https://dash.cloudflare.com/)
2. Click on your profile icon → My Profile → API Tokens
3. Click "Create Token"
4. Use "Edit zone DNS" template or create a custom token with:
   - Permissions:
     - Zone → Zone → Read
     - Zone → DNS → Read
     - Zone → DNS → Edit
   - Zone Resources:
     - Include → Specific zone → (select your test zone)
5. Copy the generated token

## Running the Tests

### Run all integration tests

Integration tests are isolated using Go build tags and run **sequentially** to avoid race conditions when modifying DNS records. To run them:

```bash
# Using the Makefile (recommended - runs tests sequentially)
make test-integration

# Or directly with go test (sequential execution)
go test -v -p 1 -parallel 1 -tags=integration ./internal/dnsmanager/
```

**Note:** The `-p 1 -parallel 1` flags ensure tests run sequentially, preventing race conditions when creating/updating/deleting DNS records in the same Cloudflare zone.

### Run only unit tests (skip integration tests)

By default, integration tests are excluded:

```bash
# Run only unit tests (integration tests excluded by default)
go test -v ./internal/dnsmanager/

# Or run all unit tests across the project
go test ./...
```

### Run with short mode

You can also use short mode to skip long-running tests:

```bash
go test -short ./internal/dnsmanager/
```

### Run a specific integration test

```bash
go test -v -p 1 -parallel 1 -tags=integration ./internal/dnsmanager/ -run TestIntegration_GetZoneIDByName
```

## Build Tags

Integration tests use the `integration` build tag:

```go
//go:build integration
// +build integration
```

This allows you to:

- Run unit tests by default without integration tests
- Explicitly include integration tests with `-tags=integration`
- Keep integration tests separate in CI/CD pipelines

## Test Coverage

The integration tests cover the following scenarios:

### 1. GetZoneIDByName

- **TestIntegration_GetZoneIDByName**: Verifies zone ID lookup by name
- **TestIntegration_GetZoneIDByName_NotFound**: Tests error handling for nonexistent zones

### 2. GetDNSRecords

- **TestIntegration_GetDNSRecords**: Retrieves all A and AAAA records from the zone

### 3. EnsureDNSRecords - Create and Update

- **TestIntegration_EnsureDNSRecords_CreateAndUpdate**:
  - Creates new A and AAAA records
  - Verifies they were created correctly
  - Updates the records with new IP addresses
  - Verifies the updates
  - Cleans up test records

### 4. EnsureDNSRecords - No Updates Needed

- **TestIntegration_EnsureDNSRecords_NoUpdatesNeeded**:
  - Creates a record
  - Calls EnsureDNSRecords again with the same IP (should skip update)
  - Verifies no errors occur

### 5. EnsureDNSRecords - Proxied Status

- **TestIntegration_EnsureDNSRecords_ProxiedToggle**:
  - Creates a record with proxied=false
  - Updates to proxied=true
  - Verifies the proxied status was updated

### 6. EnsureDNSRecords - Empty IPs

- **TestIntegration_EnsureDNSRecords_EmptyIPs**:
  - Tests that empty IPs are handled gracefully (records skipped)

## Test Isolation

All integration tests:

- Use unique subdomain names with timestamps to avoid conflicts
- Clean up created records after test completion
- Are safe to run in parallel (each test uses unique record names)
- Will not affect existing DNS records in your zone

Test records are created with names like:

- `ipwatcher-test-20231027-143025.example.com`
- `ipwatcher-test-noupdate-20231027-143030.example.com`
- `ipwatcher-test-proxy-20231027-143035.example.com`

## Important Notes

⚠️ **Warning**: These tests will create and delete actual DNS records in your Cloudflare zone. While they clean up after themselves, use a test zone or subdomain to be safe.

⚠️ **Rate Limits**: Cloudflare has API rate limits. If you run the tests too frequently, you may hit rate limits.

⚠️ **DNS Propagation**: Tests include 2-second delays to allow for DNS propagation within Cloudflare's systems.

## Troubleshooting

### Tests are skipped

- Ensure all three environment variables are set
- Check that you're not running with `-short` flag

### API Token errors

- Verify your API token has the correct permissions
- Check that the token hasn't expired
- Ensure the token has access to the specific zone

### Zone not found

- Verify `CLOUDFLARE_TEST_ZONE_NAME` matches exactly (case-sensitive)
- Check that the zone exists in your Cloudflare account
- Ensure the API token has access to this zone

### Records not cleaned up

- Check the test logs for cleanup errors
- Manually delete test records from Cloudflare Dashboard if needed
- Look for records starting with `ipwatcher-test-`

## CI/CD Integration

Integration tests are now configured in the GitHub Actions workflow and run automatically when you push to the main branch.

### GitHub Actions

The workflow includes a dedicated integration test job that:

- Only runs on the main repository (not on forks)
- Uses GitHub secrets for Cloudflare credentials
- Runs in parallel with unit tests
- Must pass before builds are created

Required GitHub repository secrets:

- `CLOUDFLARE_API_TOKEN`
- `CLOUDFLARE_TEST_ZONE_ID`
- `CLOUDFLARE_TEST_ZONE_NAME`

See `.github/workflows/build.yml` for the complete workflow configuration.

### Manual CI/CD Setup

If using other CI/CD platforms, here's an example configuration:

```yaml
- name: Run Integration Tests
  env:
    CLOUDFLARE_API_TOKEN: ${{ secrets.CLOUDFLARE_API_TOKEN }}
    CLOUDFLARE_TEST_ZONE_ID: ${{ secrets.CLOUDFLARE_TEST_ZONE_ID }}
    CLOUDFLARE_TEST_ZONE_NAME: ${{ secrets.CLOUDFLARE_TEST_ZONE_NAME }}
  run: go test -v -tags=integration ./internal/dnsmanager/
```

**Note**: Integration tests use the `integration` build tag, so you must include `-tags=integration` in the test command.
