// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"testing"
	"time"

	uitest "github.com/cppforlife/go-cli-ui/ui/test"
	"github.com/stretchr/testify/require"
)

func TestPackageRepository(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kappCtrl := Kctrl{t, env.Namespace, env.KctrlBinaryPath, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	pkgrName := "test-package-repository"
	pkgrURL := `ghcr.io/carvel-dev/kc-e2e-test-repo:latest`

	newRepoNamespace := "carvel-test-repo-a"

	kind := "PackageRepository"

	cleanUp := func() {
		RemoveClusterResource(t, kind, pkgrName, env.Namespace, kubectl)
		RemoveClusterResource(t, kind, pkgrName, newRepoNamespace, kubectl)
	}

	cleanUp()
	defer cleanUp()

	logger.Section("listing repositories with no repository present", func() {
		out, _ := kappCtrl.RunWithOpts([]string{"package", "repository", "list", "--json"}, RunOpts{})

		output := uitest.JSONUIFromBytes(t, []byte(out))

		expectedOutputRows := []map[string]string{}
		require.Exactly(t, expectedOutputRows, output.Tables[0].Rows)
	})

	logger.Section("getting a nonexistent repository", func() {
		_, err := kappCtrl.RunWithOpts([]string{"package", "repository", "get", "-r", pkgrName, "--json"}, RunOpts{
			AllowError: true,
		})
		require.Error(t, err)
	})

	logger.Section("deletion of nonexistent repository", func() {
		_, err := kappCtrl.RunWithOpts([]string{"package", "repository", "delete", "-r", pkgrName}, RunOpts{
			AllowError: true,
		})
		require.Error(t, err)
	})

	logger.Section("adding a repository", func() {
		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL})

		kubectl.Run([]string{"get", kind, pkgrName})
		kubectl.Run([]string{"get", "pkgm/pkg.test.carvel.dev"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.1.0.0"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.2.0.0"})
	})

	logger.Section("adding of existing repository", func() {
		start := time.Now()
		out := kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL})
		elapsed := time.Since(start).Seconds()
		require.Equal(t, elapsed < 5, true, "Adding of existing package repository takes more than 5 seconds")
		require.Contains(t, out, "Fetch succeeded")
		require.Contains(t, out, "Template succeeded")
		require.Contains(t, out, "Deploy succeeded")
		require.Contains(t, out, "Succeeded")
	})

	logger.Section("adding of existing repository with new url", func() {

		_, err := kappCtrl.RunWithOpts([]string{"package", "repository", "add", "-r", pkgrName, "--url", "https://carvel.dev"}, RunOpts{
			AllowError: true,
		})
		require.Error(t, err)

		kubectl.Run([]string{"get", kind, pkgrName})

		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL})

		out := kappCtrl.Run([]string{"package", "repository", "get", "-r", pkgrName, "--json"})

		output := uitest.JSONUIFromBytes(t, []byte(out))

		expectedOutputRows := []map[string]string{{
			"conditions":           "- type: ReconcileSucceeded\n  status: \"True\"\n  reason: \"\"\n  message: \"\"",
			"status":               "Reconcile succeeded",
			"namespace":            env.Namespace,
			"name":                 pkgrName,
			"source":               fmt.Sprintf("(imgpkg) %s", pkgrURL),
			"useful_error_message": "",
		}}
		require.Exactly(t, expectedOutputRows, output.Tables[0].Rows)
	})

	logger.Section("kicking a repository", func() {
		out := kappCtrl.Run([]string{"package", "repository", "kick", "-r", pkgrName})

		require.Contains(t, out, "Deploy succeeded")
	})

	logger.Section("listing repositories with one repository being present", func() {
		out, _ := kappCtrl.RunWithOpts([]string{"package", "repository", "list", "--json"}, RunOpts{})

		output := uitest.JSONUIFromBytes(t, []byte(out))

		expectedOutputRows := []map[string]string{{
			"name":   pkgrName,
			"source": fmt.Sprintf("(imgpkg) %s", pkgrURL),
			"status": "Reconcile succeeded",
		}}
		require.Exactly(t, expectedOutputRows, output.Tables[0].Rows)
	})

	logger.Section("getting a repository", func() {
		out, _ := kappCtrl.RunWithOpts([]string{"package", "repository", "get", "-r", pkgrName, "--json"}, RunOpts{})

		output := uitest.JSONUIFromBytes(t, []byte(out))

		expectedOutputRows := []map[string]string{{
			"conditions":           "- type: ReconcileSucceeded\n  status: \"True\"\n  reason: \"\"\n  message: \"\"",
			"status":               "Reconcile succeeded",
			"namespace":            env.Namespace,
			"name":                 pkgrName,
			"source":               fmt.Sprintf("(imgpkg) %s", pkgrURL),
			"useful_error_message": "",
		}}
		require.Exactly(t, expectedOutputRows, output.Tables[0].Rows)
	})

	logger.Section("deletion of an existing repository", func() {
		kappCtrl.RunWithOpts([]string{"package", "repository", "delete", "-r", pkgrName}, RunOpts{})

		_, err := kubectl.RunWithOpts([]string{"get", "packagerepository", pkgrName}, RunOpts{AllowError: true})
		require.Contains(t, err.Error(), "not found")
	})

	logger.Section("deletion of a failed repository", func() {
		repoName := "invalidrepo"
		repoBundle := "invalid.bundle.com/invalid-account/test:1.0.0"
		_, err := kappCtrl.RunWithOpts([]string{"package", "repository", "add", "-r", repoName, "--url", repoBundle}, RunOpts{AllowError: true})
		require.Contains(t, err.Error(), "Fetch failed")

		out := kappCtrl.Run([]string{"package", "repository", "delete", "-r", repoName})
		require.Contains(t, out, "Succeeded")

		_, err = kubectl.RunWithOpts([]string{"get", "packagerepository", repoName}, RunOpts{AllowError: true})
		require.Contains(t, err.Error(), "not found")
	})

	logger.Section("updating a repository", func() {
		_, err := kappCtrl.RunWithOpts([]string{"package", "repository", "add", "-r", pkgrName, "--url", "https://carvel.dev"}, RunOpts{
			AllowError: true})
		require.Error(t, err)

		kubectl.Run([]string{"get", kind, pkgrName})

		kappCtrl.Run([]string{"package", "repository", "update", "-r", pkgrName, "--url", pkgrURL})

		out := kappCtrl.Run([]string{"package", "repository", "get", "-r", pkgrName, "--json"})

		output := uitest.JSONUIFromBytes(t, []byte(out))

		expectedOutputRows := []map[string]string{{
			"conditions":           "- type: ReconcileSucceeded\n  status: \"True\"\n  reason: \"\"\n  message: \"\"",
			"status":               "Reconcile succeeded",
			"namespace":            env.Namespace,
			"name":                 pkgrName,
			"source":               fmt.Sprintf("(imgpkg) %s", pkgrURL),
			"useful_error_message": "",
		}}
		require.Exactly(t, expectedOutputRows, output.Tables[0].Rows)
	})

	logger.Section("updating a repository with no change in url", func() {
		start := time.Now()
		out := kappCtrl.Run([]string{"package", "repository", "update", "-r", pkgrName, "--url", pkgrURL})
		elapsed := time.Since(start).Seconds()
		require.Equal(t, elapsed < 5, true, "Adding of existing package repository takes more than 5 seconds")
		require.Contains(t, out, "Fetch succeeded")
		require.Contains(t, out, "Template succeeded")
		require.Contains(t, out, "Deploy succeeded")
		require.Contains(t, out, "Succeeded")

	})

	logger.Section("creating a repository in a new namespace that doesn't exist", func() {
		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL, "-n", newRepoNamespace, "--create-namespace"})

		kubectl.Run([]string{"get", kind, pkgrName, "-n", newRepoNamespace})
	})

	logger.Section("creating a repository in a namespace that already exists", func() {
		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL, "-n", env.Namespace, "--create-namespace"})

		kubectl.Run([]string{"get", kind, pkgrName, "-n", env.Namespace})
	})

}

func TestPackageRepositoryTagSemver(t *testing.T) {
	env := BuildEnv(t)
	logger := Logger{}
	kappCtrl := Kctrl{t, env.Namespace, env.KctrlBinaryPath, logger}
	kubectl := Kubectl{t, env.Namespace, logger}

	pkgrName := "test-package-repository"
	pkgrURL := `ghcr.io/carvel-dev/kc-e2e-test-repo`

	kind := "PackageRepository"

	cleanUp := func() {
		RemoveClusterResource(t, kind, pkgrName, env.Namespace, kubectl)
	}

	cleanUp()
	defer cleanUp()

	logger.Section("adding a repository", func() {
		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL + ":v1.0.0"})

		out := kubectl.Run([]string{"get", kind, pkgrName, "-oyaml"})
		require.Contains(t, out, "tag: v1.0.0")
		kubectl.Run([]string{"get", "pkgm/pkg.test.carvel.dev"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.1.0.0"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.2.0.0"})
	})

	logger.Section("deleting a repository", func() {
		kappCtrl.Run([]string{"package", "repository", "delete", "-r", pkgrName})
	})

	logger.Section("adding a repository", func() {
		kappCtrl.Run([]string{"package", "repository", "add", "-r", pkgrName, "--url", pkgrURL + ":v2.0.0", "--semver-tag-constraints", "1.0.0"})
		out := kubectl.Run([]string{"get", kind, pkgrName, "-oyaml"})
		require.Contains(t, out, "tag: v2.0.0")
	})

	logger.Section("updating a repository", func() {
		kappCtrl.Run([]string{"package", "repository", "update", "-r", pkgrName, "--url", pkgrURL, "--semver-tag-constraints", ">1.0.0"})

		out := kubectl.Run([]string{"get", kind, pkgrName, "-oyaml"})
		require.Contains(t, out, "tag: v3.0.0")
		kubectl.Run([]string{"get", "pkgm/pkg.test.carvel.dev"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.1.0.0"})
		kubectl.Run([]string{"get", "pkg/pkg.test.carvel.dev.2.0.0"})
	})

}
