// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	libvirtxml "libvirt.org/go/libvirtxml"
)

func TestNewImageFromBytes(t *testing.T) {
	testData := []byte("test image data")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)
	assert.NotNil(t, img)
}

func TestInMemoryImageSize(t *testing.T) {
	testData := []byte("test image data with some content")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	size, err := img.size()
	require.NoError(t, err)
	assert.Equal(t, uint64(len(testData)), size)
}

func TestInMemoryImageSizeEmpty(t *testing.T) {
	testData := []byte{}
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	size, err := img.size()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), size)
}

func TestInMemoryImageSizeLarge(t *testing.T) {
	// Test with 10MB of data
	testData := make([]byte, 10*1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	size, err := img.size()
	require.NoError(t, err)
	assert.Equal(t, uint64(len(testData)), size)
}

func TestInMemoryImageString(t *testing.T) {
	testData := []byte("test data")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	str := img.string()
	assert.Contains(t, str, "plain bytes")
	assert.Contains(t, str, "9") // size of "test data"
}

func TestInMemoryImageImportImage(t *testing.T) {
	testData := []byte("test image content for import")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	var capturedData []byte
	copier := func(rdr io.Reader) error {
		data, err := io.ReadAll(rdr)
		if err != nil {
			return err
		}
		capturedData = data
		return nil
	}

	volumeDef := libvirtxml.StorageVolume{
		Name: "test-volume",
	}

	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
	assert.Equal(t, testData, capturedData)
}

func TestInMemoryImageImportImageError(t *testing.T) {
	testData := []byte("test data")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	copier := func(rdr io.Reader) error {
		return assert.AnError
	}

	volumeDef := libvirtxml.StorageVolume{
		Name: "test-volume",
	}

	err = img.importImage(copier, volumeDef)
	assert.Error(t, err)
}

func TestInMemoryImageImportImagePartialRead(t *testing.T) {
	testData := []byte("test data for partial read")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	var capturedData []byte
	copier := func(rdr io.Reader) error {
		// Read only first 5 bytes
		buf := make([]byte, 5)
		n, err := rdr.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		capturedData = buf[:n]
		return nil
	}

	volumeDef := libvirtxml.StorageVolume{
		Name: "test-volume",
	}

	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
	assert.Equal(t, []byte("test "), capturedData)
}

func TestInMemoryImageImportImageMultipleCalls(t *testing.T) {
	testData := []byte("test data")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	callCount := 0
	copier := func(rdr io.Reader) error {
		callCount++
		_, err := io.ReadAll(rdr)
		return err
	}

	volumeDef := libvirtxml.StorageVolume{
		Name: "test-volume",
	}

	// Call import multiple times
	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestInMemoryImageInterface(t *testing.T) {
	testData := []byte("test")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	// Verify it implements the image interface
	var _ image = img
}

func TestInMemoryImageWithBinaryData(t *testing.T) {
	// Test with binary data (not just text)
	testData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	size, err := img.size()
	require.NoError(t, err)
	assert.Equal(t, uint64(6), size)

	var capturedData []byte
	copier := func(rdr io.Reader) error {
		data, err := io.ReadAll(rdr)
		capturedData = data
		return err
	}

	volumeDef := libvirtxml.StorageVolume{}
	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
	assert.Equal(t, testData, capturedData)
}

func TestInMemoryImageReaderBehavior(t *testing.T) {
	testData := []byte("test data for reader behavior")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	copier := func(rdr io.Reader) error {
		// Test that we can read in chunks
		buf := make([]byte, 5)
		totalRead := 0

		for {
			n, err := rdr.Read(buf)
			totalRead += n
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}

		assert.Equal(t, len(testData), totalRead)
		return nil
	}

	volumeDef := libvirtxml.StorageVolume{}
	err = img.importImage(copier, volumeDef)
	require.NoError(t, err)
}

func TestInMemoryImageConcurrentAccess(t *testing.T) {
	testData := []byte("concurrent test data")
	img, err := newImageFromBytes(testData)
	require.NoError(t, err)

	// Test that multiple goroutines can access size() safely
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			size, err := img.size()
			assert.NoError(t, err)
			assert.Equal(t, uint64(len(testData)), size)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestInMemoryImageStringWithLargeData(t *testing.T) {
	largeData := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
	img, err := newImageFromBytes(largeData)
	require.NoError(t, err)

	str := img.string()
	assert.Contains(t, str, "1048576") // 1MB in bytes
}
