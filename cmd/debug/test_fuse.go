package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func TestFUSEOperations() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: debug test-fuse <command> [args]")
		fmt.Println("Commands:")
		fmt.Println("  db <path>     - Test database-like operations")
		fmt.Println("  lock <path>   - Test file locking")
		fmt.Println("  sync <path>   - Test sync operations")
		fmt.Println("  rename <path> - Test rename operations")
		return
	}

	command := os.Args[2]
	if len(os.Args) < 4 {
		fmt.Println("Error: path argument required")
		return
	}
	
	path := os.Args[3]
	
	switch command {
	case "db":
		testDatabaseOperations(path)
	case "lock":
		testFileLocking(path)
	case "sync":
		testSyncOperations(path)
	case "rename":
		testRenameOperations(path)
	default:
		fmt.Printf("Unknown test-fuse command: %s\n", command)
	}
}

func testDatabaseOperations(basePath string) {
	log.Println("Testing database-like operations...")
	
	// Simulate database file operations
	dbFile := filepath.Join(basePath, "test.db")
	
	// 1. Create and open file with O_SYNC for durability
	file, err := os.OpenFile(dbFile, os.O_CREATE|os.O_RDWR|os.O_SYNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create database file: %v", err)
	}
	defer file.Close()
	
	// 2. Write some data
	data := []byte("DATABASE RECORD 1\n")
	n, err := file.Write(data)
	if err != nil {
		log.Fatalf("Failed to write data: %v", err)
	}
	log.Printf("Wrote %d bytes", n)
	
	// 3. Sync to ensure durability
	if err := file.Sync(); err != nil {
		log.Fatalf("Failed to sync: %v", err)
	}
	log.Println("Synced successfully")
	
	// 4. Truncate to simulate database operations
	if err := file.Truncate(100); err != nil {
		log.Fatalf("Failed to truncate: %v", err)
	}
	log.Println("Truncated to 100 bytes")
	
	// 5. Write more data at specific offset
	data2 := []byte("DATABASE RECORD 2\n")
	n2, err := file.WriteAt(data2, 50)
	if err != nil {
		log.Fatalf("Failed to write at offset: %v", err)
	}
	log.Printf("Wrote %d bytes at offset 50", n2)
	
	// 6. Final sync
	if err := file.Sync(); err != nil {
		log.Fatalf("Failed to final sync: %v", err)
	}
	
	log.Println("Database operations completed successfully!")
}

func testFileLocking(basePath string) {
	log.Println("Testing file locking operations...")
	
	lockFile := filepath.Join(basePath, "test.lock")
	
	// Create lock file
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Failed to create lock file: %v", err)
	}
	defer file.Close()
	
	// Write PID to lock file
	pid := os.Getpid()
	if _, err := fmt.Fprintf(file, "%d\n", pid); err != nil {
		log.Fatalf("Failed to write PID: %v", err)
	}
	
	log.Printf("Lock file created with PID: %d", pid)
	
	// Test concurrent access simulation
	log.Println("Simulating database lock scenario...")
	time.Sleep(2 * time.Second)
	
	log.Println("File locking test completed!")
}

func testSyncOperations(basePath string) {
	log.Println("Testing sync operations...")
	
	testFile := filepath.Join(basePath, "sync-test.txt")
	
	// Open file
	file, err := os.OpenFile(testFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	defer file.Close()
	
	// Write data in chunks with sync
	for i := 0; i < 5; i++ {
		data := fmt.Sprintf("Chunk %d: %s\n", i, time.Now().Format(time.RFC3339))
		if _, err := file.WriteString(data); err != nil {
			log.Fatalf("Failed to write chunk %d: %v", i, err)
		}
		
		// Sync after each write
		if err := file.Sync(); err != nil {
			log.Fatalf("Failed to sync chunk %d: %v", i, err)
		}
		
		log.Printf("Written and synced chunk %d", i)
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Println("Sync operations completed successfully!")
}

func testRenameOperations(basePath string) {
	log.Println("Testing rename operations...")
	
	oldPath := filepath.Join(basePath, "rename-test-old.txt")
	newPath := filepath.Join(basePath, "rename-test-new.txt")
	
	// Create original file
	if err := os.WriteFile(oldPath, []byte("Test rename content\n"), 0644); err != nil {
		log.Fatalf("Failed to create test file: %v", err)
	}
	log.Printf("Created file: %s", oldPath)
	
	// Wait a bit
	time.Sleep(1 * time.Second)
	
	// Rename file
	if err := os.Rename(oldPath, newPath); err != nil {
		log.Fatalf("Failed to rename file: %v", err)
	}
	log.Printf("Renamed to: %s", newPath)
	
	// Verify content
	content, err := os.ReadFile(newPath)
	if err != nil {
		log.Fatalf("Failed to read renamed file: %v", err)
	}
	log.Printf("Content after rename: %s", string(content))
	
	// Clean up
	os.Remove(newPath)
	
	log.Println("Rename operations completed successfully!")
}