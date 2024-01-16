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

func printHelp() {
	fmt.Println("Usage: okta-info <subcommand> <subcommand arguments>")
	fmt.Println("Subcommands:")
	fmt.Println("  group <group name> - print users in a group")
	fmt.Println("  user <user name> - print groups for a user")
	fmt.Println("  diff <group1,group2> <group3,group4> - print users in any of groups 1 or 2 but not in groups 3 or 4")
	fmt.Println("  rule [name/group] <rule name/group name> - print rules matching the search string or print group rules for a group")
}

func run() error {
	// Check which subcommand was provided
	if len(os.Args) < 3 {
		printHelp()
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
		// CommaSeparated list of groups
		groups := strings.Split(os.Args[2], ",")

		return oic.PrintUsersInGroups(groups)
	case "user":
		return oic.PrintGroupsForUser(os.Args[2])
	case "diff":
		// CommaSeparated list of groups
		groupsA := strings.Split(os.Args[2], ",")
		groupsB := strings.Split(os.Args[3], ",")

		hideDeprovisioned := false

		return oic.PrintGroupDiff(groupsA, groupsB, hideDeprovisioned)
	case "rule":
		switch os.Args[2] {
		case "group", "name":
			return oic.PrintGroupRules(os.Args[3], client.RuleType(os.Args[2]))
		default:
			printHelp()
			os.Exit(1)
		}
	default:
		printHelp()
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
