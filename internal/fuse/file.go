package fuse

import (
	"context"
	"log"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"

	"github.com/takakrypt/transparent-encryption/internal/config"
	"github.com/takakrypt/transparent-encryption/internal/filesystem"
)

type TransparentFile struct {
	fs.Inode
	interceptor *filesystem.Interceptor
	guardPoint  *config.GuardPoint
	virtualPath string
	backingPath string
}

type TransparentFileHandle struct {
	file        *os.File
	interceptor *filesystem.Interceptor
	guardPoint  *config.GuardPoint
	virtualPath string
	backingPath string
}

var _ = (fs.NodeOpener)((*TransparentFile)(nil))
var _ = (fs.NodeGetattrer)((*TransparentFile)(nil))
var _ = (fs.NodeSetattrer)((*TransparentFile)(nil))

var _ = (fs.FileReader)((*TransparentFileHandle)(nil))
var _ = (fs.FileWriter)((*TransparentFileHandle)(nil))
var _ = (fs.FileFlusher)((*TransparentFileHandle)(nil))
var _ = (fs.FileReleaser)((*TransparentFileHandle)(nil))
var _ = (fs.FileFsyncer)((*TransparentFileHandle)(nil))

func (tf *TransparentFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	op := &filesystem.FileOperation{
		Type:   "open",
		Path:   tf.virtualPath,
		Flags:  int(flags),
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := tf.interceptor.InterceptOpen(ctx, op)
	log.Printf("[FUSE] Open result: allowed=%v, err=%v, backingPath=%s", result.Allowed, err, tf.backingPath)
	if err != nil || !result.Allowed {
		log.Printf("[FUSE] Open denied by policy")
		return nil, 0, syscall.EACCES
	}

	log.Printf("[FUSE] Opening backing file: %s", tf.backingPath)
	file, err := os.OpenFile(tf.backingPath, int(flags), 0644)
	if err != nil {
		log.Printf("[FUSE] Failed to open backing file: %v", err)
		return nil, 0, syscall.EIO
	}
	log.Printf("[FUSE] Successfully opened backing file")

	fileHandle := &TransparentFileHandle{
		file:        file,
		interceptor: tf.interceptor,
		guardPoint:  tf.guardPoint,
		virtualPath: tf.virtualPath,
		backingPath: tf.backingPath,
	}

	return fileHandle, 0, 0
}

func (tf *TransparentFile) Getattr(ctx context.Context, fh fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	log.Printf("[FUSE] File Getattr called for: %s (backing: %s)", tf.virtualPath, tf.backingPath)
	
	info, err := os.Stat(tf.backingPath)
	if err != nil {
		log.Printf("[FUSE] File Getattr: os.Stat failed for %s: %v", tf.backingPath, err)
		return syscall.ENOENT
	}

	log.Printf("[FUSE] File Getattr: encrypted file size=%d", info.Size())

	attr := fileInfoToAttr(info)
	
	// Fix file size: encrypted files have 28 bytes overhead (12-byte nonce + 16-byte auth tag)
	// Report the actual decrypted content size to applications
	if info.Size() >= 28 {
		decryptedSize := info.Size() - 28
		attr.Size = uint64(decryptedSize)
		log.Printf("[FUSE] File Getattr: adjusted size from %d to %d (removed 28-byte encryption overhead)", info.Size(), decryptedSize)
	} else {
		log.Printf("[FUSE] File Getattr: file too small for encryption overhead, reporting actual size %d", info.Size())
	}
	// Ensure the FUSE view shows the correct ownership from the backing store
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		log.Printf("[FUSE] File Getattr: backing store ownership - uid=%d, gid=%d", stat.Uid, stat.Gid)
		attr.Uid = stat.Uid
		attr.Gid = stat.Gid
	} else {
		log.Printf("[FUSE] File Getattr: no syscall.Stat_t available, using fallback")
		attr.Uid = 1000 // ntoi user
		attr.Gid = 1000 // ntoi group
	}
	log.Printf("[FUSE] File Getattr: setting FUSE attr - uid=%d, gid=%d for %s", attr.Uid, attr.Gid, tf.virtualPath)
	
	// Force attribute cache timeout to 0 to always re-read
	out.AttrValid = 0
	out.Attr = attr
	return 0
}

func (tf *TransparentFile) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Setattr: path=%s, uid=%d, pid=%d, binary=%s", tf.virtualPath, uid, pid, binary)

	if in.Valid&fuse.FATTR_MODE != 0 {
		log.Printf("[FUSE] Setting mode: %o", in.Mode)
		if err := os.Chmod(tf.backingPath, os.FileMode(in.Mode)); err != nil {
			log.Printf("[FUSE] Chmod failed: %v", err)
			return syscall.EIO
		}
	}

	if in.Valid&fuse.FATTR_SIZE != 0 {
		log.Printf("[FUSE] Truncating to size: %d", in.Size)
		
		// For encrypted files, we need to handle truncation through the interceptor
		if tf.guardPoint != nil {
			// Check if user has write permission for truncation
			op := &filesystem.FileOperation{
				Type:   "write",
				Path:   tf.virtualPath,
				Data:   make([]byte, in.Size), // Create buffer of target size
				UID:    uid,
				GID:    gid,
				PID:    pid,
				Binary: binary,
			}

			result, err := tf.interceptor.InterceptWrite(ctx, op)
			if err != nil || !result.Allowed {
				log.Printf("[FUSE] Truncate denied: %v", err)
				return syscall.EACCES
			}
		}
		
		if err := os.Truncate(tf.backingPath, int64(in.Size)); err != nil {
			log.Printf("[FUSE] Truncate failed: %v", err)
			return syscall.EIO
		}
	}

	info, err := os.Stat(tf.backingPath)
	if err != nil {
		log.Printf("[FUSE] Stat failed: %v", err)
		return syscall.EIO
	}

	out.Attr = fileInfoToAttr(info)
	return 0
}

func (fh *TransparentFileHandle) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Read: path=%s, uid=%d, gid=%d, pid=%d, binary=%s", fh.virtualPath, uid, gid, pid, binary)

	op := &filesystem.FileOperation{
		Type:   "read",
		Path:   fh.virtualPath,
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := fh.interceptor.InterceptOpen(ctx, op)
	log.Printf("[FUSE] Read result: allowed=%v, err=%v", result.Allowed, err)
	if err != nil || !result.Allowed {
		return nil, syscall.EACCES
	}

	if result.Encrypted && result.Data != nil {
		if off >= int64(len(result.Data)) {
			return fuse.ReadResultData([]byte{}), 0
		}

		end := off + int64(len(dest))
		if end > int64(len(result.Data)) {
			end = int64(len(result.Data))
		}

		return fuse.ReadResultData(result.Data[off:end]), 0
	}

	n, err := fh.file.ReadAt(dest, off)
	if err != nil && err.Error() != "EOF" {
		return nil, syscall.EIO
	}

	return fuse.ReadResultData(dest[:n]), 0
}

func (fh *TransparentFileHandle) Write(ctx context.Context, data []byte, off int64) (uint32, syscall.Errno) {
	// Get real user context from FUSE
	uid, gid, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Write: path=%s, offset=%d, size=%d, uid=%d, pid=%d, binary=%s", fh.virtualPath, off, len(data), uid, pid, binary)

	op := &filesystem.FileOperation{
		Type:   "write",
		Path:   fh.virtualPath,
		Data:   data,
		UID:    uid,
		GID:    gid,
		PID:    pid,
		Binary: binary,
	}

	result, err := fh.interceptor.InterceptWrite(ctx, op)
	if err != nil || !result.Allowed {
		log.Printf("[FUSE] Write denied: %v", err)
		return 0, syscall.EACCES
	}

	if result.Encrypted {
		// For encrypted files, the interceptor handles the actual write
		// We just need to return success to the application
		log.Printf("[FUSE] Write successful (encrypted): %d bytes", len(data))
		return uint32(len(data)), 0
	}

	// For non-encrypted files, write directly to backing file
	n, err := fh.file.WriteAt(data, off)
	if err != nil {
		log.Printf("[FUSE] Write failed: %v", err)
		return 0, syscall.EIO
	}

	log.Printf("[FUSE] Write successful: %d bytes", n)
	return uint32(n), 0
}

func (fh *TransparentFileHandle) Flush(ctx context.Context) syscall.Errno {
	// Get real user context from FUSE
	uid, _, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Flush: path=%s, uid=%d, pid=%d, binary=%s", fh.virtualPath, uid, pid, binary)

	if err := fh.file.Sync(); err != nil {
		log.Printf("[FUSE] Flush failed: %v", err)
		return syscall.EIO
	}
	
	log.Printf("[FUSE] Flush successful")
	return 0
}

func (fh *TransparentFileHandle) Release(ctx context.Context) syscall.Errno {
	if err := fh.file.Close(); err != nil {
		return syscall.EIO
	}
	return 0
}

func (fh *TransparentFileHandle) Fsync(ctx context.Context, flags uint32) syscall.Errno {
	// Get real user context from FUSE
	uid, _, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Fsync: path=%s, uid=%d, pid=%d, binary=%s", fh.virtualPath, uid, pid, binary)

	if err := fh.file.Sync(); err != nil {
		log.Printf("[FUSE] Fsync failed: %v", err)
		return syscall.EIO
	}

	log.Printf("[FUSE] Fsync successful")
	return 0
}

func (fh *TransparentFileHandle) Flock(ctx context.Context, owner uint64, lock *fuse.FileLock, flags uint32) syscall.Errno {
	// Get real user context from FUSE
	uid, _, pid := getRealUserContext(ctx)
	binary := getProcessBinaryFromPid(pid)

	log.Printf("[FUSE] Flock: path=%s, uid=%d, pid=%d, binary=%s, type=%d", fh.virtualPath, uid, pid, binary, lock.Typ)

	// For database operations, we need to support file locking
	// This is crucial for MariaDB/MySQL data consistency
	
	// Convert FUSE lock to syscall flock
	var lockType int
	switch lock.Typ {
	case 0: // F_RDLCK - Read lock
		lockType = syscall.LOCK_SH // Shared lock
		log.Printf("[FUSE] Setting shared lock")
	case 1: // F_WRLCK - Write lock  
		lockType = syscall.LOCK_EX // Exclusive lock
		log.Printf("[FUSE] Setting exclusive lock")
	case 2: // F_UNLCK - Unlock
		lockType = syscall.LOCK_UN // Unlock
		log.Printf("[FUSE] Unlocking")
	default:
		log.Printf("[FUSE] Unknown lock type: %d", lock.Typ)
		return syscall.EINVAL
	}

	if flags&fuse.FUSE_LK_FLOCK != 0 {
		// Use flock() syscall for file locking
		if err := syscall.Flock(int(fh.file.Fd()), lockType); err != nil {
			log.Printf("[FUSE] Flock failed: %v", err)
			return syscall.Errno(err.(syscall.Errno))
		}
	}

	log.Printf("[FUSE] Flock successful")
	return 0
}

