package fca

import (
	"distributed-chord/chord/node"
	"fmt"
	"io"

	// "math/big"
	"os"
	"path/filepath"

	// "sort"
	"strings"
)

func Chunker() *ChunkInfo {
	// Paths
	dataDir := "./data" // Change if needed
	const chunkSize = 1024
	var sourceFolder, sourceFile, absPath string

	// Initialize the ChunkInfo struct
	chunkInfo := &ChunkInfo{
		ChunkLocations: []string{},
		Name:           "",
	}

	// Loop to allow user to switch nodes or select a file
	for {
		// Select source node folder
		sourceFolder = promptUserForFolder(dataDir)
		if sourceFolder == "" {
			fmt.Println("No valid folder selected.")
			return nil
		}

		sourceFile, absPath = promptUserForFile(dataDir, sourceFolder)
		if sourceFile == "-1" { // If user enters -1, switch the node
			continue
		}
		if sourceFile == "" {
			fmt.Println("No valid file selected.")
			return nil
		}

		// Set the original file name in the ChunkInfo struct
		chunkInfo.Name = removeFileExtension(sanitizeFileName(absPath))
		break // Exit the loop after a valid file is selected
	}

	// Open the source file
	file, err := os.Open(sourceFile)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil
	}
	defer file.Close()

	// Get all the node folders and assign finger tables
	nodeFolders, err := getNodeFolders(dataDir, sourceFolder)
	if err != nil {
		fmt.Println("Error retrieving folders:", err)
		return nil
	}

	if len(nodeFolders) == 0 {
		fmt.Println("No other folders found for distributing chunks.")
		return nil
	}

	// var nodes []*node.Node
	// add finger tables implementation here

	buffer := make([]byte, chunkSize)
	chunkNumber := 1

	// Replace special characters in the absolute path to make it valid as a file name
	absPathForFileName := sanitizeFileName(absPath)

	// Remove the file extension from the absolute path
	absPathWithoutExtension := removeFileExtension(absPathForFileName)

	for {
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Println("Error reading file:", err)
			return nil
		}
		if bytesRead == 0 {
			break
		}

		// Create the chunk file name by appending the chunk number at the end of the sanitized path without extension
		chunkFileName := fmt.Sprintf("%s-chunk%d.txt", absPathWithoutExtension, chunkNumber)

		// Hash the chunk file name using SHA-1 and convert it to a big integer
		// hashedChunkFileName := node.hashSHA1(chunkFileName)
		// hashedChunkBigInt := node.hashToBigInt(hashedChunkFileName)
		hashedChunkBigInt := node.SHA1Hash(chunkFileName)

		// Find the appropriate node based on the chunk's big integer value
		assignedNode := node.findSuccessor(hashedChunkBigInt)

		if assignedNode != nil {
			// Save the chunk in the assigned node's folder
			destinationFolder := filepath.Join("", assignedNode.FolderName)
			chunkPath := filepath.Join(destinationFolder, chunkFileName)
			err = writeChunk(chunkPath, buffer[:bytesRead])
			if err != nil {
				fmt.Println("Error writing chunk:", err)
				return nil
			}
			fmt.Printf("Chunk %d assigned to node: %s\n", chunkNumber, filepath.Base(assignedNode.FolderName))

			// Append the chunk's location info to the ChunkLocations field in chunkInfo
			chunkInfo.ChunkLocations = append(chunkInfo.ChunkLocations, filepath.Base(assignedNode.FolderName))
		} else {
			fmt.Println("No node found for chunk:", chunkFileName)
		}

		chunkNumber++
	}

	return chunkInfo
}

func sanitizeFileName(path string) string {
	replacer := strings.NewReplacer("\\", "_", ":", "_")
	return replacer.Replace(path)
}

func removeFileExtension(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext)
}

func promptUserForFolder(dataDir string) string {
	folders, err := getNodeFolders(dataDir, "")
	if err != nil {
		fmt.Println("Error retrieving folders:", err)
		return ""
	}

	for {
		fmt.Println("Select the source node folder:")
		for i, folder := range folders {
			fmt.Printf("[%d] %s\n", i+1, filepath.Base(folder))
		}
		fmt.Println("[0] Create a new node")

		var choice int
		fmt.Print("Enter your choice: ")
		fmt.Scan(&choice)

		if choice == 0 {
			// Option to create a new node
			var newNodeName string
			fmt.Print("Enter the name of the new node: ")
			fmt.Scan(&newNodeName)

			// Construct the new node path
			newNodePath := filepath.Join(dataDir, newNodeName)

			// Create the new directory for the node
			err := os.Mkdir(newNodePath, os.ModePerm)
			if err != nil {
				fmt.Println("Error creating new node folder:", err)
				continue
			}

			fmt.Printf("New node '%s' created successfully.\n", newNodeName)
			fmt.Printf("Current node is '%s'\n", newNodeName)
			return newNodeName
		} else if choice >= 1 && choice <= len(folders) {
			return filepath.Base(folders[choice-1])
		}

		fmt.Println("Invalid choice. Please select a valid node.")
	}
}

func promptUserForFile(dataDir, folder string) (string, string) {
	for {
		var fileName string
		fmt.Print("Enter the file name (with extension) to share (or enter -1 to switch node): ")
		fmt.Scan(&fileName)

		if fileName == "-1" {
			return "-1", ""
		}
		filePath := filepath.Join(dataDir, folder, fileName)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Println("Invalid file name. Please try again.")
		} else {
			absPath, _ := filepath.Abs(filePath)
			return filePath, absPath
		}
	}
}

// Function to get all node folders in the data directory
func getNodeFolders(dataDir, sourceFolder string) ([]string, error) {
	var nodeFolders []string

	// Read the entries in the dataDir folder
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, err
	}

	// Loop through the entries and append only directories
	for _, entry := range entries {
		if entry.IsDir() {
			nodeFolders = append(nodeFolders, filepath.Join(dataDir, entry.Name()))
		}
	}

	return nodeFolders, nil
}

// write chunk data to a file
func writeChunk(filePath string, data []byte) error {
	fmt.Printf("Writing chunk to file: %s\n", filePath)
	return os.WriteFile(filePath, data, 0644)
}
