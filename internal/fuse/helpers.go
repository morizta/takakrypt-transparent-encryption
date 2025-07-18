package fuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fuse"
)

func fileInfoToAttr(info os.FileInfo) fuse.Attr {
	attr := fuse.Attr{
		Size:   uint64(info.Size()),
		Mode:   uint32(info.Mode()),
		Mtime:  uint64(info.ModTime().Unix()),
		Ctime:  uint64(info.ModTime().Unix()),
		Atime:  uint64(info.ModTime().Unix()),
	}

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		attr.Ino = stat.Ino
		attr.Nlink = uint32(stat.Nlink)
		attr.Uid = stat.Uid
		attr.Gid = stat.Gid
		attr.Rdev = uint32(stat.Rdev)
		attr.Blocks = uint64(stat.Blocks)
		
		// Use basic timestamp - more compatible
		attr.Mtime = uint64(info.ModTime().Unix())
		attr.Ctime = uint64(info.ModTime().Unix())
		attr.Atime = uint64(info.ModTime().Unix())
	} else {
		// Fallback: if we can't get system stat, use default permissions
		attr.Uid = 1000  // ntoi user
		attr.Gid = 1000  // ntoi group
		attr.Nlink = 1
	}

	return attr
}

// Extract real user context from FUSE operation
func getRealUserContext(ctx context.Context) (uid, gid, pid int) {
	// Try to get from FUSE context
	if caller, ok := fuse.FromContext(ctx); ok {
		uid, gid, pid = int(caller.Uid), int(caller.Gid), int(caller.Pid)
		log.Printf("[FUSE] Got context from FUSE: uid=%d, gid=%d, pid=%d", uid, gid, pid)
		return uid, gid, pid
	}
	// Fallback to current process
	uid, gid, pid = os.Getuid(), os.Getgid(), os.Getpid()
	log.Printf("[FUSE] Using fallback context: uid=%d, gid=%d, pid=%d", uid, gid, pid)
	return uid, gid, pid
}

func getProcessBinaryFromPid(pid int) string {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	binary, err := os.Readlink(exePath)
	if err != nil {
		return "unknown"
	}
	return binary
}