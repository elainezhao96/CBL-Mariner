// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/file"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safechroot"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/safemount.go"
	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/shell"
	"golang.org/x/sys/unix"
)

const (
	configDirMountPathInChroot = "/_imageconfigs"
	resolveConfPath            = "/etc/resolv.conf"
)

func doCustomizations(buildDir string, baseConfigPath string, config *imagecustomizerapi.SystemConfig,
	imageChroot *safechroot.Chroot, rpmsSources []string, useBaseImageRpmRepos bool,
) error {
	var err error

	// Note: The ordering of the customization steps here should try to mirror the order of the equivalent steps in imager
	// tool as closely as possible.

	err = overrideResolvConf(imageChroot)
	if err != nil {
		return err
	}

	err = addRemoveAndUpdatePackages(buildDir, baseConfigPath, config, imageChroot, rpmsSources, useBaseImageRpmRepos)
	if err != nil {
		return err
	}

	err = updateHostname(config.Hostname, imageChroot)
	if err != nil {
		return err
	}

	err = copyAdditionalFiles(baseConfigPath, config.AdditionalFiles, imageChroot)
	if err != nil {
		return err
	}

	err = runScripts(baseConfigPath, config.PostInstallScripts, imageChroot)
	if err != nil {
		return err
	}

	err = runScripts(baseConfigPath, config.FinalizeImageScripts, imageChroot)
	if err != nil {
		return err
	}

	err = deleteResolvConf(imageChroot)
	if err != nil {
		return err
	}

	return nil
}

// Override the resolv.conf file, so that in-chroot processes can access the network.
// For example, to install packages from packages.microsoft.com.
func overrideResolvConf(imageChroot *safechroot.Chroot) error {
	logger.Log.Debugf("Overriding resolv.conf file")

	imageResolveConfPath := filepath.Join(imageChroot.RootDir(), resolveConfPath)

	// Remove the existing resolv.conf file, if it exists.
	// Note: It is assumed that the image will have a process that runs on boot that will override the resolv.conf
	// file. For example, systemd-resolved. So, it isn't neccessary to make a back-up of the existing file.
	err := os.RemoveAll(imageResolveConfPath)
	if err != nil {
		return fmt.Errorf("failed to delete existing resolv.conf file: %w", err)
	}

	err = file.Copy(resolveConfPath, imageResolveConfPath)
	if err != nil {
		return fmt.Errorf("failed to override resolv.conf file with host's resolv.conf: %w", err)
	}

	return nil
}

// Delete the overridden resolv.conf file.
// Note: It is assumed that the image will have a process that runs on boot that will override the resolv.conf
// file. For example, systemd-resolved.
func deleteResolvConf(imageChroot *safechroot.Chroot) error {
	logger.Log.Debugf("Deleting overridden resolv.conf file")

	imageResolveConfPath := filepath.Join(imageChroot.RootDir(), resolveConfPath)

	err := os.RemoveAll(imageResolveConfPath)
	if err != nil {
		return fmt.Errorf("failed to delete overridden resolv.conf file: %w", err)
	}

	return err
}

func updateHostname(hostname string, imageChroot *safechroot.Chroot) error {
	var err error

	if hostname == "" {
		return nil
	}

	hostnameFilePath := filepath.Join(imageChroot.RootDir(), "etc/hostname")
	err = file.Write(hostname, hostnameFilePath)
	if err != nil {
		return fmt.Errorf("failed to write hostname file: %w", err)
	}

	return nil
}

func copyAdditionalFiles(baseConfigPath string, additionalFiles map[string]imagecustomizerapi.FileConfigList, imageChroot *safechroot.Chroot) error {
	var err error

	for sourceFile, fileConfigs := range additionalFiles {
		for _, fileConfig := range fileConfigs {
			fileToCopy := safechroot.FileToCopy{
				Src:         filepath.Join(baseConfigPath, sourceFile),
				Dest:        fileConfig.Path,
				Permissions: (*fs.FileMode)(fileConfig.Permissions),
			}

			err = imageChroot.AddFiles(fileToCopy)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func runScripts(baseConfigPath string, scripts []imagecustomizerapi.Script, imageChroot *safechroot.Chroot) error {
	configDirMountPath := filepath.Join(imageChroot.RootDir(), configDirMountPathInChroot)

	// Bind mount the config directory so that the scripts can access any required resources.
	mount, err := safemount.NewMount(baseConfigPath, configDirMountPath, "", unix.MS_BIND|unix.MS_RDONLY, "", true)
	if err != nil {
		return err
	}
	defer mount.Close()

	for _, script := range scripts {
		scriptPathInChroot := filepath.Join(configDirMountPathInChroot, script.Path)
		command := fmt.Sprintf("%s %s", scriptPathInChroot, script.Args)

		// Run the script.
		err = imageChroot.UnsafeRun(func() error {
			err := shell.ExecuteLive(false, shell.ShellProgram, "-c", command)
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	err = mount.Close()
	if err != nil {
		return err
	}

	return nil
}
