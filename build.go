// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	fmt.Println("🐳 Building Docker image...")
	
	cmd := exec.Command("docker", "build", "-t", "local-container-registry", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		log.Fatalf("❌ Docker build failed: %v", err)
	}
	
	fmt.Println("✅ Docker image built successfully!")
	fmt.Println("🚀 You can now run: docker run --rm -it local-container-registry")
}
