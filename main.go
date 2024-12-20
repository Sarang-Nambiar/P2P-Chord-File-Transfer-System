package main

import (
	"distributed-chord/node"
	"distributed-chord/utils"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"
)

func showmenu() {
	red := "\033[31m"  // ANSI code for red text
	reset := "\033[0m" // ANSI code to reset color

	fmt.Println(red + "--------------------------------" + reset)
	fmt.Println(red + "\t\tMENU" + reset)
	fmt.Println(red + "Press 1 to see the fingertable" + reset)
	fmt.Println(red + "Press 2 to see the successor and predecessor" + reset)
	fmt.Println(red + "Press 3 to do the file transfer" + reset)
	fmt.Println(red + "Press 4 to see the menu" + reset)
	fmt.Println(red + "Press 5 to see all nodes in the network" + reset)
	fmt.Println(red + "--------------------------------" + reset)
}

func getAllNodes(n *node.Node) ([]node.Pointer, error) {
	nodes := []node.Pointer{}
	visited := make(map[int]bool)
	currentID := n.ID
	currentIP := n.IP
	nodes = append(nodes, node.Pointer{ID: currentID, IP: currentIP})
	visited[currentID] = true
	currentSuccessor := n.Successor

	for {
		if currentSuccessor.ID == n.ID {
			// We've come full circle
			break
		}
		if visited[currentSuccessor.ID] {
			// We've already visited this node
			break
		}

		client, err := rpc.Dial("tcp", currentSuccessor.IP)
		if err != nil {
			return nil, err
		}
		var successorInfo node.NodeInfo
		err = client.Call("Node.GetNodeInfo", struct{}{}, &successorInfo)
		client.Close()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node.Pointer{ID: successorInfo.ID, IP: successorInfo.IP})
		visited[successorInfo.ID] = true
		currentSuccessor = successorInfo.Successor
	}
	return nodes, nil
}

func main() {
	joinAddr := os.Getenv("BOOTSTRAP_ADDR")
	chordPort := os.Getenv("CHORD_PORT")

	containerIP, err := utils.GetContainerIP()

	fmt.Printf("Container IP: %s\n", containerIP)
	if err != nil {
		log.Fatalf("Failed to get container IP: %v", err)
	}
	n := node.CreateNode(containerIP + ":" + chordPort)

	fmt.Printf("Node %d created\n", n.ID)

	// Create a channel to signal when the RPC server is ready
	ready := make(chan bool)

	// Start the RPC server with the ready channel
	go n.StartRPCServer(ready)

	// Wait for the RPC server to be ready
	if serverReady := <-ready; !serverReady {
		log.Fatal("Failed to start RPC server")
	}

	if joinAddr != "" {
		// Join the network
		n.Join(joinAddr)
	}

	// Stabilize the chord network
	go n.Stabilize()
	// Update finger table
	go n.FixFingers()
	// Periodically check if predecessor is down
	go n.CheckPredecessor()

	showmenu()

	for {
		var choice int
		fmt.Print("Enter choice:")
		fmt.Scan(&choice)
		time.Sleep(1 * time.Second)

		switch choice {
		case 0:
			continue
		case 1:
			fmt.Println("Finger Table:")
			for i, entry := range n.FingerTable {
				// fmt.Printf("Finger table entry %d: Node %d (%s)\n", i+1, entry)
				if entry.ID != 0 {
					fmt.Printf("- Finger table entry %d: Node %d (%s)\n", i+1, entry.ID, entry.IP)
				} else {
					fmt.Printf("- Finger table entry %d: No node assigned\n", i+1)
				}
			}
		case 2:
			message := node.Message {
				ID: n.Successor.ID,
			}
			var reply node.Message
			n.FindSuccessor(message, &reply)
			fmt.Printf("Successor: %v, Predecessor: %v\n", reply.ID, n.Predecessor)
		case 3:
			var targetNodeID int
			var fileName string
			fmt.Print("Enter Target Node ID: ")
			fmt.Scan(&targetNodeID)
			// time.Sleep(5 * time.Second)
			fmt.Print("Enter the file name to transfer: ")
			fmt.Scan(&fileName)
			// time.Sleep(5 * time.Second)
			fmt.Printf("File transfer initiated successfully.\n")
			fmt.Printf("File Name: %s, Target Node IP: %d\n", fileName, targetNodeID)
			// time.Sleep(5 * time.Second)

			// Call a function to handle the file transfer (implement this function in node package)
			err := n.RequestFileTransfer(targetNodeID, fileName)

			if err != nil {
				fmt.Printf("File transfer failed: %v\n", err)
			}
			// 	fmt.Println("File transfer initiated successfully.")
			// }
		case 4:
			showmenu()
		case 5:
			nodes, err := getAllNodes(n)
			if err != nil {
				fmt.Printf("Error getting all nodes: %v\n", err)
			} else {
				fmt.Println("Nodes in the network:")
				for _, node := range nodes {
					fmt.Printf("Node ID: %d, IP: %s\n", node.ID, node.IP)
				}
			}
		default:
			// fmt.Println(choice)
			fmt.Println("Invalid choice")
		}
		time.Sleep(5 * time.Second)
	}
}
