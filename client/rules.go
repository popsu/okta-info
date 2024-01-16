package client

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

type RuleType string

const (
	RuleTypeGroup RuleType = "group"
	RuleTypeName  RuleType = "name"
)

var reOktaGroupID = regexp.MustCompile(`^00g.{17}$`)

// PrintGroupRules prints all the group rules that have the searchGroup as either source or destination
func (oi *OIClient) PrintGroupRules(searchString string, ruletype RuleType) error {
	var wg sync.WaitGroup

	var groups []OktaGroup

	// TODO use errgroup.Group, it should be suited for this kind of use cases
	// https://pkg.go.dev/golang.org/x/sync/errgroup
	wg.Add(1)
	go func() {
		var err error
		groups, err = oi.ListGroups()
		if err != nil {
			panic(err)
		}
		wg.Done()
	}()

	var rules []OktaGroupRule

	wg.Add(1)
	go func() {
		var err error
		rules, err = oi.ListGroupRules(searchString)
		if err != nil {
			panic(err)
		}
		wg.Done()
	}()

	wg.Wait()

	// Create a map of groupID -> groupName
	groupIDMap := make(map[string]string)
	for _, group := range groups {
		groupIDMap[group.ID] = group.Name
	}

	// replace groupID with groupName in rules
	// groupName can also be in plain text - first see if there is a match with groupID
	// if not, treat groupName as plain text
	for i, rule := range rules {
		rule.DestinationGroupID = groupIDMap[rule.DestinationGroupID]
		sourceGroupIDs := make([]string, len(rule.SourceGroupIDs))
		for i, sourceGroupID := range rule.SourceGroupIDs {
			_, exists := groupIDMap[sourceGroupID]
			if exists {
				sourceGroupIDs[i] = groupIDMap[sourceGroupID]
			} else {
				// if sourceGroupID is not found and it matches the Okta group ID pattern,
				// the group does not exist in Okta anymore.
				match := reOktaGroupID.MatchString(sourceGroupID)
				if match {
					sourceGroupIDs[i] = sourceGroupID + " [missing in Okta!]"
				} else {
					sourceGroupIDs[i] = sourceGroupID
				}
			}
		}
		rule.SourceGroupIDs = sourceGroupIDs

		rules[i] = rule
	}

	groupRulesString := filterRulesToFormatted(searchString, rules, ruletype)
	fmt.Print(groupRulesString)

	return nil
}

// filterRulesToFormatted filters the rules to only include the ones that have searchGroup as either source or destination
// and formats them in a string that is ready to be printed to terminal
func filterRulesToFormatted(searchString string, ogr []OktaGroupRule, ruletype RuleType) string {
	var filteredOgr []OktaGroupRule

	nameMaxLength := 0
	sourceMaxLength := 0

	// Only pick the rules that have searchGroup as either source or destination
	for _, o := range ogr {
		// is this shouldAdd stuff even needed?
		shoulAdd := true

		nameMaxLength = max(nameMaxLength, len(o.Name))

		switch ruletype {
		case RuleTypeGroup:
			if strings.EqualFold(o.DestinationGroupID, searchString) {
				shoulAdd = true
			}

			var wantedSourceGroupValue string
			for _, sourceGroupID := range o.SourceGroupIDs {
				if strings.EqualFold(sourceGroupID, searchString) {
					sourceMaxLength = max(sourceMaxLength, len(sourceGroupID))
					shoulAdd = true
					// we need separate value to make sure Capitalization is proper
					wantedSourceGroupValue = sourceGroupID
				}
			}

			if !shoulAdd {
				continue
			}
			// Only add the dependency to/from wantedValue, ignore other rules
			if strings.EqualFold(o.DestinationGroupID, searchString) {
				for _, sourceGroupID := range o.SourceGroupIDs {
					sourceMaxLength = max(sourceMaxLength, len(sourceGroupID))
				}
				// add all
				filteredOgr = append(filteredOgr, o)
			}
			// wantedValue is one of the sourceGroups, drop the other sourceGroups
			if strings.EqualFold(wantedSourceGroupValue, searchString) {
				ogrNew := OktaGroupRule{
					Name:               o.Name,
					DestinationGroupID: o.DestinationGroupID,
					SourceGroupIDs:     []string{wantedSourceGroupValue},
				}
				filteredOgr = append(filteredOgr, ogrNew)
			}
		case RuleTypeName:
			if strings.EqualFold(o.Name, searchString) {
				for _, sourceGroupID := range o.SourceGroupIDs {
					sourceMaxLength = max(sourceMaxLength, len(sourceGroupID))
				}
				filteredOgr = append(filteredOgr, o)
			}
		}
	}

	// separate slice for printing so we can get output alphabetically sorted
	var printSlice []string

	for _, o := range filteredOgr {
		for _, sourceGroupID := range o.SourceGroupIDs {
			printSlice = append(printSlice, fmt.Sprintf("%-*s: %-*s -> %s", nameMaxLength, o.Name, sourceMaxLength, sourceGroupID, o.DestinationGroupID))
		}
	}

	slices.Sort(printSlice)

	var sb strings.Builder

	for _, s := range printSlice {
		sb.WriteString(s)
		sb.WriteString("\n")
	}

	return sb.String()
}
