// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagerParseCmd(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	// Verify flags are registered
	assert.NotNil(t, flags.Lookup("uri"))
	assert.NotNil(t, flags.Lookup("pool-name"))
	assert.NotNil(t, flags.Lookup("network-name"))
	assert.NotNil(t, flags.Lookup("vol-name"))
	assert.NotNil(t, flags.Lookup("launch-security"))
	assert.NotNil(t, flags.Lookup("firmware"))
	assert.NotNil(t, flags.Lookup("cpu"))
	assert.NotNil(t, flags.Lookup("memory"))
	assert.NotNil(t, flags.Lookup("data-dir"))
	assert.NotNil(t, flags.Lookup("disable-cvm"))
}

func TestManagerParseCmdWithValues(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	// Set some values
	err := flags.Set("uri", "qemu:///system")
	require.NoError(t, err)
	err = flags.Set("pool-name", "test-pool")
	require.NoError(t, err)
	err = flags.Set("network-name", "test-network")
	require.NoError(t, err)
	err = flags.Set("vol-name", "test-vol.qcow2")
	require.NoError(t, err)
	err = flags.Set("cpu", "4")
	require.NoError(t, err)
	err = flags.Set("memory", "4096")
	require.NoError(t, err)

	// Verify values are set
	assert.Equal(t, "qemu:///system", libvirtcfg.URI)
	assert.Equal(t, "test-pool", libvirtcfg.PoolName)
	assert.Equal(t, "test-network", libvirtcfg.NetworkName)
	assert.Equal(t, "test-vol.qcow2", libvirtcfg.VolName)
	assert.Equal(t, uint(4), libvirtcfg.CPU)
	assert.Equal(t, uint(4096), libvirtcfg.Memory)
}

func TestManagerLoadEnv(t *testing.T) {
	manager := &Manager{}
	// LoadEnv should do nothing (it's a no-op)
	manager.LoadEnv()
	// No assertion needed, just verify it doesn't panic
}

func TestManagerGetConfig(t *testing.T) {
	manager := &Manager{}
	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, &libvirtcfg, config)
}

func TestManagerNewProvider(t *testing.T) {
	checkConfig(t)

	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	manager.ParseCmd(flags)

	// Set required config
	err := flags.Set("uri", testCfg.URI)
	require.NoError(t, err)
	err = flags.Set("pool-name", testCfg.PoolName)
	require.NoError(t, err)
	err = flags.Set("network-name", testCfg.NetworkName)
	require.NoError(t, err)
	err = flags.Set("vol-name", testCfg.VolName)
	require.NoError(t, err)

	provider, err := manager.NewProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestManagerNewProviderInvalidConfig(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	manager.ParseCmd(flags)

	// Set invalid URI
	err := flags.Set("uri", "invalid://uri")
	require.NoError(t, err)

	_, err = manager.NewProvider()
	assert.Error(t, err)
}

func TestDefaultConstants(t *testing.T) {
	assert.Equal(t, "qemu+ssh://root@192.168.122.1/system?no_verify=1", defaultURI)
	assert.Equal(t, "default", defaultPoolName)
	assert.Equal(t, "default", defaultNetworkName)
	assert.Equal(t, "/var/lib/libvirt/images", defaultDataDir)
	assert.Equal(t, "podvm-base.qcow2", defaultVolName)
	assert.Equal(t, "", defaultLaunchSecurity)
	assert.Equal(t, "/usr/share/OVMF/OVMF_CODE_4M.fd", defaultFirmware)
	assert.Equal(t, "2", defaultCPU)
	assert.Equal(t, "8192", defaultMemory)
}

func TestManagerParseCmdLaunchSecurity(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	err := flags.Set("launch-security", "s390-pv")
	require.NoError(t, err)

	assert.Equal(t, "s390-pv", libvirtcfg.LaunchSecurity)
}

func TestManagerParseCmdDisableCVM(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	err := flags.Set("disable-cvm", "false")
	require.NoError(t, err)

	assert.False(t, libvirtcfg.DisableCVM)
}

func TestManagerParseCmdFirmware(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	customFirmware := "/custom/path/to/firmware.fd"
	err := flags.Set("firmware", customFirmware)
	require.NoError(t, err)

	assert.Equal(t, customFirmware, libvirtcfg.Firmware)
}

func TestManagerParseCmdDataDir(t *testing.T) {
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	customDataDir := "/custom/data/dir"
	err := flags.Set("data-dir", customDataDir)
	require.NoError(t, err)

	assert.Equal(t, customDataDir, libvirtcfg.DataDir)
}
