# Testing Guide for DevPod Oracle Provider

This document outlines the testing strategy for the DevPod Oracle Provider.

## Unit Tests

Unit tests are located in the same package as the code they test, with a `_test.go` suffix. For example, the tests for `pkg/oracle/oracle.go` are in `pkg/oracle/oracle_test.go`.

To run all unit tests:

```bash
go test ./...
```

To run tests for a specific package:

```bash
go test ./pkg/oracle
```

To run a specific test:

```bash
go test ./pkg/oracle -run TestFingerPrintGenerate
```

## Mock Testing

For more complex unit tests that require mocking the Oracle Cloud Infrastructure API, we use the `testify/mock` package. This allows us to test our code without making actual API calls.

Example of a mock test:

```go
// Create a mock for the OCI compute client
type MockComputeClient struct {
    mock.Mock
}

// Implement the necessary methods from the OCI SDK interfaces
func (m *MockComputeClient) ListInstances(ctx context.Context, request core.ListInstancesRequest) (core.ListInstancesResponse, error) {
    args := m.Called(ctx, request)
    return args.Get(0).(core.ListInstancesResponse), args.Error(1)
}

// Test using the mock
func TestGetInstance(t *testing.T) {
    mockClient := new(MockComputeClient)
    
    // Setup expectations
    mockClient.On("ListInstances", mock.Anything, mock.Anything).Return(
        core.ListInstancesResponse{
            Items: []core.Instance{
                {
                    Id: common.String("instance-id"),
                    FreeformTags: map[string]string{
                        labelMachineID: "test-machine-id",
                    },
                },
            },
        }, nil)
    
    // Create Oracle instance with mock
    o := &Oracle{
        computeClient: mockClient,
    }
    
    // Call the method
    instance, err := o.GetInstance(context.Background(), "test-machine-id")
    
    // Assert results
    assert.NoError(t, err)
    assert.NotNil(t, instance)
    assert.Equal(t, "instance-id", *instance.Id)
    
    // Verify expectations
    mockClient.AssertExpectations(t)
}
```

## Integration Tests

Integration tests interact with the actual Oracle Cloud Infrastructure API. These tests require valid OCI credentials and will create and delete real resources.

To run integration tests:

```bash
# Set required environment variables
export OCI_CONFIG_FILE=~/.oci/config
export OCI_PROFILE=DEFAULT
export COMPARTMENT_ID=ocid1.compartment.oc1..example
export REGION=us-ashburn-1
export AVAILABILITY_DOMAIN=AD-1
export DISK_IMAGE=Oracle-Linux-8.6-2022.05.31-0
export DISK_SIZE=50
export MACHINE_TYPE=VM.Standard.E4.Flex
export MACHINE_ID=test-machine-id
export MACHINE_FOLDER=/tmp/devpod-test

# Run integration tests
go test ./integration -tags=integration
```

**Note**: Integration tests are expensive and should be run sparingly. They are not run as part of the normal test suite.

## Manual Testing

For manual testing, you can use the CLI directly:

```bash
# Set required environment variables (same as for integration tests)

# Create an instance
go run . create

# Check status
go run . status

# Run a command
COMMAND="ls -la" go run . command

# Stop the instance
go run . stop

# Start the instance
go run . start

# Delete the instance
go run . delete
```

## Testing with DevPod

To test the provider with DevPod:

1. Install the provider in DevPod:
   ```bash
   devpod provider add ./provider.yaml
   ```

2. Create a workspace:
   ```bash
   devpod up github.com/microsoft/vscode-remote-try-go --provider oracle
   ```

3. Connect to the workspace:
   ```bash
   devpod ssh github.com/microsoft/vscode-remote-try-go
   ```

4. Delete the workspace:
   ```bash
   devpod down github.com/microsoft/vscode-remote-try-go
   ```

## CI/CD Testing

The CI/CD pipeline runs unit tests on every pull request and push to the main branch. Integration tests are run on a schedule or manually triggered.

### GitHub Actions Workflow

The GitHub Actions workflow is defined in `.github/workflows/test.yml` and includes:

1. Unit tests on multiple Go versions
2. Code linting
3. Integration tests (on schedule or manual trigger)
4. Binary building and testing

## Test Coverage

To check test coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Adding New Tests

When adding new functionality, please follow these guidelines:

1. Add unit tests for all new functions
2. Use mocks for external dependencies
3. Add integration tests for critical functionality
4. Update this document if testing procedures change 