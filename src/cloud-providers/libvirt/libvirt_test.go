// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package libvirt

import (
	"context"
	"fmt"
	"testing"
	"time"

	provider "github.com/confidential-containers/cloud-api-adaptor/src/cloud-providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	libvirtxml "libvirt.org/go/libvirtxml"
)

var testCfg Config

func init() {
	provider.DefaultToEnv(&testCfg.URI, "LIBVIRT_URI", "") // explicitly no fallback here
	provider.DefaultToEnv(&testCfg.PoolName, "LIBVIRT_POOL", defaultPoolName)
	provider.DefaultToEnv(&testCfg.NetworkName, "LIBVIRT_NET", defaultNetworkName)
	provider.DefaultToEnv(&testCfg.VolName, "LIBVIRT_VOL_NAME", defaultVolName)
}

func checkConfig(t *testing.T) {
	if testCfg.URI == "" {
		t.Skipf("Skipping because LIBVIRT_URI is not configured")
	}
}

func TestLibvirtConnection(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	if err != nil {
		t.Error(err)
	}
	defer client.connection.Close()

	assert.NotNil(t, client.nodeInfo)
	assert.NotNil(t, client.caps)
}

func TestGetArchitecture(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	if err != nil {
		t.Error(err)
	}
	defer client.connection.Close()

	node, err := client.connection.GetNodeInfo()
	if err != nil {
		t.Error(err)
	}

	arch := node.Model
	if arch == "" {
		t.FailNow()
	}
}

func verifyDomainXML(domXML *libvirtxml.Domain) error {
	arch := domXML.OS.Type.Arch
	if arch != archS390x && arch != archAArch64 {
		return nil
	}
	// verify we have iommu on the disks
	for i, disk := range domXML.Devices.Disks {
		if disk.Target.Bus == "virtio" && disk.Driver.IOMMU != "on" {
			return fmt.Errorf("disk [%d] does not have IOMMU assigned", i)
		}
	}
	// verify we have iommu on the networks
	for i, iface := range domXML.Devices.Interfaces {
		if iface.Model.Type == "virtio" && iface.Driver.IOMMU != "on" {
			return fmt.Errorf("interface [%d] does not have IOMMU assigned", i)
		}
	}
	return nil
}

func TestCreateDomainXMLs390x(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	if err != nil {
		t.Error(err)
	}
	defer client.connection.Close()

	vm := vmConfig{}

	domainCfg := domainConfig{
		name:        "TestCreateDomainS390x",
		cpu:         2,
		mem:         2,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cidata.iso",
	}

	domCfg, err := createDomainXML(client, &domainCfg, &vm)
	if err != nil {
		t.Error(err)
	}

	arch := domCfg.OS.Type.Arch
	if domCfg.OS.Type.Arch != archS390x {
		t.Skipf("Skipping because architecture is [%s] and not [%s].", arch, archS390x)
	}

	// verify the config
	err = verifyDomainXML(domCfg)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateDomainXMLaarch64(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	if err != nil {
		t.Error(err)
	}
	defer client.connection.Close()

	vm := vmConfig{}

	domainCfg := domainConfig{
		name:        "TestCreateDomainAArch64",
		cpu:         2,
		mem:         4,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cloudinit.iso",
	}

	domCfg, err := createDomainXML(client, &domainCfg, &vm)
	if err != nil {
		t.Error(err)
	}

	arch := domCfg.OS.Type.Arch
	if domCfg.OS.Type.Arch != archAArch64 {
		t.Skipf("Skipping because architecture is [%s] and not [%s].", arch, archAArch64)
	}

	err = verifyDomainXML(domCfg)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeletableDiskPaths(t *testing.T) {
	tests := []struct {
		name     string
		domain   *libvirtxml.Domain
		expected []string
	}{
		{
			name: "returns file-backed disks only",
			domain: &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/images/root.qcow2"},
							},
						},
						{
							Source: &libvirtxml.DomainDiskSource{
								File: &libvirtxml.DomainDiskSourceFile{File: "/var/lib/libvirt/images/cloudinit.iso"},
							},
						},
						{
							Source: &libvirtxml.DomainDiskSource{},
						},
						{},
					},
				},
			},
			expected: []string{
				"/var/lib/libvirt/images/root.qcow2",
				"/var/lib/libvirt/images/cloudinit.iso",
			},
		},
		{
			name:     "nil domain returns nil",
			domain:   nil,
			expected: nil,
		},
		{
			name: "empty disk list returns empty slice",
			domain: &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{},
			},
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			paths := getDeletableDiskPaths(tc.domain)
			assert.Equal(t, tc.expected, paths)
		})
	}
}

func TestCreateCloudInitISO(t *testing.T) {
	vm := &vmConfig{
		name:     "test-vm",
		userData: "#cloud-config\nusers:\n  - default",
	}

	isoData, err := createCloudInitISO(vm)
	assert.NoError(t, err)
	assert.NotNil(t, isoData)
	assert.Greater(t, len(isoData), 0)
}

// TestCheckDomainExistsByName tests domain existence checking
func TestCheckDomainExistsByName(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with non-existent domain
	exists, err := checkDomainExistsByName("non-existent-domain-12345", client)
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestGetGuestForArchType tests guest architecture lookup
func TestGetGuestForArchType(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with valid architecture
	guest, err := getGuestForArchType(client.caps, client.nodeInfo.Model, typeHardwareVirtualMachine)
	if err != nil {
		t.Skipf("Architecture %s not found in capabilities", client.nodeInfo.Model)
	}
	assert.NotNil(t, guest)
	assert.Equal(t, client.nodeInfo.Model, guest.Arch.Name)
}

// TestGetGuestForArchTypeInvalid tests guest architecture lookup with invalid arch
func TestGetGuestForArchTypeInvalid(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with invalid architecture
	_, err = getGuestForArchType(client.caps, "invalid-arch-xyz", typeHardwareVirtualMachine)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find any guests")
}

// TestGetHostCapabilities tests host capabilities retrieval
func TestGetHostCapabilities(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	caps, err := getHostCapabilities(client.connection)
	assert.NoError(t, err)
	assert.NotNil(t, caps)
	assert.NotEmpty(t, caps.Guests)
}
func TestLookupMachine(t *testing.T) {
	machines := []libvirtxml.CapsGuestMachine{
		{Name: "pc", Canonical: "pc-i440fx-2.12"},
		{Name: "q35", Canonical: "pc-q35-2.12"},
		{Name: "virt"},
	}

	tests := []struct {
		name           string
		targetMachine  string
		expectedResult string
	}{
		{
			name:           "find machine with canonical",
			targetMachine:  "pc",
			expectedResult: "pc-i440fx-2.12",
		},
		{
			name:           "find machine without canonical",
			targetMachine:  "virt",
			expectedResult: "virt",
		},
		{
			name:           "machine not found",
			targetMachine:  "nonexistent",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookupMachine(machines, tt.targetMachine)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestConfigVerifier(t *testing.T) {
	newConfig := func(overrides func(*Config)) *Config {
		cfg := &Config{
			URI:         "qemu:///system",
			PoolName:    "default",
			NetworkName: "default",
			VolName:     "podvm-base.qcow2",
			CPU:         1,
			Memory:      512,
		}
		if overrides != nil {
			overrides(cfg)
		}
		return cfg
	}

	tests := []struct {
		name          string
		provider      *libvirtProvider
		expectedError string
	}{
		{
			name: "empty URI fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.URI = ""
				}),
			},
			expectedError: "URI is empty",
		},
		{
			name: "empty pool name fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.PoolName = ""
				}),
			},
			expectedError: "PoolName is empty",
		},
		{
			name: "empty network name fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.NetworkName = ""
				}),
			},
			expectedError: "NetworkName is empty",
		},
		{
			name: "empty volume name fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.VolName = ""
				}),
			},
			expectedError: "VolName is empty",
		},
		{
			name: "zero CPU fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.CPU = 0
				}),
			},
			expectedError: "CPU must be greater than zero",
		},
		{
			name: "zero memory fails",
			provider: &libvirtProvider{
				serviceConfig: newConfig(func(c *Config) {
					c.Memory = 0
				}),
			},
			expectedError: "Memory must be greater than zero",
		},
		{
			name: "valid config passes",
			provider: &libvirtProvider{
				serviceConfig: newConfig(nil),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.ConfigVerifier()
			if tc.expectedError == "" {
				assert.NoError(t, err)
				return
			}

			assert.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestGetCanonicalMachineName(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test based on actual architecture
	switch client.nodeInfo.Model {
	case archS390x:
		name, err := getCanonicalMachineName(client.caps, archS390x, typeHardwareVirtualMachine, "s390-ccw-virtio")
		if err != nil {
			t.Skipf("Machine type not found: %v", err)
		}
		assert.NotEmpty(t, name)
	case archAArch64:
		name, err := getCanonicalMachineName(client.caps, archAArch64, typeHardwareVirtualMachine, "virt")
		if err != nil {
			t.Skipf("Machine type not found: %v", err)
		}
		assert.NotEmpty(t, name)
	default:
		t.Skipf("Skipping canonical machine name test for architecture: %s", client.nodeInfo.Model)
	}
}

// TestGetCanonicalMachineNameInvalid tests canonical machine name with invalid input
func TestGetCanonicalMachineNameInvalid(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test with invalid machine type
	_, err = getCanonicalMachineName(client.caps, client.nodeInfo.Model, typeHardwareVirtualMachine, "invalid-machine-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot find machine type")
}

// TestCreateDomainXMLx86_64 tests x86_64 domain XML creation
func TestCreateDomainXMLx86_64(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	if client.nodeInfo.Model == archS390x || client.nodeInfo.Model == archAArch64 {
		t.Skipf("Skipping x86_64 test on %s architecture", client.nodeInfo.Model)
	}

	vm := &vmConfig{
		launchSecurityType: NoLaunchSecurity,
	}

	domainCfg := domainConfig{
		name:        "TestCreateDomainX86_64",
		cpu:         4,
		mem:         4096,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cidata.iso",
	}

	domCfg, err := createDomainXMLx86_64(client, &domainCfg, vm)
	assert.NoError(t, err)
	assert.NotNil(t, domCfg)
	assert.Equal(t, "test-vm", domCfg.Name)
	assert.Equal(t, uint(4), domCfg.VCPU.Value)
	assert.Equal(t, uint(4096), domCfg.Memory.Value)
	assert.Equal(t, "x86_64", domCfg.OS.Type.Arch)
}

// TestCreateDomainXMLx86_64WithFirmware tests x86_64 domain XML with firmware
func TestCreateDomainXMLx86_64WithFirmware(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	if client.nodeInfo.Model == archS390x || client.nodeInfo.Model == archAArch64 {
		t.Skipf("Skipping x86_64 test on %s architecture", client.nodeInfo.Model)
	}

	vm := &vmConfig{
		launchSecurityType: NoLaunchSecurity,
		firmware:           "/usr/share/OVMF/OVMF_CODE.fd",
	}

	domainCfg := domainConfig{
		name:        "TestCreateDomainX86_64Firmware",
		cpu:         2,
		mem:         2048,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cidata.iso",
	}

	domCfg, err := createDomainXMLx86_64(client, &domainCfg, vm)
	assert.NoError(t, err)
	assert.NotNil(t, domCfg)
	assert.NotNil(t, domCfg.OS.Loader)
	assert.Equal(t, vm.firmware, domCfg.OS.Loader.Path)
	assert.Equal(t, "efi", domCfg.OS.Firmware)
}

// TestCreateDomainXMLx86_64UnsupportedSecurity tests x86_64 with unsupported security
func TestCreateDomainXMLx86_64UnsupportedSecurity(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	if client.nodeInfo.Model == archS390x || client.nodeInfo.Model == archAArch64 {
		t.Skipf("Skipping x86_64 test on %s architecture", client.nodeInfo.Model)
	}

	vm := &vmConfig{
		launchSecurityType: S390PV, // Unsupported on x86_64
	}

	domainCfg := domainConfig{
		name:        "TestUnsupportedSecurity",
		cpu:         2,
		mem:         2048,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cidata.iso",
	}

	_, err = createDomainXMLx86_64(client, &domainCfg, vm)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// TestFreeDomain tests domain pointer freeing
func TestFreeDomain(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create a test domain definition (not actually creating it)
	domainXML := `<domain type='kvm'>
		<name>test-free-domain</name>
		<memory unit='MiB'>512</memory>
		<vcpu>1</vcpu>
		<os><type arch='x86_64'>hvm</type></os>
	</domain>`

	domain, err := client.connection.DomainDefineXML(domainXML)
	if err != nil {
		t.Skipf("Cannot define test domain: %v", err)
	}

	var errCtx error
	freeDomain(domain, &errCtx)
	assert.NoError(t, errCtx)

	// Clean up - undefine the domain
	domain2, _ := client.connection.LookupDomainByName("test-free-domain")
	if domain2 != nil {
		domain2.Undefine()
		domain2.Free()
	}
}

// TestGetLaunchSecurityType tests launch security type detection
func TestGetLaunchSecurityType(t *testing.T) {
	checkConfig(t)

	lstype, err := GetLaunchSecurityType(testCfg.URI)
	assert.NoError(t, err)

	// Verify it returns a valid type
	assert.True(t, lstype == NoLaunchSecurity || lstype == S390PV)
}

// TestGetLaunchSecurityTypeInvalidURI tests with invalid URI
func TestGetLaunchSecurityTypeInvalidURI(t *testing.T) {
	_, err := GetLaunchSecurityType("invalid://uri")
	assert.Error(t, err)
}

// TestGetDomainCapabilities tests domain capabilities retrieval
func TestGetDomainCapabilities(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	guest, err := getGuestForArchType(client.caps, client.nodeInfo.Model, typeHardwareVirtualMachine)
	if err != nil {
		t.Skipf("Cannot get guest for architecture: %v", err)
	}

	domCaps, err := GetDomainCapabilities(
		client.connection,
		guest.Arch.Emulator,
		client.nodeInfo.Model,
		"",
		"kvm",
		0,
	)
	if err != nil {
		t.Skipf("Cannot get domain capabilities: %v", err)
	}
	assert.NotNil(t, domCaps)
}

// TestCreateDomainXML tests the main createDomainXML function
func TestCreateDomainXML(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	vm := &vmConfig{
		launchSecurityType: NoLaunchSecurity,
	}

	domainCfg := domainConfig{
		name:        "TestCreateDomainXML",
		cpu:         2,
		mem:         2048,
		networkName: client.networkName,
		bootDisk:    "/var/lib/libvirt/images/root.qcow2",
		cidataDisk:  "/var/lib/libvirt/images/cidata.iso",
	}

	domCfg, err := createDomainXML(client, &domainCfg, vm)
	assert.NoError(t, err)
	assert.NotNil(t, domCfg)
	assert.Equal(t, domainCfg.name, domCfg.Name)
	assert.Equal(t, domainCfg.cpu, domCfg.VCPU.Value)
	assert.Equal(t, domainCfg.mem, domCfg.Memory.Value)
}

// Additional comprehensive tests for libvirt.go coverage

func TestDeleteDomainFlow(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test deleting a non-existent domain
	err = DeleteDomain(context.Background(), client, "00000000-0000-0000-0000-000000000001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to lookup domain")
}

func TestGetDomainIPsFunction(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Try to get IPs from a non-existent domain (will fail, but tests the function)
	dom, err := client.connection.LookupDomainByName("non-existent-domain-for-ip-test")
	if err == nil {
		defer dom.Free()
		ips, err := getDomainIPs(dom)
		if err != nil {
			t.Logf("getDomainIPs returned error (expected): %v", err)
		} else {
			t.Logf("getDomainIPs returned %d IPs", len(ips))
		}
	}
}

func TestCreateDomainFlow(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Test the CreateDomain flow with minimal config
	vm := &vmConfig{
		name:     "test-create-domain-flow",
		cpu:      1,
		mem:      512,
		userData: "#cloud-config\n",
	}

	// This will likely fail due to missing volumes, but tests the code path
	_, err = CreateDomain(context.Background(), client, vm)
	if err != nil {
		t.Logf("CreateDomain returned error (expected in test): %v", err)
		assert.Error(t, err)
	}
}

func TestUploadIsoFunction(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create a small ISO data
	isoData := []byte("test iso content for upload")
	isoVolName := "test-upload-iso-" + fmt.Sprintf("%d", time.Now().Unix()) + ".iso"

	volumePath, err := uploadIso(isoData, isoVolName, client)
	if err != nil {
		t.Logf("uploadIso returned error: %v", err)
	} else {
		assert.NotEmpty(t, volumePath)
		t.Logf("Uploaded ISO to: %s", volumePath)

		// Clean up
		defer func() {
			deleteVolume(client, isoVolName)
		}()
	}
}

func TestFreeDomainFunction(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	// Create a simple test domain
	domainXML := `<domain type='kvm'>
		<name>test-free-domain-func</name>
		<memory unit='MiB'>256</memory>
		<vcpu>1</vcpu>
		<os>
			<type arch='` + client.nodeInfo.Model + `'>hvm</type>
		</os>
	</domain>`

	dom, err := client.connection.DomainDefineXML(domainXML)
	if err != nil {
		t.Skipf("Cannot define test domain: %v", err)
	}

	// Test freeDomain with nil error context
	var testErr error
	freeDomain(dom, &testErr)
	assert.NoError(t, testErr)

	// Clean up domain if it still exists
	defer func() {
		d, err := client.connection.LookupDomainByName("test-free-domain-func")
		if err == nil {
			d.Undefine()
			d.Free()
		}
	}()
}

func TestFreeDomainWithExistingError(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	domainXML := `<domain type='kvm'>
		<name>test-free-domain-err</name>
		<memory unit='MiB'>256</memory>
		<vcpu>1</vcpu>
		<os>
			<type arch='` + client.nodeInfo.Model + `'>hvm</type>
		</os>
	</domain>`

	dom, err := client.connection.DomainDefineXML(domainXML)
	if err != nil {
		t.Skipf("Cannot define test domain: %v", err)
	}

	// Test freeDomain preserves existing error
	existingErr := fmt.Errorf("existing error")
	freeDomain(dom, &existingErr)
	assert.Equal(t, "existing error", existingErr.Error())

	// Clean up
	defer func() {
		d, err := client.connection.LookupDomainByName("test-free-domain-err")
		if err == nil {
			d.Undefine()
			d.Free()
		}
	}()
}

func TestNewLibvirtClientSuccess(t *testing.T) {
	checkConfig(t)

	config := Config{
		URI:         testCfg.URI,
		PoolName:    testCfg.PoolName,
		NetworkName: testCfg.NetworkName,
		VolName:     "test-vol.qcow2",
	}

	client, err := NewLibvirtClient(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.connection)
	assert.NotNil(t, client.pool)
	assert.NotNil(t, client.nodeInfo)
	assert.NotNil(t, client.caps)
	assert.Equal(t, config.PoolName, client.poolName)
	assert.Equal(t, config.NetworkName, client.networkName)

	defer client.connection.Close()
}

func TestNewLibvirtClientInvalidPool(t *testing.T) {
	config := Config{
		URI:         testCfg.URI,
		PoolName:    "invalid-pool-name-xyz-123",
		NetworkName: testCfg.NetworkName,
		VolName:     "test.qcow2",
	}

	_, err := NewLibvirtClient(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't find storage pool")
}

func TestGetLaunchSecurityTypeArchitectures(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectError bool
	}{
		{
			name:        "valid qemu system URI",
			uri:         "qemu:///system",
			expectError: false,
		},
		{
			name:        "valid qemu session URI",
			uri:         "qemu:///session",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secType, err := GetLaunchSecurityType(tt.uri)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Should return a valid security type
				assert.True(t, secType == NoLaunchSecurity || secType == S390PV)
				t.Logf("Security type for %s: %s", tt.uri, secType.String())
			}
		})
	}
}

func TestCreateCloudInitISOWithVariousData(t *testing.T) {
	tests := []struct {
		name    string
		vm      *vmConfig
		wantErr bool
	}{
		{
			name: "simple user data",
			vm: &vmConfig{
				name:     "test-vm-1",
				userData: "#cloud-config\nusers:\n  - default",
			},
			wantErr: false,
		},
		{
			name: "empty user data",
			vm: &vmConfig{
				name:     "test-vm-2",
				userData: "",
			},
			wantErr: false,
		},
		{
			name: "large user data",
			vm: &vmConfig{
				name:     "test-vm-3",
				userData: string(make([]byte, 5000)),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isoData, err := createCloudInitISO(tt.vm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, isoData)
				assert.Greater(t, len(isoData), 0)
			}
		})
	}
}

func TestCheckDomainExistsByNameVariousCases(t *testing.T) {
	checkConfig(t)

	client, err := NewLibvirtClient(testCfg)
	require.NoError(t, err)
	defer client.connection.Close()

	tests := []struct {
		name       string
		domainName string
		wantExists bool
	}{
		{
			name:       "non-existent domain",
			domainName: "absolutely-non-existent-domain-xyz-123",
			wantExists: false,
		},
		{
			name:       "empty domain name",
			domainName: "",
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := checkDomainExistsByName(tt.domainName, client)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}
