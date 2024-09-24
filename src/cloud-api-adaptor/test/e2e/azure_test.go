//go:build azure

// (C) Copyright Confidential Containers Contributors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"os"
	"strings"
	"testing"

	_ "github.com/confidential-containers/cloud-api-adaptor/src/cloud-api-adaptor/test/provisioner/azure"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func TestDeletePodAzure(t *testing.T) {
	t.Parallel()
	DoTestDeleteSimplePod(t, testEnv, assert)
}

func TestCreateSimplePodAzure(t *testing.T) {
	t.Parallel()
	DoTestCreateSimplePod(t, testEnv, assert)
}

func TestCreatePodWithConfigMapAzure(t *testing.T) {
	t.Parallel()
	DoTestCreatePodWithConfigMap(t, testEnv, assert)
}

func TestCreatePodWithSecretAzure(t *testing.T) {
	t.Parallel()
	DoTestCreatePodWithSecret(t, testEnv, assert)
}

func TestCreateNginxDeploymentAzure(t *testing.T) {
	t.Parallel()
	DoTestNginxDeployment(t, testEnv, assert)
}

func TestPodToServiceCommunicationAzure(t *testing.T) {
	t.Parallel()
	DoTestPodToServiceCommunication(t, testEnv, assert)
}

func TestPodsMTLSCommunicationAzure(t *testing.T) {
	t.Parallel()
	DoTestPodsMTLSCommunication(t, testEnv, assert)
}

func TestPodVMwithAnnotationsInstanceTypeAzure(t *testing.T) {
	SkipTestOnCI(t)
	t.Parallel()
	instanceSize := "Standard_DC2as_v5"
	DoTestPodVMwithAnnotationsInstanceType(t, testEnv, assert, instanceSize)
}

func TestPodVMwithAnnotationsInvalidInstanceTypeAzure(t *testing.T) {
	t.Parallel()
	// Using an instance type that's not configured in the AZURE_INSTANCE_SIZE
	instanceSize := "Standard_D8as_v5"
	DoTestPodVMwithAnnotationsInvalidInstanceType(t, testEnv, assert, instanceSize)
}

// Test with device annotation
func TestPodWithCrioDeviceAnnotationAzure(t *testing.T) {
	if !isTestOnCrio() {
		t.Skip("Skipping test as it is not running on CRI-O")
	}
	t.Parallel()
	DoTestPodWithCrioDeviceAnnotation(t, testEnv, assert)
}

// Negative test with device annotation
func TestPodWithIncorrectDeviceAnnotationAzure(t *testing.T) {
	if !isTestOnCrio() {
		t.Skip("Skipping test as it is not running on CRI-O")
	}
	t.Parallel()
	DoTestPodWithIncorrectCrioDeviceAnnotation(t, testEnv, assert)
}

// Test with init container
func TestPodWithInitContainerAzure(t *testing.T) {
	t.Parallel()
	DoTestPodWithInitContainer(t, testEnv, assert)
}

// Test to check the presence if pod can access files from internet
// Use DoTestPodWithSpecificCommands and provide the commands to be executed in the pod
func TestPodToDownloadExternalFileAzure(t *testing.T) {
	t.Parallel()
	// Create TestCommand struct with the command to download index.html
	command1 := TestCommand{
		Command:             []string{"wget", "-q", "www.google.com"},
		TestCommandStdoutFn: IsBufferEmpty,
		TestCommandStderrFn: IsBufferEmpty,
	}

	// Check index.html is downloaded
	command2 := TestCommand{
		Command: []string{"ls", "index.html"},
		TestCommandStdoutFn: func(stdout bytes.Buffer) bool {
			if strings.Contains(stdout.String(), "index.html") {
				t.Logf("index.html is present in the pod")
				return true
			} else {
				t.Logf("index.html is not present in the pod")
				return false
			}
		},
		TestCommandStderrFn: IsBufferEmpty,
	}

	commands := []TestCommand{command1, command2}

	DoTestPodWithSpecificCommands(t, testEnv, assert, commands)
}

// Method to check external IP access using ping
func TestCreatePeerPodContainerWithExternalIPAccessAzure(t *testing.T) {
	SkipTestOnCI(t)
	t.Parallel()
	DoTestCreatePeerPodContainerWithExternalIPAccess(t, testEnv, assert)
}

func TestKbsKeyRelease(t *testing.T) {
	if !isTestWithKbs() {
		t.Skip("Skipping kbs related test as kbs is not deployed")
	}
	t.Parallel()
	kbsEndpoint, _ := keyBrokerService.GetCachedKbsEndpoint()
	testSecret := envconf.RandomName("coco-pp-e2e-secret", 25)
	resourcePath := "caa/workload_key/test_key.bin"
	err := keyBrokerService.SetSecret(resourcePath, []byte(testSecret))
	DoTestKbsKeyRelease(t, testEnv, assert, kbsEndpoint, resourcePath, testSecret)
}

func TestRemoteAttestation(t *testing.T) {
	t.Parallel()
	var kbsEndpoint string
	if ep := os.Getenv("KBS_ENDPOINT"); ep != "" {
		kbsEndpoint = ep
	} else if keyBrokerService == nil {
		t.Skip("Skipping because KBS config is missing")
	} else {
		kbsEndpoint, _ = keyBrokerService.GetCachedKbsEndpoint()
	}
	DoTestRemoteAttestation(t, testEnv, assert, kbsEndpoint)
}

func TestTrusteeOperatorKeyReleaseForSpecificKey(t *testing.T) {
	if !isTestWithTrusteeOperator() {
		t.Skip("Skipping kbs related test as Trustee Operator is not deployed")
	}
	t.Parallel()
	kbsEndpoint, _ := keyBrokerService.GetCachedKbsEndpoint()
	DoTestTrusteeOperatorKeyReleaseForSpecificKey(t, testEnv, assert, kbsEndpoint)
}

func TestAzureImageDecryption(t *testing.T) {
	if !isTestWithKbs() {
		t.Skip("Skipping kbs related test as kbs is not deployed")
	}
	t.Parallel()

	DoTestImageDecryption(t, testEnv, assert, keyBrokerService)
}
