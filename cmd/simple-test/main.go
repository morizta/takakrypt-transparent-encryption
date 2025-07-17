package main

import (
	"fmt"
	"os"
)

func main() {
	// Test if we can create files in the mount point when agent is running
	fmt.Println("Testing file creation in mounted filesystem...")
	
	// Test with a simple write
	file, err := os.Create("/tmp/test-mount/simple.txt")
	if err != nil {
		fmt.Printf("❌ Create failed: %v\n", err)
		return
	}
	defer file.Close()
	
	_, err = file.WriteString("Hello World!")
	if err != nil {
		fmt.Printf("❌ Write failed: %v\n", err)
		return
	}
	
	fmt.Println("✅ File created and written successfully!")
	
	// Test reading
	data, err := os.ReadFile("/tmp/test-mount/simple.txt")
	if err != nil {
		fmt.Printf("❌ Read failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ File content: %s\n", string(data))
}