package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
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

	oic, err := NewOIClient()

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

type OIClient struct {
	c *okta.Client
	// Not sure if this is needed, the okta.NewClient returns context also, so storing it here for now
	ctx context.Context
}

func NewOIClient() (*OIClient, error) {
	apiToken, err := getAPIToken()
	if err != nil {
		return nil, err
	}

	ctx, client, err := okta.NewClient(
		context.TODO(),
		okta.WithOrgUrl(oktaOrgURL),
		okta.WithToken(apiToken),
	)

	if err != nil {
		// Okta URL missing
		if strings.Contains(err.Error(), "Okta URL is missing.") {
			fmt.Println("Okta org url missing. Please set OKTA_INFO_ORG_URL environment variable to your okta org url, It should look like https://<org>.okta.com")
			return nil, err
		}
		// API token missing
		if strings.Contains(err.Error(), "your Okta API token is missing") {
			fmt.Println("Okta API token missing or invalid. Please set OKTA_INFO_API_TOKEN environment variable to your okta API token")
		}

		return nil, err
	}

	return &OIClient{
		c:   client,
		ctx: ctx,
	}, nil
}

func (oi *OIClient) PrintGroupsForUser(wantUserName string) error {
	filter := query.NewQueryParams(query.WithQ(wantUserName))
	users, _, err := oi.c.User.ListUsers(oi.ctx, filter)

	if err != nil {
		return err
	}

	var userID string

	for _, user := range users {
		profile := *user.Profile
		profileEmail := profile["email"].(string)
		// strip host out from email
		profileUserName := strings.Split(profileEmail, "@")[0]

		if strings.EqualFold(profileUserName, wantUserName) {
			userID = user.Id
		}
	}

	if userID == "" {
		fmt.Println("User not found")
		return nil
	}

	groups, _, err := oi.c.User.ListUserGroups(oi.ctx, userID)

	if err != nil {
		return err
	}

	foundGroups := make([]string, 0, len(groups))

	for _, group := range groups {
		groupName := group.Profile.Name

		foundGroups = append(foundGroups, groupName)
	}

	sort.Strings(foundGroups)

	for _, group := range foundGroups {
		fmt.Println(group)
	}

	return nil
}

func (oi *OIClient) PrintUsersInGroup(wantGroupName string) error {
	filter := query.NewQueryParams(query.WithQ(wantGroupName))
	groups, _, err := oi.c.Group.ListGroups(oi.ctx, filter)

	if err != nil {
		return err
	}

	var groupID string

	for _, group := range groups {
		if strings.EqualFold(group.Profile.Name, wantGroupName) {
			groupID = group.Id
		}
	}

	if groupID == "" {
		fmt.Println("Group not found")
		return nil
	}

	users, _, err := oi.c.Group.ListGroupUsers(oi.ctx, groupID, query.NewQueryParams())

	if err != nil {
		return err
	}

	foundUsers := make([]string, 0, len(users))

	for _, user := range users {
		profile := *user.Profile

		email := profile["email"].(string)
		status := user.Status
		combined := fmt.Sprintf("%s (%s)", email, status)

		foundUsers = append(foundUsers, combined)
	}

	sort.Strings(foundUsers)

	for _, user := range foundUsers {
		fmt.Println(user)
	}

	return nil
}
