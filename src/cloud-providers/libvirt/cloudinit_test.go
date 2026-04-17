// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	CR "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"libvirt.org/go/libvirtxml"

	"github.com/kdomanski/iso9660"
)

func TestCloudInit(t *testing.T) {

	file, err := os.CreateTemp("", "CloudInit-*.iso")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	fmt.Printf("temp file: %s", file.Name())

	userDataContent := []byte("userdata")
	metaDataContent := []byte("metadata")

	isoData, err := createCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)

	err = os.WriteFile(file.Name(), isoData, os.ModePerm)
	require.NoError(t, err)

	isoFile, err := os.Open(file.Name())
	require.NoError(t, err)

	isoImg, err := iso9660.OpenImage(isoFile)
	require.NoError(t, err)

	rootFile, err := isoImg.RootDir()
	require.NoError(t, err)

	children, err := rootFile.GetChildren()
	require.NoError(t, err)

	files := make(map[string][]byte)
	for _, child := range children {
		key := child.Name()
		data, err := io.ReadAll(child.Reader())
		require.NoError(t, err)

		files[key] = data
	}

	assert.Equal(t, userDataContent, files[userDataFilename])
	assert.Equal(t, metaDataContent, files[metaDataFilename])

	err = isoFile.Close()
	require.NoError(t, err)
}

func TestInMemoryCopier(t *testing.T) {
	// generate some test data
	size := rand.Intn(1000) + 1000
	buf := make([]byte, size)
	_, err := CR.Read(buf)
	require.NoError(t, err)
	// build the image abstraction
	img, err := newImageFromBytes(buf)
	require.NoError(t, err)

	sizeFromImg, err := img.size()
	require.NoError(t, err)
	assert.Equal(t, uint64(size), sizeFromImg)

	var otherBuf []byte
	err = img.importImage(func(rdr io.Reader) error {
		bufRead, err := io.ReadAll(rdr)
		otherBuf = bufRead
		return err
	}, libvirtxml.StorageVolume{})
	require.NoError(t, err)

	assert.Equal(t, buf, otherBuf)
}

func TestCreateCloudInitWithEmptyData(t *testing.T) {
	userDataContent := []byte("")
	metaDataContent := []byte("")

	isoData, err := createCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)
	assert.NotNil(t, isoData)
	assert.Greater(t, len(isoData), 0)
}

func TestCreateCloudInitWithLargeData(t *testing.T) {
	// Create large user data
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}

	userDataContent := largeData
	metaDataContent := []byte("instance-id: test-instance\nlocal-hostname: test-host")

	isoData, err := createCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)
	assert.NotNil(t, isoData)
	assert.Greater(t, len(isoData), len(largeData))
}

func TestCreateCloudInitWithSpecialCharacters(t *testing.T) {
	userDataContent := []byte("#cloud-config\nusers:\n  - name: test\n    ssh-authorized-keys:\n      - ssh-rsa AAAAB3...")
	metaDataContent := []byte("instance-id: test-123\nlocal-hostname: test-host-456")

	isoData, err := createCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)
	assert.NotNil(t, isoData)

	// Verify ISO can be read
	file, err := os.CreateTemp("", "CloudInitSpecial-*.iso")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	err = os.WriteFile(file.Name(), isoData, os.ModePerm)
	require.NoError(t, err)

	isoFile, err := os.Open(file.Name())
	require.NoError(t, err)
	defer isoFile.Close()

	isoImg, err := iso9660.OpenImage(isoFile)
	require.NoError(t, err)

	rootFile, err := isoImg.RootDir()
	require.NoError(t, err)

	children, err := rootFile.GetChildren()
	require.NoError(t, err)
	assert.Equal(t, 3, len(children)) // user-data, meta-data, vendor-data
}

func TestCreateCloudInitVerifyVendorData(t *testing.T) {
	userDataContent := []byte("userdata")
	metaDataContent := []byte("metadata")

	isoData, err := createCloudInit(userDataContent, metaDataContent)
	require.NoError(t, err)

	file, err := os.CreateTemp("", "CloudInitVendor-*.iso")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	err = os.WriteFile(file.Name(), isoData, os.ModePerm)
	require.NoError(t, err)

	isoFile, err := os.Open(file.Name())
	require.NoError(t, err)
	defer isoFile.Close()

	isoImg, err := iso9660.OpenImage(isoFile)
	require.NoError(t, err)

	rootFile, err := isoImg.RootDir()
	require.NoError(t, err)

	children, err := rootFile.GetChildren()
	require.NoError(t, err)

	files := make(map[string][]byte)
	for _, child := range children {
		key := child.Name()
		data, err := io.ReadAll(child.Reader())
		require.NoError(t, err)
		files[key] = data
	}

	// Verify vendor-data exists and is empty
	assert.Contains(t, files, vendorDataFilename)
	assert.Equal(t, []byte{}, files[vendorDataFilename])
}
