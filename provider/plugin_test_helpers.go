package provider

import (
	"fmt"
	"net/http"
	"os/exec"
	"testing"
	"time"
)

const (
	localRegistryHost        = "localhost:5000"
	clusterRegistryHost      = "docker-registry:5000"
	clusterPluginRegistryURL = "http://plugin-registry:4000"
	clusterPluginManagerURL  = "http://plugin-manager:4000"
	clusterTestPluginManager = "test-plugin-manager"
	pluginImageName        = "plugin-hello-world"
	pluginImageTag         = "1.0.0"
	pluginImageLocal       = localRegistryHost + "/" + pluginImageName + ":" + pluginImageTag
	pluginImageCluster     = clusterRegistryHost + "/" + pluginImageName + ":" + pluginImageTag
	pluginImageSource      = "docker.io/cycloid/" + pluginImageName
)

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
	resp.Body.Close()
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
	resp.Body.Close()
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
