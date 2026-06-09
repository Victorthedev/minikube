/*
Copyright 2019 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package auxdriver

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/blang/semver/v4"

	"k8s.io/klog/v2"
)

func newAuxUnthealthyError(path string) error {
	return fmt.Errorf(`failed to execute auxiliary version command "%s --version"`, path)
}

func newAuxNotFoundError(name, path string) error {
	return fmt.Errorf("auxiliary driver %s not found in path %s", name, path)
}

// ErrAuxDriverVersionCommandFailed indicates the aux driver 'version' command failed to run
var ErrAuxDriverVersionCommandFailed error

// ErrAuxDriverVersionNotinPath was not found in PATH
var ErrAuxDriverVersionNotinPath error

// InstallOrUpdate downloads driver if it is not present, or updates it if there's a newer version
func InstallOrUpdate(_ string, _ string, _ bool, _ bool) error {
	return nil
}

// validateDriver validates if a driver appears to be up-to-date and installed properly
func validateDriver(executable string, v semver.Version) (string, error) {
	klog.Infof("Validating %s, PATH=%s", executable, os.Getenv("PATH"))
	path, err := exec.LookPath(executable)
	if err != nil {
		klog.Warningf("driver not in path : %s, %v", path, err.Error())
		ErrAuxDriverVersionNotinPath = newAuxNotFoundError(executable, path)
		return path, ErrAuxDriverVersionNotinPath
	}

	cmd := exec.Command(path, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		klog.Warningf("%s failed: %v: %s", cmd, err, output)
		ErrAuxDriverVersionCommandFailed = newAuxUnthealthyError(path)
		return path, ErrAuxDriverVersionCommandFailed
	}

	ev := extractDriverVersion(string(output))
	if len(ev) == 0 {
		return path, fmt.Errorf("%s: unable to extract version from %q", executable, output)
	}

	driverVersion, err := semver.Make(ev)
	if err != nil {
		return path, fmt.Errorf("can't parse driver version: %w", err)
	}
	klog.Infof("%s version is %s", path, driverVersion)

	if driverVersion.LT(v) {
		return path, fmt.Errorf("%s is version %s, want %s", executable, driverVersion, v)
	}
	return path, nil
}

// extractDriverVersion extracts the driver version.
// KVM drivers support the 'version' command, that display the information as:
// version: vX.X.X
// commit: XXXX
// This method returns the version 'vX.X.X' or empty if the version isn't found.
func extractDriverVersion(s string) string {
	versionRegex := regexp.MustCompile(`version:(.*)`)
	matches := versionRegex.FindStringSubmatch(s)

	if len(matches) != 2 {
		return ""
	}

	v := strings.TrimSpace(matches[1])
	return strings.TrimPrefix(v, "v")
}

func driverExists(driverName string) bool {
	_, err := exec.LookPath(driverName)
	return err == nil
}
