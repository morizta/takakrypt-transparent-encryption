package fuse

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/filesystem"
)

type TransparentFS struct {
	fs.Inode
	interceptor  *filesystem.Interceptor
	guardPoint   *config.GuardPoint
	backingPath  string
}

func NewTransparentFS(interceptor *filesystem.Interceptor, guardPoint *config.GuardPoint) *TransparentFS {
	return &TransparentFS{
		interceptor: interceptor,
		guardPoint:  guardPoint,
		backingPath: guardPoint.SecureStoragePath,
	}
}

var _ = (fs.NodeLookuper)((*TransparentFS)(nil))
var _ = (fs.NodeCreater)((*TransparentFS)(nil))
var _ = (fs.NodeMkdirer)((*TransparentFS)(nil))
var _ = (fs.NodeRmdirer)((*TransparentFS)(nil))
var _ = (fs.NodeUnlinker)((*TransparentFS)(nil))
var _ = (fs.NodeOpener)((*TransparentFS)(nil))
var _ = (fs.NodeReaddirer)((*TransparentFS)(nil))
var _ = (fs.NodeGetattrer)((*TransparentFS)(nil))
var _ = (fs.NodeSetattrer)((*TransparentFS)(nil))

func (tfs *TransparentFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	virtualPath := filepath.Join(tfs.getVirtualPath(), name)
	backingPath := filepath.Join(tfs.backingPath, name)

	info, err := os.Stat(backingPath)
	if err != nil {
		return nil, syscall.ENOENT
	}

	var child fs.InodeEmbedder
	if info.IsDir() {
		child = &TransparentFS{
			interceptor: tfs.interceptor,
			guardPoint:  tfs.guardPoint,
			backingPath: backingPath,
		}
	} else {
		child = &TransparentFile{
			interceptor: tfs.interceptor,
			guardPoint:  tfs.guardPoint,
			virtualPath: virtualPath,
			backingPath: backingPath,
		}
	}

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  1,
	}
	if info.IsDir() {
		stable.Mode = fuse.S_IFDIR
	}

	out.Attr = fileInfoToAttr(info)
	return tfs.Inode.NewInode(ctx, child, stable), 0
}

func (tfs *TransparentFS) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	virtualPath := filepath.Join(tfs.getVirtualPath(), name)
	backingPath := filepath.Join(tfs.backingPath, name)

	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	op := &filesystem.FileOperation{
		Type:   "create",
		Path:   virtualPath,
		Mode:   os.FileMode(mode),
		Flags:  int(flags),
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := tfs.interceptor.InterceptWrite(ctx, op)
	if err != nil || !result.Allowed {
		return nil, nil, 0, syscall.EACCES
	}

	if err := os.MkdirAll(filepath.Dir(backingPath), 0755); err != nil {
		return nil, nil, 0, syscall.EIO
	}

	file, err := os.OpenFile(backingPath, int(flags), os.FileMode(mode))
	if err != nil {
		return nil, nil, 0, syscall.EIO
	}

	child := &TransparentFile{
		interceptor: tfs.interceptor,
		guardPoint:  tfs.guardPoint,
		virtualPath: virtualPath,
		backingPath: backingPath,
	}

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  1,
	}

	info, _ := file.Stat()
	out.Attr = fileInfoToAttr(info)

	fileHandle := &TransparentFileHandle{
		file:        file,
		interceptor: tfs.interceptor,
		guardPoint:  tfs.guardPoint,
		virtualPath: virtualPath,
		backingPath: backingPath,
	}

	return tfs.NewInode(ctx, child, stable), fileHandle, 0, 0
}

func (tfs *TransparentFS) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	backingPath := filepath.Join(tfs.backingPath, name)

	if err := os.MkdirAll(backingPath, os.FileMode(mode)); err != nil {
		return nil, syscall.EIO
	}

	child := &TransparentFS{
		interceptor: tfs.interceptor,
		guardPoint:  tfs.guardPoint,
		backingPath: backingPath,
	}

	stable := fs.StableAttr{
		Mode: fuse.S_IFDIR,
		Ino:  1,
	}

	info, _ := os.Stat(backingPath)
	out.Attr = fileInfoToAttr(info)

	return tfs.Inode.NewInode(ctx, child, stable), 0
}

func (tfs *TransparentFS) Rmdir(ctx context.Context, name string) syscall.Errno {
	backingPath := filepath.Join(tfs.backingPath, name)
	if err := os.Remove(backingPath); err != nil {
		return syscall.EIO
	}
	return 0
}

func (tfs *TransparentFS) Unlink(ctx context.Context, name string) syscall.Errno {
	backingPath := filepath.Join(tfs.backingPath, name)
	if err := os.Remove(backingPath); err != nil {
		return syscall.EIO
	}
	return 0
}

func (tfs *TransparentFS) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, 0, syscall.EISDIR
}

func (tfs *TransparentFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	virtualPath := tfs.getVirtualPath()

	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Readdir: path=%s, uid=%d, gid=%d, pid=%d, binary=%s", virtualPath, uid, gid, pid, binary)

	op := &filesystem.FileOperation{
		Type:   "browse", // Changed from "list" to "browse" to match policy
		Path:   virtualPath,
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := tfs.interceptor.InterceptList(ctx, op)
	log.Printf("[FUSE] Readdir result: allowed=%v, err=%v", result.Allowed, err)
	if err != nil || !result.Allowed {
		return nil, syscall.EACCES
	}

	entries, err := os.ReadDir(tfs.backingPath)
	if err != nil {
		return nil, syscall.EIO
	}

	var dirEntries []fuse.DirEntry
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		dirEntry := fuse.DirEntry{
			Name: entry.Name(),
			Ino:  1,
		}

		if info.IsDir() {
			dirEntry.Mode = fuse.S_IFDIR
		} else {
			dirEntry.Mode = fuse.S_IFREG
		}

		dirEntries = append(dirEntries, dirEntry)
	}

	return fs.NewListDirStream(dirEntries), 0
}

func (tfs *TransparentFS) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	info, err := os.Stat(tfs.backingPath)
	if err != nil {
		return syscall.ENOENT
	}

	out.Attr = fileInfoToAttr(info)
	return 0
}

func (tfs *TransparentFS) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if in.Valid&fuse.FATTR_MODE != 0 {
		if err := os.Chmod(tfs.backingPath, os.FileMode(in.Mode)); err != nil {
			return syscall.EIO
		}
	}

	if in.Valid&fuse.FATTR_SIZE != 0 {
		if err := os.Truncate(tfs.backingPath, int64(in.Size)); err != nil {
			return syscall.EIO
		}
	}

	info, err := os.Stat(tfs.backingPath)
	if err != nil {
		return syscall.EIO
	}

	out.Attr = fileInfoToAttr(info)
	return 0
}

func (tfs *TransparentFS) getVirtualPath() string {
	rel, err := filepath.Rel(tfs.guardPoint.SecureStoragePath, tfs.backingPath)
	if err != nil {
		return tfs.guardPoint.ProtectedPath
	}
	return filepath.Join(tfs.guardPoint.ProtectedPath, rel)
}

func getProcessBinary(pid int) string {
	exePath := filepath.Join("/proc", string(rune(pid)), "exe")
	binary, err := os.Readlink(exePath)
	if err != nil {
		return "unknown"
	}
	return binary
}

