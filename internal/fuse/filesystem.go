package fuse

import (
	"context"
	"fmt"
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
var _ = (fs.NodeRenamer)((*TransparentFS)(nil))

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

	attr := fileInfoToAttr(info)
	// Force correct ownership display in FUSE
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		log.Printf("[FUSE] Lookup: backing store ownership - uid=%d, gid=%d", stat.Uid, stat.Gid)
		attr.Uid = stat.Uid
		attr.Gid = stat.Gid
	} else {
		log.Printf("[FUSE] Lookup: no syscall.Stat_t available, using fallback")
		// Fallback for when syscall.Stat_t is not available
		attr.Uid = 1000 // ntoi user
		attr.Gid = 1000 // ntoi group
	}
	log.Printf("[FUSE] Lookup: setting FUSE attr - uid=%d, gid=%d for %s", attr.Uid, attr.Gid, name)
	
	// Disable attribute caching to ensure fresh reads
	out.AttrValid = 0
	out.EntryValid = 0
	out.Attr = attr
	return tfs.Inode.NewInode(ctx, child, stable), 0
}

func (tfs *TransparentFS) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	virtualPath := filepath.Join(tfs.getVirtualPath(), name)
	backingPath := filepath.Join(tfs.backingPath, name)

	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Create: path=%s, flags=%d, mode=%o, uid=%d, gid=%d, pid=%d, binary=%s", virtualPath, flags, mode, uid, gid, pid, binary)

	op := &filesystem.FileOperation{
		Type:   "write", // Changed from "create" to "write" to match policy actions
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
		log.Printf("[FUSE] Create denied: %v", err)
		return nil, nil, 0, syscall.EACCES
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(backingPath), 0755); err != nil {
		log.Printf("[FUSE] Create mkdir failed: %v", err)
		return nil, nil, 0, syscall.EIO
	}

	// Create file with proper flags - ensure O_CREATE is set
	fileFlags := int(flags)
	if fileFlags&os.O_CREATE == 0 {
		fileFlags |= os.O_CREATE
	}
	
	file, err := os.OpenFile(backingPath, fileFlags, os.FileMode(mode))
	if err != nil {
		log.Printf("[FUSE] Create file failed: %v", err)
		return nil, nil, 0, syscall.EIO
	}

	// Set file ownership to the requesting user on both the file handle and backing store
	if err := file.Chown(uid, gid); err != nil {
		log.Printf("[FUSE] Warning: Could not set file ownership on handle: %v", err)
	}
	
	// Also set ownership on the backing store file directly
	if err := os.Chown(backingPath, uid, gid); err != nil {
		log.Printf("[FUSE] Warning: Could not set backing store ownership: %v", err)
	}

	child := &TransparentFile{
		interceptor: tfs.interceptor,
		guardPoint:  tfs.guardPoint,
		virtualPath: virtualPath,
		backingPath: backingPath,
	}

	stable := fs.StableAttr{
		Mode: fuse.S_IFREG,
		Ino:  getInoForPath(virtualPath),
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		log.Printf("[FUSE] Create stat failed: %v", err)
		return nil, nil, 0, syscall.EIO
	}

	attr := fileInfoToAttr(info)
	// Force correct ownership for FUSE presentation
	attr.Uid = uint32(uid)
	attr.Gid = uint32(gid)
	out.Attr = attr
	out.SetAttrTimeout(0)
	out.SetEntryTimeout(0)

	fileHandle := &TransparentFileHandle{
		file:        file,
		interceptor: tfs.interceptor,
		guardPoint:  tfs.guardPoint,
		virtualPath: virtualPath,
		backingPath: backingPath,
	}

	log.Printf("[FUSE] Create successful: %s", virtualPath)
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

func (tfs *TransparentFS) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	oldVirtualPath := filepath.Join(tfs.getVirtualPath(), name)
	oldBackingPath := filepath.Join(tfs.backingPath, name)

	newFS := newParent.(*TransparentFS)
	newVirtualPath := filepath.Join(newFS.getVirtualPath(), newName)
	newBackingPath := filepath.Join(newFS.backingPath, newName)

	log.Printf("[FUSE] Rename: from=%s to=%s, uid=%d, pid=%d, binary=%s", oldVirtualPath, newVirtualPath, uid, pid, binary)

	// Check permissions for both source and destination
	writeOp := &filesystem.FileOperation{
		Type:   "write",
		Path:   newVirtualPath,
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := tfs.interceptor.InterceptWrite(ctx, writeOp)
	if err != nil || !result.Allowed {
		log.Printf("[FUSE] Rename denied: %v", err)
		return syscall.EACCES
	}

	// Perform the rename operation
	if err := os.Rename(oldBackingPath, newBackingPath); err != nil {
		log.Printf("[FUSE] Rename failed: %v", err)
		return syscall.EIO
	}

	log.Printf("[FUSE] Rename successful")
	return 0
}

func (tfs *TransparentFS) getVirtualPath() string {
	rel, err := filepath.Rel(tfs.guardPoint.SecureStoragePath, tfs.backingPath)
	if err != nil {
		log.Printf("[FUSE] getVirtualPath error: %v, guardPoint=%s, backingPath=%s", err, tfs.guardPoint.SecureStoragePath, tfs.backingPath)
		return tfs.guardPoint.ProtectedPath
	}
	virtualPath := filepath.Join(tfs.guardPoint.ProtectedPath, rel)
	log.Printf("[FUSE] getVirtualPath: guardPoint=%s, backingPath=%s, rel=%s, virtualPath=%s", 
		tfs.guardPoint.SecureStoragePath, tfs.backingPath, rel, virtualPath)
	return virtualPath
}

func getProcessBinary(pid int) string {
	exePath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "exe")
	binary, err := os.Readlink(exePath)
	if err != nil {
		return "unknown"
	}
	return binary
}

func getInoForPath(path string) uint64 {
	// Simple hash function for inode generation
	var hash uint64 = 5381
	for _, c := range []byte(path) {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}

