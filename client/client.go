package client

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/samber/lo"
)

type OIClient struct {
	c *okta.Client
	// Not sure if this is needed, the okta.NewClient returns context also, so storing it here for now
	ctx context.Context
}

func NewOIClient(apiToken, oktaOrgURL string) (*OIClient, error) {
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

func (oi *OIClient) GetUsersInGroup(wantGroupName string) ([]string, error) {
	filter := query.NewQueryParams(query.WithQ(wantGroupName))
	groups, _, err := oi.c.Group.ListGroups(oi.ctx, filter)

	if err != nil {
		return nil, err
	}

	var groupID string

	for _, group := range groups {
		if strings.EqualFold(group.Profile.Name, wantGroupName) {
			groupID = group.Id
		}
	}

	if groupID == "" {
		fmt.Println("Group not found")
		return nil, nil
	}

	users, _, err := oi.c.Group.ListGroupUsers(oi.ctx, groupID, query.NewQueryParams())

	if err != nil {
		return nil, err
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

	return foundUsers, nil
}

func (oi *OIClient) PrintUsersInGroups(wantGroupsName []string) error {
	foundUsers, err := oi.getUsersInGroupsUnion(wantGroupsName)
	if err != nil {
		return err
	}

	for _, user := range foundUsers {
		fmt.Println(user)
	}

	return nil
}

// PrintGroupDiff prints the difference of two sets of groups
func (oi *OIClient) PrintGroupDiff(groupsA, groupsB []string, hideDeprovisioned bool) error {
	groupsAUsers, err := oi.getUsersInGroupsUnion(groupsA)
	if err != nil {
		return err
	}

	groupBUsers, err := oi.getUsersInGroupsUnion(groupsB)
	if err != nil {
		return err
	}

	notInB, notInA := lo.Difference(groupsAUsers, groupBUsers)

	groupA := strings.Join(groupsA, ", ")
	groupB := strings.Join(groupsB, ", ")

	headerStringFmt := "Users in %s, but not in %s:\n"
	if hideDeprovisioned {
		headerStringFmt = "Users (excluding deprovisioned) in %s, but not in %s:\n"
	}

	fmt.Printf(headerStringFmt, groupA, groupB)
	for _, user := range notInB {
		if strings.Contains(user, "(DEPROVISIONED)") && hideDeprovisioned {
			continue
		}

		fmt.Println(user)
	}
	fmt.Println()

	fmt.Printf(headerStringFmt, groupB, groupA)
	for _, user := range notInA {
		if strings.Contains(user, "(DEPROVISIONED)") && hideDeprovisioned {
			continue
		}

		fmt.Println(user)
	}

	return nil
}

// getUsersInGroupsUnion returns a deduplicated slice of users who are in at least one of the given groups
func (oi *OIClient) getUsersInGroupsUnion(groups []string) ([]string, error) {
	users := make([]string, 0)

	for _, group := range groups {
		groupUsers, err := oi.GetUsersInGroup(group)
		if err != nil {
			return nil, err
		}
		users = append(users, groupUsers...)
	}

	// dedup
	users = lo.Uniq(users)

	// sort
	sort.Strings(users)

	return users, nil
}
