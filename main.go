package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/popsu/okta-info/client"
)

var (
	oktaOrgURL = os.Getenv("OKTA_INFO_ORG_URL")
	apiToken   = os.Getenv("OKTA_INFO_API_TOKEN")
)

func run() error {
	// Check which subcommand was provided
	if len(os.Args) < 3 {
		fmt.Println("Please provide a subcommand and user/group name")
		os.Exit(1)
	}

	token, err := getAPIToken()
	if err != nil {
		return err
	}

	oic, err := client.NewOIClient(token, oktaOrgURL)
	if err != nil {
		return err
	}

	// Handle the subcommands
	switch os.Args[1] {
	case "group":
		return oic.PrintUsersInGroup(os.Args[2])
	case "user":
		return oic.PrintGroupsForUser(os.Args[2])
	default:
		fmt.Println("Invalid subcommand. Valid commands are: group and user")
		os.Exit(1)
	}
	// should not get here ever
	return nil
}

func main() {
	err := run()
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func getAPIToken() (string, error) {
	if apiToken != "" {
		return apiToken, nil
	}

	if os.Getenv("OKTA_INFO_USE_1PASSWORD") == "" {
		return "", nil
	}

	// Use 1password vault to fetch token
	// This probably doesn't work for anyone else than me, sorry
	cmd := exec.Command("op", "item", "get",
		"product-Okta ApiToken",
		"--vault", "Private",
		"--field", "password")

	var outb, errb bytes.Buffer

	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err := cmd.Run()

	if err != nil {
		fmt.Println(outb.String())
		fmt.Println(errb.String())
		return "", err
	}

	// trim extra whitespace
	return strings.TrimSpace(outb.String()), nil
}
