package client

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/samber/lo"
)

const deprovisionedUserStatus = "DEPROVISIONED"

type OIClient struct {
	c *okta.Client
	// Not sure if this is needed, the okta.NewClient returns context also, so storing it here for now
	ctx                    context.Context
	showDeprovisionedUsers bool
}

func NewOIClient(apiToken, oktaOrgURL string, showDeprovisionedUsers bool) (*OIClient, error) {
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
		c:                      client,
		ctx:                    ctx,
		showDeprovisionedUsers: showDeprovisionedUsers,
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

		// searching for username with email address
		if strings.Contains(wantUserName, "@") {
			if strings.EqualFold(profileEmail, wantUserName) {
				userID = user.Id
			}
		} else { // no email address, just name
			// strip host out from email
			profileUserName := strings.Split(profileEmail, "@")[0]

			if strings.EqualFold(profileUserName, wantUserName) {
				userID = user.Id
			}
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
		if !oi.showDeprovisionedUsers && strings.Contains(user, deprovisionedUserStatus) {
			continue
		}
		fmt.Println(user)
	}

	return nil
}

// PrintGroupDiff prints the difference of two sets of groups
func (oi *OIClient) PrintGroupDiff(groupsA, groupsB []string) error {
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
	if !oi.showDeprovisionedUsers {
		headerStringFmt = "Users (excluding deprovisioned) in %s, but not in %s:\n"
	}

	fmt.Printf(headerStringFmt, groupA, groupB)
	for _, user := range notInB {
		if !oi.showDeprovisionedUsers && strings.Contains(user, deprovisionedUserStatus) {
			continue
		}

		fmt.Println(user)
	}
	fmt.Println()

	fmt.Printf(headerStringFmt, groupB, groupA)
	for _, user := range notInA {
		if !oi.showDeprovisionedUsers && strings.Contains(user, deprovisionedUserStatus) {
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

type OktaGroup struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

func (oi *OIClient) ListGroups() ([]OktaGroup, error) {
	var oktaGroups []OktaGroup

	addToGroup := func(g []*okta.Group) {
		for _, group := range g {
			oktaGroups = append(oktaGroups, OktaGroup{
				Name: group.Profile.Name,
				ID:   group.Id,
			})
		}
	}

	qp := query.NewQueryParams() // default limit per docs is 10_000

	respGroups, resp, err := oi.c.Group.ListGroups(context.TODO(), qp)
	if err != nil {
		return nil, err
	}
	addToGroup(respGroups)

	// Pagination
	for resp.HasNextPage() {
		respGroups = nil
		resp, err = resp.Next(context.TODO(), &respGroups)
		if err != nil {
			return nil, err
		}

		addToGroup(respGroups)
	}

	return oktaGroups, nil
}

type OktaGroupRule struct {
	Name               string   `json:"name"`
	ID                 string   `json:"id"`
	DestinationGroupID string   `json:"destination_group_id"`
	SourceGroupIDs     []string `json:"source_group_ids"`
	// Currently we don't support Users assigned via rule, but rather manually to the group
	// SourceUserIDs      []string `json:"user_ids"`
}

func (oi *OIClient) ListGroupRules(searchString string) ([]OktaGroupRule, error) {
	var oktaGroupRules []OktaGroupRule

	addToGroupRule := func(gr []*okta.GroupRule) error {
		for _, groupRule := range gr {
			ogr := OktaGroupRule{
				Name:               groupRule.Name,
				ID:                 groupRule.Id,
				DestinationGroupID: groupRule.Actions.AssignUserToGroups.GroupIds[0],
			}

			if groupRule.Actions == nil || groupRule.Actions.AssignUserToGroups == nil || len(groupRule.Actions.AssignUserToGroups.GroupIds) != 1 {
				return fmt.Errorf("group rule %s has no destination group", groupRule.Name)
			}
			ogr.DestinationGroupID = groupRule.Actions.AssignUserToGroups.GroupIds[0]

			if groupRule.Conditions == nil || groupRule.Conditions.Expression == nil {
				return fmt.Errorf("group rule %s has no conditions", groupRule.Name)
			}
			expression := groupRule.Conditions.Expression.Value
			ogr.SourceGroupIDs = parseGroupRuleExpression(expression)

			oktaGroupRules = append(oktaGroupRules, ogr)
		}

		return nil
	}

	opts := []query.ParamOptions{
		query.WithLimit(200), // max limit per docs
	}
	// use search string if not empty
	if searchString != "" {
		opts = append(opts, query.WithSearch(searchString))
	}
	qp := query.NewQueryParams(opts...)

	groupRules, resp, err := oi.c.Group.ListGroupRules(context.TODO(), qp)
	if err != nil {
		return nil, err
	}

	err = addToGroupRule(groupRules)
	if err != nil {
		return nil, err
	}

	// pagination
	for resp.HasNextPage() {
		groupRules = nil
		resp, err = resp.Next(context.TODO(), &groupRules)
		if err != nil {
			return nil, err
		}

		err = addToGroupRule(groupRules)
		if err != nil {
			return nil, err
		}
	}

	return oktaGroupRules, nil
}

func regexMatcher(expression *regexp.Regexp, matchString string, regexGroupMatch bool) []string {
	var regexMatches []string
	matches := expression.FindAllStringSubmatch(matchString, -1)

	for _, match := range matches {
		switch regexGroupMatch {
		case false:
			if match[0] != "" {
				regexMatches = append(regexMatches, match[0])
			}
		case true:
			if len(match) < 2 {
				continue
			}

			if match[1] != "" {
				regexMatches = append(regexMatches, match[1])
			}
		}
	}
	return regexMatches
}

// OR and AND. The AND  doesn't work properly because the output is just a slice of strings which we infer as OR, not AND
var reDividers = regexp.MustCompile(`\|\||&&`)

// this probably doesn't work properly with `isMemberOfGroupNameStartsWith` since it won't give the actual groups, just the prefix
var reGroupPrefixes = regexp.MustCompile(`^"?isMemberOf.*Group.*`)
var reGroupRuleExpression = regexp.MustCompile(`."(.+?)"`)

// parseGroupRuleExpression parses the expression string from Okta API response
// and returns a slice of group IDs. See TestParseGroupRuleExpression for example input and output.
func parseGroupRuleExpression(groupRules string) []string {
	var groupIDs []string

	divided := reDividers.Split(groupRules, -1)
	for i := range divided {
		trimmedString := strings.TrimSpace(divided[i])
		prefixParse := regexMatcher(reGroupPrefixes, trimmedString, false)
		for _, s := range prefixParse {
			ruleParse := regexMatcher(reGroupRuleExpression, s, true)
			groupIDs = append(groupIDs, ruleParse...)
		}
	}
	return groupIDs
}
