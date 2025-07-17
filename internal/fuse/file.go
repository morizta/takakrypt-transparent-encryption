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
	if err != nil || !result.Allowed {
		return nil, 0, syscall.EACCES
	}

	file, err := os.OpenFile(tf.backingPath, int(flags), 0644)
	if err != nil {
		return nil, 0, syscall.EIO
	}

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
	info, err := os.Stat(tf.backingPath)
	if err != nil {
		return syscall.ENOENT
	}

	out.Attr = fileInfoToAttr(info)
	return 0
}

func (tf *TransparentFile) Setattr(ctx context.Context, fh fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if in.Valid&fuse.FATTR_MODE != 0 {
		if err := os.Chmod(tf.backingPath, os.FileMode(in.Mode)); err != nil {
			return syscall.EIO
		}
	}

	if in.Valid&fuse.FATTR_SIZE != 0 {
		if err := os.Truncate(tf.backingPath, int64(in.Size)); err != nil {
			return syscall.EIO
		}
	}

	info, err := os.Stat(tf.backingPath)
	if err != nil {
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
		return 0, syscall.EACCES
	}

	if result.Encrypted {
		return uint32(len(data)), 0
	}

	n, err := fh.file.WriteAt(data, off)
	if err != nil {
		return 0, syscall.EIO
	}

	return uint32(n), 0
}

func (fh *TransparentFileHandle) Flush(ctx context.Context) syscall.Errno {
	if err := fh.file.Sync(); err != nil {
		return syscall.EIO
	}
	return 0
}

func (fh *TransparentFileHandle) Release(ctx context.Context) syscall.Errno {
	if err := fh.file.Close(); err != nil {
		return syscall.EIO
	}
	return 0
}

