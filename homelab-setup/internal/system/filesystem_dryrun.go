package system

import (
	"fmt"
	"os"

	"github.com/zoro11031/homelab-coreos-minipc/homelab-setup/internal/ui"
)

// DryRunFileSystem wraps FileSystem and logs operations without executing them
type DryRunFileSystem struct {
	fs     *FileSystem
	ui     *ui.UI
	dryRun bool
}

// NewDryRunFileSystem creates a new DryRunFileSystem
func NewDryRunFileSystem(fs *FileSystem, ui *ui.UI, dryRun bool) *DryRunFileSystem {
	return &DryRunFileSystem{
		fs:     fs,
		ui:     ui,
		dryRun: dryRun,
	}
}

// EnsureDirectory wraps FileSystem.EnsureDirectory with dry-run support
func (d *DryRunFileSystem) EnsureDirectory(path string, owner string, perms os.FileMode) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would create directory: %s (owner: %s, perms: %o)", path, owner, perms)
		return nil
	}
	return d.fs.EnsureDirectory(path, owner, perms)
}

// Chown wraps FileSystem.Chown with dry-run support
func (d *DryRunFileSystem) Chown(path string, owner string) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would change ownership of %s to %s", path, owner)
		return nil
	}
	return d.fs.Chown(path, owner)
}

// ChownRecursive wraps FileSystem.ChownRecursive with dry-run support
func (d *DryRunFileSystem) ChownRecursive(path string, owner string) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would recursively change ownership of %s to %s", path, owner)
		return nil
	}
	return d.fs.ChownRecursive(path, owner)
}

// Chmod wraps FileSystem.Chmod with dry-run support
func (d *DryRunFileSystem) Chmod(path string, perms os.FileMode) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would change permissions of %s to %o", path, perms)
		return nil
	}
	return d.fs.Chmod(path, perms)
}

// ChmodRecursive wraps FileSystem.ChmodRecursive with dry-run support
func (d *DryRunFileSystem) ChmodRecursive(path string, perms os.FileMode) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would recursively change permissions of %s to %o", path, perms)
		return nil
	}
	return d.fs.ChmodRecursive(path, perms)
}

// WriteFile wraps FileSystem.WriteFile with dry-run support
func (d *DryRunFileSystem) WriteFile(path string, content []byte, perms os.FileMode) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would write %d bytes to %s (perms: %o)", len(content), path, perms)
		return nil
	}
	return d.fs.WriteFile(path, content, perms)
}

// RemoveDirectory wraps FileSystem.RemoveDirectory with dry-run support
func (d *DryRunFileSystem) RemoveDirectory(path string) error {
	if d.dryRun {
		d.ui.Warningf("[DRY-RUN] Would remove directory: %s", path)
		return nil
	}
	return d.fs.RemoveDirectory(path)
}

// RemoveFile wraps FileSystem.RemoveFile with dry-run support
func (d *DryRunFileSystem) RemoveFile(path string) error {
	if d.dryRun {
		d.ui.Warningf("[DRY-RUN] Would remove file: %s", path)
		return nil
	}
	return d.fs.RemoveFile(path)
}

// CopyFile wraps FileSystem.CopyFile with dry-run support
func (d *DryRunFileSystem) CopyFile(src, dst string) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would copy %s to %s", src, dst)
		return nil
	}
	return d.fs.CopyFile(src, dst)
}

// CreateSymlink wraps FileSystem.CreateSymlink with dry-run support
func (d *DryRunFileSystem) CreateSymlink(target, linkPath string) error {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would create symlink: %s -> %s", linkPath, target)
		return nil
	}
	return d.fs.CreateSymlink(target, linkPath)
}

// Read-only operations pass through directly (no need for dry-run)
func (d *DryRunFileSystem) FileExists(path string) (bool, error) {
	return d.fs.FileExists(path)
}

func (d *DryRunFileSystem) DirectoryExists(path string) (bool, error) {
	return d.fs.DirectoryExists(path)
}

func (d *DryRunFileSystem) GetOwner(path string) (string, error) {
	return d.fs.GetOwner(path)
}

func (d *DryRunFileSystem) GetPermissions(path string) (os.FileMode, error) {
	return d.fs.GetPermissions(path)
}

func (d *DryRunFileSystem) BackupFile(path string) (string, error) {
	if d.dryRun {
		d.ui.Infof("[DRY-RUN] Would create backup of: %s", path)
		return fmt.Sprintf("%s.backup.DRYRUN", path), nil
	}
	return d.fs.BackupFile(path)
}

func (d *DryRunFileSystem) GetDiskUsage(path string) (total, used, free uint64, err error) {
	return d.fs.GetDiskUsage(path)
}

func (d *DryRunFileSystem) GetDiskUsageHuman(path string) (string, error) {
	return d.fs.GetDiskUsageHuman(path)
}

func (d *DryRunFileSystem) CountFiles(path string) (int, error) {
	return d.fs.CountFiles(path)
}

func (d *DryRunFileSystem) ListDirectory(path string) ([]string, error) {
	return d.fs.ListDirectory(path)
}

func (d *DryRunFileSystem) GetFileSize(path string) (int64, error) {
	return d.fs.GetFileSize(path)
}

func (d *DryRunFileSystem) IsMount(path string) (bool, error) {
	return d.fs.IsMount(path)
}
