package fuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/filesystem"
)

type MountManager struct {
	interceptor *filesystem.Interceptor
	mounts      map[string]*Mount
}

type Mount struct {
	GuardPoint *config.GuardPoint
	Server     *fuse.Server
	MountPoint string
}

func NewMountManager(interceptor *filesystem.Interceptor) *MountManager {
	return &MountManager{
		interceptor: interceptor,
		mounts:      make(map[string]*Mount),
	}
}

func (mm *MountManager) MountGuardPoints(ctx context.Context, guardPoints []config.GuardPoint) error {
	for _, gp := range guardPoints {
		if !gp.Enabled {
			continue
		}

		if err := mm.MountGuardPoint(ctx, &gp); err != nil {
			log.Printf("Failed to mount guard point %s: %v", gp.Code, err)
			continue
		}

		log.Printf("Mounted guard point: %s -> %s", gp.ProtectedPath, gp.SecureStoragePath)
	}

	return nil
}

func (mm *MountManager) MountGuardPoint(ctx context.Context, gp *config.GuardPoint) error {
	if err := os.MkdirAll(gp.ProtectedPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	if err := os.MkdirAll(gp.SecureStoragePath, 0755); err != nil {
		return fmt.Errorf("failed to create backing storage: %w", err)
	}

	root := NewTransparentFS(mm.interceptor, gp)

	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			AllowOther: true,
			Debug:      false,
			Name:       "takakrypt-te",
			FsName:     "takakrypt-transparent-encryption",
		},
	}

	server, err := fs.Mount(gp.ProtectedPath, root, opts)
	if err != nil {
		return fmt.Errorf("failed to mount FUSE filesystem: %w", err)
	}

	mount := &Mount{
		GuardPoint: gp,
		Server:     server,
		MountPoint: gp.ProtectedPath,
	}

	mm.mounts[gp.Code] = mount

	go server.Wait()

	return nil
}

func (mm *MountManager) UnmountAll() error {
	for code := range mm.mounts {
		if err := mm.UnmountGuardPoint(code); err != nil {
			log.Printf("Failed to unmount guard point %s: %v", code, err)
		}
	}
	return nil
}

func (mm *MountManager) UnmountGuardPoint(code string) error {
	mount, exists := mm.mounts[code]
	if !exists {
		return fmt.Errorf("guard point %s not mounted", code)
	}

	if err := mount.Server.Unmount(); err != nil {
		return fmt.Errorf("failed to unmount %s: %w", mount.MountPoint, err)
	}

	delete(mm.mounts, code)
	log.Printf("Unmounted guard point: %s", mount.MountPoint)

	return nil
}

func (mm *MountManager) GetMountInfo() map[string]*Mount {
	return mm.mounts
}

func (mm *MountManager) IsMounted(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, mount := range mm.mounts {
		mountPath, err := filepath.Abs(mount.MountPoint)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(mountPath, absPath)
		if err != nil {
			continue
		}

		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../")) {
			return true
		}
	}

	return false
}