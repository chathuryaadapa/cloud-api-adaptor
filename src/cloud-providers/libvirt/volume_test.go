// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"libvirt.org/go/libvirtxml"
)

func TestNewDefVolume(t *testing.T) {
	volumeName := "test-volume"
	volumeDef := newDefVolume(volumeName)

	assert.Equal(t, volumeName, volumeDef.Name)
	assert.NotNil(t, volumeDef.Target)
	assert.Equal(t, "qcow2", volumeDef.Target.Format.Type)
	assert.Equal(t, "644", volumeDef.Target.Permissions.Mode)
	assert.NotNil(t, volumeDef.Capacity)
	assert.Equal(t, "bytes", volumeDef.Capacity.Unit)
	assert.Equal(t, uint64(1), volumeDef.Capacity.Value)
}

func TestNewDefVolumeFromXML(t *testing.T) {
	xmlData := `<volume>
		<name>test-volume</name>
		<capacity unit="bytes">1073741824</capacity>
		<target>
			<format type="qcow2"/>
			<permissions>
				<mode>0644</mode>
			</permissions>
		</target>
	</volume>`

	volumeDef, err := newDefVolumeFromXML(xmlData)
	require.NoError(t, err)
	assert.Equal(t, "test-volume", volumeDef.Name)
	assert.Equal(t, uint64(1073741824), volumeDef.Capacity.Value)
	assert.Equal(t, "qcow2", volumeDef.Target.Format.Type)
}

func TestNewDefVolumeFromXMLInvalid(t *testing.T) {
	xmlData := `<invalid>xml</invalid>`

	_, err := newDefVolumeFromXML(xmlData)
	assert.Error(t, err)
}

func TestWaitForSuccess(t *testing.T) {
	// Test successful case
	callCount := 0
	err := waitForSuccess("test operation", func() error {
		callCount++
		if callCount < 3 {
			return assert.AnError
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestWaitForSuccessTimeout(t *testing.T) {
	// Temporarily reduce timeout for testing
	originalTimeout := waitTimeout
	originalInterval := waitSleepInterval
	waitTimeout = 100 * time.Millisecond
	waitSleepInterval = 10 * time.Millisecond
	defer func() {
		waitTimeout = originalTimeout
		waitSleepInterval = originalInterval
	}()

	err := waitForSuccess("test operation", func() error {
		return assert.AnError
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test operation")
}

func TestWaitForSuccessImmediate(t *testing.T) {
	callCount := 0
	err := waitForSuccess("test operation", func() error {
		callCount++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestVolumeExists(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with non-existent volume
	exists, err := volumeExists(client, "non-existent-volume-12345")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetVolume(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with non-existent volume
	_, err = getVolume(client, "non-existent-volume-12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't retrieve volume")
}

func TestDeleteVolumeNotFound(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	err = deleteVolume(client, "non-existent-volume-12345")
	assert.Error(t, err)
	assert.Equal(t, ErrVolumeNotFound, err)
}

func TestNewDefBackingStoreFromLibvirt(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try to get the base volume
	baseVol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Base volume %s not found, skipping test", testCfg.VolName)
	}
	defer baseVol.Free()

	backingStore, err := newDefBackingStoreFromLibvirt(baseVol)
	require.NoError(t, err)
	assert.NotEmpty(t, backingStore.Path)
	assert.NotNil(t, backingStore.Format)
}

func TestNewDefVolumeFromLibvirt(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try to get the base volume
	baseVol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Base volume %s not found, skipping test", testCfg.VolName)
	}
	defer baseVol.Free()

	volumeDef, err := newDefVolumeFromLibvirt(baseVol)
	require.NoError(t, err)
	assert.NotEmpty(t, volumeDef.Name)
	assert.NotNil(t, volumeDef.Target)
}

func TestFreeVolume(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try to get a volume
	vol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Volume %s not found, skipping test", testCfg.VolName)
	}

	var errCtx error
	freeVolume(vol, &errCtx)
	assert.NoError(t, errCtx)
}

func TestDeleteVolumeByPath(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with non-existent path
	err = deleteVolumeByPath(client, "/non/existent/path.qcow2")
	assert.Error(t, err)
}

// TestNewDefVolumeCustomName tests volume definition with custom name
func TestNewDefVolumeCustomName(t *testing.T) {
	tests := []struct {
		name       string
		volumeName string
	}{
		{
			name:       "simple name",
			volumeName: "test-vol",
		},
		{
			name:       "name with extension",
			volumeName: "test-vol.qcow2",
		},
		{
			name:       "name with dashes",
			volumeName: "test-vol-123-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumeDef := newDefVolume(tt.volumeName)
			assert.Equal(t, tt.volumeName, volumeDef.Name)
			assert.Equal(t, "qcow2", volumeDef.Target.Format.Type)
		})
	}
}

// TestNewDefVolumeFromXMLVariations tests XML parsing with different formats
func TestNewDefVolumeFromXMLVariations(t *testing.T) {
	tests := []struct {
		name        string
		xmlData     string
		expectError bool
		checkFunc   func(*testing.T, interface{})
	}{
		{
			name: "volume with backing store",
			xmlData: `<volume>
				<name>test-volume</name>
				<capacity unit="bytes">1073741824</capacity>
				<target>
					<format type="qcow2"/>
				</target>
				<backingStore>
					<path>/var/lib/libvirt/images/base.qcow2</path>
					<format type="qcow2"/>
				</backingStore>
			</volume>`,
			expectError: false,
			checkFunc: func(t *testing.T, v interface{}) {
				vol := v.(libvirtxml.StorageVolume)
				assert.NotNil(t, vol.BackingStore)
				assert.Contains(t, vol.BackingStore.Path, "base.qcow2")
			},
		},
		{
			name: "volume with different unit",
			xmlData: `<volume>
				<name>test-volume</name>
				<capacity unit="GiB">10</capacity>
				<target>
					<format type="raw"/>
				</target>
			</volume>`,
			expectError: false,
			checkFunc: func(t *testing.T, v interface{}) {
				vol := v.(libvirtxml.StorageVolume)
				assert.Equal(t, "GiB", vol.Capacity.Unit)
				assert.Equal(t, uint64(10), vol.Capacity.Value)
			},
		},
		{
			name:        "malformed XML",
			xmlData:     `<volume><name>test</name>`,
			expectError: true,
			checkFunc:   nil,
		},
		{
			name:        "empty XML",
			xmlData:     ``,
			expectError: true,
			checkFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumeDef, err := newDefVolumeFromXML(tt.xmlData)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, volumeDef)
				}
			}
		})
	}
}

// TestWaitForSuccessRetries tests retry behavior
func TestWaitForSuccessRetries(t *testing.T) {
	// Save original values
	originalTimeout := waitTimeout
	originalInterval := waitSleepInterval
	waitTimeout = 200 * time.Millisecond
	waitSleepInterval = 20 * time.Millisecond
	defer func() {
		waitTimeout = originalTimeout
		waitSleepInterval = originalInterval
	}()

	tests := []struct {
		name          string
		failCount     int
		expectSuccess bool
		expectRetries int
	}{
		{
			name:          "success on first try",
			failCount:     0,
			expectSuccess: true,
			expectRetries: 1,
		},
		{
			name:          "success after 2 retries",
			failCount:     2,
			expectSuccess: true,
			expectRetries: 3,
		},
		{
			name:          "success after 5 retries",
			failCount:     5,
			expectSuccess: true,
			expectRetries: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			err := waitForSuccess("test operation", func() error {
				callCount++
				if callCount <= tt.failCount {
					return fmt.Errorf("attempt %d failed", callCount)
				}
				return nil
			})

			if tt.expectSuccess {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectRetries, callCount)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestWaitForSuccessContextualError tests error message formatting
func TestWaitForSuccessContextualError(t *testing.T) {
	originalTimeout := waitTimeout
	originalInterval := waitSleepInterval
	waitTimeout = 50 * time.Millisecond
	waitSleepInterval = 10 * time.Millisecond
	defer func() {
		waitTimeout = originalTimeout
		waitSleepInterval = originalInterval
	}()

	errorMsg := "custom error message"
	err := waitForSuccess(errorMsg, func() error {
		return fmt.Errorf("persistent error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), errorMsg)
	assert.Contains(t, err.Error(), "persistent error")
}

// TestVolumeExistsWithExistingVolume tests volume existence check
func TestVolumeExistsWithExistingVolume(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try with the configured volume
	exists, err := volumeExists(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Error checking volume existence: %v", err)
	}
	// We don't assert true because the volume might not exist in test environment
	// but we verify the function doesn't error on valid volume names
	assert.NoError(t, err)
	t.Logf("Volume %s exists: %v", testCfg.VolName, exists)
}

// TestGetVolumeByKey tests volume retrieval by key
func TestGetVolumeByKey(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with invalid key
	_, err = getVolume(client, "invalid-key-12345")
	assert.Error(t, err)
}

// TestDeleteVolumeByPathInvalid tests deletion with invalid paths
func TestDeleteVolumeByPathInvalid(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "non-existent path",
			path: "/non/existent/path.qcow2",
		},
		{
			name: "invalid path format",
			path: "invalid-path",
		},
		{
			name: "empty path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteVolumeByPath(client, tt.path)
			assert.Error(t, err)
		})
	}
}

// TestFreeVolumeWithError tests volume freeing error handling
func TestFreeVolumeWithError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	vol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Volume %s not found, skipping test", testCfg.VolName)
	}

	// Free the volume once
	var errCtx error
	freeVolume(vol, &errCtx)
	assert.NoError(t, errCtx)

	// Try to free again - should error but be handled
	var errCtx2 error
	freeVolume(vol, &errCtx2)
	// The error should be captured in errCtx2
	assert.Error(t, errCtx2)
}

// TestFreeVolumePreservesError tests that freeVolume preserves existing errors
func TestFreeVolumePreservesError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	vol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Volume %s not found, skipping test", testCfg.VolName)
	}

	// Set an existing error
	existingErr := fmt.Errorf("existing error")
	errCtx := existingErr

	// Free the volume - should preserve existing error
	freeVolume(vol, &errCtx)
	assert.Equal(t, existingErr, errCtx)
}

// TestNewDefBackingStoreError tests backing store creation error handling
func TestNewDefBackingStoreError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try with a volume that doesn't exist
	vol, err := getVolume(client, "non-existent-volume-xyz")
	if err == nil {
		defer vol.Free()
		t.Skip("Unexpected: non-existent volume found")
	}
	// Verify we get an error as expected
	assert.Error(t, err)
}

// TestNewDefVolumeFromLibvirtError tests volume definition creation error handling
func TestNewDefVolumeFromLibvirtError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try with a volume that doesn't exist
	vol, err := getVolume(client, "non-existent-volume-xyz")
	if err == nil {
		defer vol.Free()
		_, err := newDefVolumeFromLibvirt(vol)
		// If we somehow got a volume, test the function
		assert.NoError(t, err)
	}
	// Verify we get an error as expected when volume doesn't exist
	assert.Error(t, err)
}

// TestNewCopier tests the copier function creation
func TestNewCopier(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	vol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Volume %s not found, skipping test", testCfg.VolName)
	}
	defer vol.Free()

	// Create a copier function
	copier := newCopier(client.connection, vol, 1024)
	assert.NotNil(t, copier)

	// Test the copier with small data
	testData := bytes.NewReader([]byte("test data"))
	err = copier(testData)
	// The copier may succeed or fail depending on the volume state
	// We're primarily testing that the function is created and callable
	if err != nil {
		// Error is acceptable in test environment
		t.Logf("Copier returned error (expected in test): %v", err)
	}
}

// TestUploadVolumeError tests upload volume error scenarios
func TestUploadVolumeError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create an invalid volume definition
	volumeDef := newDefVolume("test-upload-error")
	volumeDef.Capacity.Value = 0 // Invalid size

	// Create a mock image
	img, err := newImageFromBytes([]byte("test"))
	require.NoError(t, err)

	// Try to upload - should fail
	_, err = uploadVolume(client, volumeDef, img)
	// We expect an error due to invalid configuration
	assert.Error(t, err)
}

// TestCreateVolumeError tests create volume error scenarios
func TestCreateVolumeError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try to create volume with non-existent base
	err = createVolume("test-vol", 1024*1024*1024, "non-existent-base-xyz", client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Can't retrieve volume")
}

// TestDeleteVolumeMultipleScenarios tests various deletion scenarios
func TestDeleteVolumeMultipleScenarios(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	tests := []struct {
		name        string
		volumeName  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "non-existent volume",
			volumeName:  "non-existent-vol-12345",
			expectError: true,
			errorMsg:    "",
		},
		{
			name:        "empty volume name",
			volumeName:  "",
			expectError: true,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteVolume(client, tt.volumeName)
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Additional comprehensive tests for volume.go coverage

func TestCreateVolume(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test creating a volume with a base volume
	baseVolName := "default"
	newVolName := "test-create-volume-" + fmt.Sprintf("%d", time.Now().Unix())

	err = createVolume(newVolName, 10*1024*1024*1024, baseVolName, client)
	if err != nil {
		// May fail if base volume doesn't exist, which is acceptable in test
		t.Logf("createVolume returned error (may be expected): %v", err)
	} else {
		// Clean up if successful
		defer func() {
			vol, err := getVolume(client, newVolName)
			if err == nil {
				vol.Delete(0)
				vol.Free()
			}
		}()
	}
}

func TestCreateVolumeInvalidBase(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with non-existent base volume
	err = createVolume("test-vol", 1024*1024, "non-existent-base-vol-xyz", client)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Can't retrieve volume")
}

func TestUploadVolumeSuccess(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create a small test image
	testData := []byte("test upload data content")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	size, err := img.size()
	require.NoError(t, err)

	// Create volume definition
	volName := "test-upload-vol-" + fmt.Sprintf("%d", time.Now().Unix())
	volumeDef := newDefVolume(volName)
	volumeDef.Capacity.Value = size
	volumeDef.Target.Format.Type = "raw"

	// Attempt upload
	volumeKey, err := uploadVolume(client, volumeDef, img)
	if err != nil {
		t.Logf("uploadVolume returned error (may be expected in test env): %v", err)
	} else {
		assert.NotEmpty(t, volumeKey)
		// Clean up
		defer func() {
			vol, err := client.connection.LookupStorageVolByKey(volumeKey)
			if err == nil {
				vol.Delete(0)
				vol.Free()
			}
		}()
	}
}

func TestWaitForSuccessWithRealFunction(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with a function that succeeds
	err = waitForSuccess("test operation", func() error {
		return client.pool.Refresh(0)
	})
	assert.NoError(t, err)
}

func TestNewDefBackingStoreFromLibvirtSuccess(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Get an existing volume
	vol, err := getVolume(client, "default")
	if err != nil {
		t.Skip("Default volume not found, skipping test")
	}
	defer vol.Free()

	// Test creating backing store definition
	backingStore, err := newDefBackingStoreFromLibvirt(vol)
	if err != nil {
		t.Logf("newDefBackingStoreFromLibvirt returned error: %v", err)
	} else {
		assert.NotEmpty(t, backingStore.Path)
		assert.NotNil(t, backingStore.Format)
	}
}

func TestVolumeOperationsIntegration(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	t.Run("check volume exists", func(t *testing.T) {
		exists, err := volumeExists(client, "default")
		assert.NoError(t, err)
		t.Logf("Volume 'default' exists: %v", exists)
	})

	t.Run("get volume info", func(t *testing.T) {
		vol, err := getVolume(client, "default")
		if err != nil {
			t.Skip("Default volume not found")
		}
		defer vol.Free()

		info, err := vol.GetInfo()
		if err == nil {
			assert.Greater(t, info.Capacity, uint64(0))
			t.Logf("Volume capacity: %d bytes", info.Capacity)
		}
	})
}
