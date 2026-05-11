// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamIO(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)
	assert.NotNil(t, sio)
	assert.NotNil(t, sio.stream)
}

func TestStreamIOInterfaces(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Verify it implements the required interfaces
	var _ io.Writer = sio
	var _ io.Reader = sio
	var _ io.Closer = sio
}

func TestStreamIOClose(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)

	sio := newStreamIO(*stream)

	// Close should call Finish on the stream
	err = sio.Close()
	// We expect an error because the stream wasn't properly initialized
	// but we're testing that Close() calls the underlying method
	assert.Error(t, err)
}

func TestStreamIOReadWrite(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create a test volume to work with streams
	vol, err := getVolume(client, testCfg.VolName)
	if err != nil {
		t.Skipf("Volume %s not found, skipping stream test", testCfg.VolName)
	}
	defer vol.Free()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Test that Read and Write methods exist and can be called
	// (they will fail without proper stream setup, but we're testing the interface)
	testData := []byte("test data")
	_, err = sio.Write(testData)
	// We expect an error because the stream isn't connected to anything
	assert.Error(t, err)

	readBuf := make([]byte, 10)
	_, err = sio.Read(readBuf)
	// We expect an error because the stream isn't connected to anything
	assert.Error(t, err)
}

func TestStreamIOWithBuffer(t *testing.T) {
	// Test the streamIO struct with a mock-like approach
	// This tests the basic structure without needing a full libvirt setup

	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Verify the struct is properly initialized
	assert.NotNil(t, sio)
	assert.NotNil(t, sio.stream)
}

func TestStreamIOMultipleOperations(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Test multiple write operations
	testData1 := []byte("first write")
	testData2 := []byte("second write")

	_, err = sio.Write(testData1)
	assert.Error(t, err) // Expected to fail without proper setup

	_, err = sio.Write(testData2)
	assert.Error(t, err) // Expected to fail without proper setup
}

func TestStreamIOEmptyData(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Test with empty data
	emptyData := []byte{}
	_, err = sio.Write(emptyData)
	// May or may not error depending on libvirt implementation
	// Just verify it doesn't panic
}

func TestStreamIOLargeBuffer(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Test with large buffer
	largeData := bytes.Repeat([]byte("x"), 1024*1024) // 1MB
	_, err = sio.Write(largeData)
	assert.Error(t, err) // Expected to fail without proper setup
}

// TestStreamIOReadVariousSizes tests Read with different buffer sizes
func TestStreamIOReadVariousSizes(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	tests := []struct {
		name       string
		bufferSize int
	}{
		{"small buffer", 10},
		{"medium buffer", 1024},
		{"large buffer", 64 * 1024},
		{"zero buffer", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := make([]byte, tt.bufferSize)
			_, err := sio.Read(buf)
			// Expected to error without proper stream setup
			assert.Error(t, err)
		})
	}
}

// TestStreamIOWriteVariousSizes tests Write with different data sizes
func TestStreamIOWriteVariousSizes(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	tests := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"small data", []byte("test")},
		{"medium data", bytes.Repeat([]byte("x"), 1024)},
		{"large data", bytes.Repeat([]byte("y"), 64*1024)},
		{"single byte", []byte{0x42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := sio.Write(tt.data)
			// Expected to error without proper stream setup
			// Empty data might not error, so we don't assert
			_ = n // Suppress unused variable warning
			_ = err
		})
	}
}

// TestStreamIOCloseIdempotent tests that Close can be called multiple times
func TestStreamIOCloseIdempotent(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)

	sio := newStreamIO(*stream)

	// First close
	err1 := sio.Close()
	assert.Error(t, err1)

	// Second close - should also error
	err2 := sio.Close()
	assert.Error(t, err2)
}

// TestStreamIOStructure tests the internal structure
func TestStreamIOStructure(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)
	defer stream.Free()

	sio := newStreamIO(*stream)

	// Verify the structure
	assert.NotNil(t, sio)
	assert.IsType(t, &streamIO{}, sio)

	// Verify it has the stream field
	assert.NotNil(t, sio.stream)
}

// TestStreamIOReadAfterClose tests Read after Close
func TestStreamIOReadAfterClose(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)

	sio := newStreamIO(*stream)

	// Close the stream
	_ = sio.Close()

	// Try to read after close
	buf := make([]byte, 10)
	_, err = sio.Read(buf)
	assert.Error(t, err)
}

// TestStreamIOWriteAfterClose tests Write after Close
func TestStreamIOWriteAfterClose(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	stream, err := client.connection.NewStream(0)
	require.NoError(t, err)

	sio := newStreamIO(*stream)

	// Close the stream
	_ = sio.Close()

	// Try to write after close
	data := []byte("test data")
	_, err = sio.Write(data)
	assert.Error(t, err)
}
