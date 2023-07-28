package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func main() {
	// SSH connection details
	username := "admin"
	password := "admin@123"

	// Configuration file path (input from the user as a command-line argument)
	configFilePath := ""
	flag.StringVar(&configFilePath, "config", "", "Path to the configuration file")
	flag.Parse()

	// Check if the configuration file path is provided
	if configFilePath == "" {
		fmt.Println("Error: Configuration file path is required. Please use the '-config' flag.")
		os.Exit(1)
	}

	// Read the configuration file
	configData, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", err)
	}

	// Prompt the user to enter the Juniper nodes (comma-separated)
	fmt.Print("Enter the Juniper nodes (IP addresses or hostnames):seperated by comma\n")
	reader := bufio.NewReader(os.Stdin)
	nodesInput, _ := reader.ReadString('\n')
	nodesInput = strings.TrimSuffix(nodesInput, "\n")
	nodes := strings.Split(nodesInput, ",")

	// Iterate over each Juniper node
	for _, node := range nodes {
		node = strings.TrimSpace(node)

		// Create the SSH configuration
		sshConfig := &ssh.ClientConfig{
			User: username,
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		// Connect to the Juniper node
		client, err := ssh.Dial("tcp", node+":22", sshConfig)
		if err != nil {
			log.Printf("Failed to connect to %s: %s", node, err)
			continue
		}

		// Create a new session
		session, err := client.NewSession()
		if err != nil {
			log.Printf("Failed to create SSH session to %s: %s", node, err)
			client.Close()
			continue
		}

		// Get the session's standard input
		stdin, err := session.StdinPipe()
		if err != nil {
			log.Printf("Failed to get session's standard input: %s", err)
			session.Close()
			client.Close()
			continue
		}

		// Start the remote shell
		err = session.Shell()
		if err != nil {
			log.Printf("Failed to start shell on %s: %s", node, err)
			stdin.Close()
			session.Close()
			client.Close()
			continue
		}

		// Write the configuration data to the session's standard input
		go func() {
			defer stdin.Close()
			fmt.Fprintln(stdin, string(configData))
			fmt.Fprintln(stdin, "commit")
		}()

		// Wait for the session to finish
		err = session.Wait()
		if err != nil {
			log.Printf("Failed to execute configuration on %s: %s", node, err)
			continue
		}

		fmt.Printf("Successfully executed configuration on %s\n", node)

		// Close the session and client
		session.Close()
		client.Close()
	}
}
