package fuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	log.Printf("[FUSE] NewTransparentFS: creating root FS with backingPath=%s, guardPoint.SecureStoragePath=%s", 
		guardPoint.SecureStoragePath, guardPoint.SecureStoragePath)
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
var _ = (fs.NodeFsyncer)((*TransparentFS)(nil))

func (tfs *TransparentFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	virtualPath := filepath.Join(tfs.getVirtualPath(), name)
	backingPath := filepath.Join(tfs.backingPath, name)

	log.Printf("[FUSE] Lookup: name=%s, currentVirtualPath=%s, newVirtualPath=%s, currentBackingPath=%s, newBackingPath=%s", 
		name, tfs.getVirtualPath(), virtualPath, tfs.backingPath, backingPath)

	info, err := os.Stat(backingPath)
	if err != nil {
		log.Printf("[FUSE] Lookup: stat failed for %s: %v", backingPath, err)
		return nil, syscall.ENOENT
	}

	var child fs.InodeEmbedder
	if info.IsDir() {
		child = &TransparentFS{
			interceptor: tfs.interceptor,
			guardPoint:  tfs.guardPoint,
			backingPath: backingPath,
		}
		log.Printf("[FUSE] Lookup: created directory child for %s -> %s", virtualPath, backingPath)
	} else {
		child = &TransparentFile{
			interceptor: tfs.interceptor,
			guardPoint:  tfs.guardPoint,
			virtualPath: virtualPath,
			backingPath: backingPath,
		}
		log.Printf("[FUSE] Lookup: created file child for %s -> %s", virtualPath, backingPath)
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
	log.Printf("[FUSE] Create: ========== FILE CREATE START ==========")
	log.Printf("[FUSE] Create: name=%s, currentBackingPath=%s", name, tfs.backingPath)
	log.Printf("[FUSE] Create: guardPoint - protected=%s, secure=%s", tfs.guardPoint.ProtectedPath, tfs.guardPoint.SecureStoragePath)
	
	virtualPath := filepath.Join(tfs.getVirtualPath(), name)
	backingPath := filepath.Join(tfs.backingPath, name)

	log.Printf("[FUSE] Create: COMPUTED PATHS - virtual=%s, backing=%s", virtualPath, backingPath)

	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Create: user context - uid=%d, gid=%d, pid=%d, binary=%s, flags=%d, mode=%o", uid, gid, pid, binary, flags, mode)

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

	log.Printf("[FUSE] Create successful: virtual=%s, backing=%s", virtualPath, backingPath)
	log.Printf("[FUSE] Create: ========== FILE CREATE END ==========")
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
	log.Printf("[FUSE] ========== READDIR OPERATION START ==========")
	
	virtualPath := tfs.getVirtualPath()
	log.Printf("[FUSE] Readdir: Generated virtual path=%s from backing path=%s", virtualPath, tfs.backingPath)

	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Readdir: User context - uid=%d, gid=%d, pid=%d, binary=%s", uid, gid, pid, binary)
	log.Printf("[FUSE] Readdir: Guard point - protected=%s, secure=%s", tfs.guardPoint.ProtectedPath, tfs.guardPoint.SecureStoragePath)

	op := &filesystem.FileOperation{
		Type:   "browse", // Changed from "list" to "browse" to match policy
		Path:   virtualPath,
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	log.Printf("[FUSE] Readdir: Calling interceptor with operation: Type=%s, Path=%s, UID=%d, Binary=%s", op.Type, op.Path, op.UID, op.Binary)
	
	result, err := tfs.interceptor.InterceptList(ctx, op)
	log.Printf("[FUSE] Readdir: Interceptor response - allowed=%v, err=%v", result.Allowed, err)
	if err != nil || !result.Allowed {
		log.Printf("[FUSE] Readdir: ACCESS DENIED - returning EACCES")
		log.Printf("[FUSE] ========== READDIR OPERATION END (DENIED) ==========")
		return nil, syscall.EACCES
	}

	log.Printf("[FUSE] Readdir: ACCESS GRANTED - reading directory %s", tfs.backingPath)
	entries, err := os.ReadDir(tfs.backingPath)
	if err != nil {
		log.Printf("[FUSE] Readdir: Failed to read directory %s: %v", tfs.backingPath, err)
		log.Printf("[FUSE] ========== READDIR OPERATION END (ERROR) ==========")
		return nil, syscall.EIO
	}

	var dirEntries []fuse.DirEntry
	log.Printf("[FUSE] Readdir: Found %d entries in directory", len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("[FUSE] Readdir: Skipping entry %s due to info error: %v", entry.Name(), err)
			continue
		}

		dirEntry := fuse.DirEntry{
			Name: entry.Name(),
			Ino:  1,
		}

		if info.IsDir() {
			dirEntry.Mode = fuse.S_IFDIR
			log.Printf("[FUSE] Readdir: Added directory entry: %s", entry.Name())
		} else {
			dirEntry.Mode = fuse.S_IFREG
			log.Printf("[FUSE] Readdir: Added file entry: %s", entry.Name())
		}

		dirEntries = append(dirEntries, dirEntry)
	}

	log.Printf("[FUSE] Readdir: Returning %d entries to FUSE", len(dirEntries))
	log.Printf("[FUSE] ========== READDIR OPERATION END (SUCCESS) ==========")
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

func (tfs *TransparentFS) Fsync(ctx context.Context, fh fs.FileHandle, flags uint32) syscall.Errno {
	// For directories, we open the directory and call fsync on the file descriptor
	// This ensures directory metadata (like new file entries) are synced to disk
	dir, err := os.Open(tfs.backingPath)
	if err != nil {
		log.Printf("[FUSE] Directory Fsync failed to open: %v", err)
		return syscall.EIO
	}
	defer dir.Close()

	if err := dir.Sync(); err != nil {
		log.Printf("[FUSE] Directory Fsync failed: %v", err)
		return syscall.EIO
	}

	log.Printf("[FUSE] Directory Fsync successful: %s", tfs.backingPath)
	return 0
}

func (tfs *TransparentFS) getVirtualPath() string {
	// Clean paths to handle any path inconsistencies
	guardSecurePath := filepath.Clean(tfs.guardPoint.SecureStoragePath)
	backingPath := filepath.Clean(tfs.backingPath)
	guardProtectedPath := filepath.Clean(tfs.guardPoint.ProtectedPath)
	
	log.Printf("[FUSE] getVirtualPath: input - guardSecure=%s, backing=%s, guardProtected=%s", 
		guardSecurePath, backingPath, guardProtectedPath)
	
	// Handle root guard point directory case
	if backingPath == guardSecurePath {
		log.Printf("[FUSE] getVirtualPath: root directory case, returning %s", guardProtectedPath)
		return guardProtectedPath
	}
	
	// Check if backingPath is actually under the guard point
	if !strings.HasPrefix(backingPath, guardSecurePath+string(filepath.Separator)) && backingPath != guardSecurePath {
		log.Printf("[FUSE] getVirtualPath: WARNING - backingPath %s is not under guardSecurePath %s", backingPath, guardSecurePath)
		return guardProtectedPath
	}
	
	rel, err := filepath.Rel(guardSecurePath, backingPath)
	if err != nil {
		log.Printf("[FUSE] getVirtualPath error: %v, guardSecure=%s, backing=%s", err, guardSecurePath, backingPath)
		return guardProtectedPath
	}
	
	// Handle "." case (same directory)
	if rel == "." {
		log.Printf("[FUSE] getVirtualPath: same directory case, returning %s", guardProtectedPath)
		return guardProtectedPath
	}
	
	virtualPath := filepath.Join(guardProtectedPath, rel)
	virtualPath = filepath.Clean(virtualPath)
	
	log.Printf("[FUSE] getVirtualPath: FINAL - guardSecure=%s, backing=%s, rel=%s, virtual=%s", 
		guardSecurePath, backingPath, rel, virtualPath)
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

