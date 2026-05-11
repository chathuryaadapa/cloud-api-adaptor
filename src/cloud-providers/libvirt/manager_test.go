// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetLibvirtConfig() {
	libvirtcfg = Config{
		URI:            defaultURI,
		PoolName:       defaultPoolName,
		NetworkName:    defaultNetworkName,
		DataDir:        defaultDataDir,
		VolName:        defaultVolName,
		LaunchSecurity: defaultLaunchSecurity,
		Firmware:       defaultFirmware,
		CPU:            2,
		Memory:         8192,
		DisableCVM:     true,
	}
}

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

func TestManagerParseCmdWithEnvVars(t *testing.T) {
	resetLibvirtConfig()

	// Set environment variables
	t.Setenv("LIBVIRT_URI", "qemu+ssh://testhost/system")
	t.Setenv("LIBVIRT_POOL", "env-pool")
	t.Setenv("LIBVIRT_NET", "env-network")
	t.Setenv("LIBVIRT_VOL_NAME", "env-vol.qcow2")
	t.Setenv("LIBVIRT_LAUNCH_SECURITY", "sev")
	t.Setenv("LIBVIRT_EFI_FIRMWARE", "/env/firmware.fd")
	t.Setenv("LIBVIRT_CPU", "8")
	t.Setenv("LIBVIRT_MEMORY", "16384")
	t.Setenv("DISABLECVM", "false")

	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	// Verify environment variables were applied
	assert.Equal(t, "qemu+ssh://testhost/system", libvirtcfg.URI)
	assert.Equal(t, "env-pool", libvirtcfg.PoolName)
	assert.Equal(t, "env-network", libvirtcfg.NetworkName)
	assert.Equal(t, "env-vol.qcow2", libvirtcfg.VolName)
	assert.Equal(t, "sev", libvirtcfg.LaunchSecurity)
	assert.Equal(t, "/env/firmware.fd", libvirtcfg.Firmware)
	assert.Equal(t, uint(8), libvirtcfg.CPU)
	assert.Equal(t, uint(16384), libvirtcfg.Memory)
	assert.False(t, libvirtcfg.DisableCVM)
}

func TestManagerParseCmdFlagOverridesEnv(t *testing.T) {
	resetLibvirtConfig()

	// Set environment variable
	t.Setenv("LIBVIRT_URI", "qemu+ssh://envhost/system")
	t.Setenv("LIBVIRT_CPU", "8")

	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	// Verify env var was applied initially
	assert.Equal(t, "qemu+ssh://envhost/system", libvirtcfg.URI)
	assert.Equal(t, uint(8), libvirtcfg.CPU)

	// Override with flag
	err := flags.Set("uri", "qemu:///system")
	require.NoError(t, err)
	err = flags.Set("cpu", "4")
	require.NoError(t, err)

	// Verify flag overrides env var
	assert.Equal(t, "qemu:///system", libvirtcfg.URI)
	assert.Equal(t, uint(4), libvirtcfg.CPU)
}

func TestManagerParseCmdInvalidValues(t *testing.T) {
	tests := []struct {
		name         string
		flagName     string
		invalidValue string
		description  string
	}{
		{
			name:         "invalid CPU value",
			flagName:     "cpu",
			invalidValue: "invalid",
			description:  "non-numeric CPU value should be rejected",
		},
		{
			name:         "invalid memory value",
			flagName:     "memory",
			invalidValue: "not-a-number",
			description:  "non-numeric memory value should be rejected",
		},
		{
			name:         "invalid bool value",
			flagName:     "disable-cvm",
			invalidValue: "maybe",
			description:  "invalid boolean value should be rejected",
		},
		{
			name:         "negative CPU value",
			flagName:     "cpu",
			invalidValue: "-1",
			description:  "negative CPU value should be rejected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetLibvirtConfig()
			manager := &Manager{}
			flags := flag.NewFlagSet("test", flag.ContinueOnError)

			manager.ParseCmd(flags)

			err := flags.Set(tc.flagName, tc.invalidValue)
			assert.Error(t, err, tc.description)
		})
	}
}

func TestManagerParseCmdEmptyStringValues(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	manager.ParseCmd(flags)

	// Set empty string values (should be allowed)
	err := flags.Set("uri", "")
	require.NoError(t, err)
	err = flags.Set("pool-name", "")
	require.NoError(t, err)
	err = flags.Set("launch-security", "")
	require.NoError(t, err)

	// Verify empty strings are set
	assert.Equal(t, "", libvirtcfg.URI)
	assert.Equal(t, "", libvirtcfg.PoolName)
	assert.Equal(t, "", libvirtcfg.LaunchSecurity)
}

func TestManagerLoadEnv(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	// LoadEnv should do nothing (it's a no-op)
	manager.LoadEnv()
	// No assertion needed, just verify it doesn't panic
}

func TestManagerGetConfig(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, &libvirtcfg, config)
}

func TestManagerNewProvider(t *testing.T) {
	resetLibvirtConfig()
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
	resetLibvirtConfig()
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
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{
			name:     "defaultURI",
			actual:   defaultURI,
			expected: "qemu+ssh://root@192.168.122.1/system?no_verify=1",
		},
		{
			name:     "defaultPoolName",
			actual:   defaultPoolName,
			expected: "default",
		},
		{
			name:     "defaultNetworkName",
			actual:   defaultNetworkName,
			expected: "default",
		},
		{
			name:     "defaultDataDir",
			actual:   defaultDataDir,
			expected: "/var/lib/libvirt/images",
		},
		{
			name:     "defaultVolName",
			actual:   defaultVolName,
			expected: "podvm-base.qcow2",
		},
		{
			name:     "defaultLaunchSecurity",
			actual:   defaultLaunchSecurity,
			expected: "",
		},
		{
			name:     "defaultFirmware",
			actual:   defaultFirmware,
			expected: "/usr/share/OVMF/OVMF_CODE_4M.fd",
		},
		{
			name:     "defaultCPU",
			actual:   defaultCPU,
			expected: "2",
		},
		{
			name:     "defaultMemory",
			actual:   defaultMemory,
			expected: "8192",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.actual)
		})
	}
}

func TestManagerParseCmdFlags(t *testing.T) {
	tests := []struct {
		name          string
		flagName      string
		flagValue     string
		expectedValue interface{}
		getActual     func() interface{}
	}{
		{
			name:          "uri flag",
			flagName:      "uri",
			flagValue:     "qemu:///system",
			expectedValue: "qemu:///system",
			getActual:     func() interface{} { return libvirtcfg.URI },
		},
		{
			name:          "pool-name flag",
			flagName:      "pool-name",
			flagValue:     "test-pool",
			expectedValue: "test-pool",
			getActual:     func() interface{} { return libvirtcfg.PoolName },
		},
		{
			name:          "network-name flag",
			flagName:      "network-name",
			flagValue:     "test-network",
			expectedValue: "test-network",
			getActual:     func() interface{} { return libvirtcfg.NetworkName },
		},
		{
			name:          "vol-name flag",
			flagName:      "vol-name",
			flagValue:     "test-vol.qcow2",
			expectedValue: "test-vol.qcow2",
			getActual:     func() interface{} { return libvirtcfg.VolName },
		},
		{
			name:          "launch-security flag",
			flagName:      "launch-security",
			flagValue:     "s390-pv",
			expectedValue: "s390-pv",
			getActual:     func() interface{} { return libvirtcfg.LaunchSecurity },
		},
		{
			name:          "firmware flag",
			flagName:      "firmware",
			flagValue:     "/custom/path/to/firmware.fd",
			expectedValue: "/custom/path/to/firmware.fd",
			getActual:     func() interface{} { return libvirtcfg.Firmware },
		},
		{
			name:          "data-dir flag",
			flagName:      "data-dir",
			flagValue:     "/custom/data/dir",
			expectedValue: "/custom/data/dir",
			getActual:     func() interface{} { return libvirtcfg.DataDir },
		},
		{
			name:          "cpu flag",
			flagName:      "cpu",
			flagValue:     "4",
			expectedValue: uint(4),
			getActual:     func() interface{} { return libvirtcfg.CPU },
		},
		{
			name:          "memory flag",
			flagName:      "memory",
			flagValue:     "4096",
			expectedValue: uint(4096),
			getActual:     func() interface{} { return libvirtcfg.Memory },
		},
		{
			name:          "disable-cvm flag",
			flagName:      "disable-cvm",
			flagValue:     "false",
			expectedValue: false,
			getActual:     func() interface{} { return libvirtcfg.DisableCVM },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetLibvirtConfig()
			manager := &Manager{}
			flags := flag.NewFlagSet("test", flag.ContinueOnError)

			manager.ParseCmd(flags)

			err := flags.Set(tc.flagName, tc.flagValue)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedValue, tc.getActual())
		})
	}
}

func TestManagerParseCmdInvalidCPU(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	manager.ParseCmd(flags)

	err := flags.Set("cpu", "not-a-number")
	assert.Error(t, err, "should reject non-numeric CPU value")
}

func TestManagerParseCmdNegativeMemory(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	manager.ParseCmd(flags)

	// Note: flag package may accept this, but it's semantically invalid
	err := flags.Set("memory", "-1")
	// This might not error at parse time, but should fail at validation
	if err == nil {
		// If parsing succeeds, NewProvider should catch it
		_, err = manager.NewProvider()
		assert.Error(t, err, "should fail with negative memory")
	}
}

func TestManagerNewProviderMissingRequiredConfig(t *testing.T) {
	resetLibvirtConfig()
	manager := &Manager{}
	flags := flag.NewFlagSet("test", flag.ContinueOnError)
	manager.ParseCmd(flags)

	// Set empty required fields
	err := flags.Set("uri", "")
	require.NoError(t, err)

	_, err = manager.NewProvider()
	assert.Error(t, err, "should fail with empty URI")
}
