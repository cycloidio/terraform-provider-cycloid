package provider

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	// clusterRegistryHost is the IN-NETWORK registry reference (service name on
	// the compose network, internal port always 5000). It is what the
	// plugin-registry/plugin-manager containers use to PULL the image. It is
	// independent of the published HOST port, so it MUST stay hardcoded — the
	// repository name is what matches across host-side push and in-network pull
	// on the same registry instance.
	clusterRegistryHost      = "docker-registry:5000"
	clusterPluginRegistryURL = "http://plugin-registry:4000"
	clusterPluginManagerURL  = "http://plugin-manager:4000"
	clusterTestPluginManager = "test-plugin-manager"
	pluginImageName          = "plugin-hello-world"
	pluginImageTag           = "1.0.0"
	// pluginImageCluster is the in-network pull reference (unchanged).
	pluginImageCluster = clusterRegistryHost + "/" + pluginImageName + ":" + pluginImageTag
	pluginImageSource  = "docker.io/cycloid/" + pluginImageName
)

// localRegistryHost is the HOST-SIDE registry reference used by the test process
// (docker login / tag / push, reachability + manifest checks) from the host. In
// CI each parallel run publishes the registry on a distinct host port, so this
// reads TFACC_REGISTRY_HOST (default "localhost:5000" for local dev). It does
// NOT affect the in-network pull path — the manager still pulls via
// clusterRegistryHost (docker-registry:5000) against the same registry instance.
var localRegistryHost = func() string {
	if h := os.Getenv("TFACC_REGISTRY_HOST"); h != "" {
		return h
	}
	return "localhost:5000"
}()

// pluginImageLocal is the host-side push tag, derived from the env-driven host.
var pluginImageLocal = localRegistryHost + "/" + pluginImageName + ":" + pluginImageTag

// ensurePluginHelloWorld pushes plugin-hello-world:1.0.0 to the local docker-registry
// and returns the in-cluster image reference (docker-registry:5000/...) that the
// plugin-registry API expects when publishing a version.
//
// Skips the test if docker is not available or localhost:5000 is unreachable.
// Idempotent: skips the push if the manifest already exists.
func ensurePluginHelloWorld(t *testing.T) string {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("skipping plugin image test: docker binary not found")
	}

	if !isRegistryReachable() {
		t.Skip("skipping plugin image test: localhost:5000 unreachable (run just be-start first)")
	}

	if !manifestExists() {
		pushImage(t)
	}

	return pluginImageCluster
}

func isRegistryReachable() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + localRegistryHost + "/v2/")
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return true
}

func manifestExists() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://%s/v2/%s/manifests/%s", localRegistryHost, pluginImageName, pluginImageTag)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.SetBasicAuth("cycloid", "cycloid123")
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func pushImage(t *testing.T) {
	t.Helper()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("docker command failed %v: %s", args, string(out))
		}
	}

	run("docker", "login", localRegistryHost, "-u", "cycloid", "-p", "cycloid123")
	run("docker", "pull", pluginImageSource)
	run("docker", "tag", pluginImageSource, pluginImageLocal)
	run("docker", "push", pluginImageLocal)
}
