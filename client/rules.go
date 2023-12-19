package client

import (
	"fmt"
	"slices"
	"strings"
	"sync"
)

// PrintGroupRules prints all the group rules that have the searchGroup as either source or destination
func (oi *OIClient) PrintGroupRules(searchGroup string) error {
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
		rules, err = oi.ListGroupRules(searchGroup)
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
	for i, rule := range rules {
		rule.DestinationGroupID = groupIDMap[rule.DestinationGroupID]
		sourceGroupIDs := make([]string, len(rule.SourceGroupIDs))
		for i, sourceGroupID := range rule.SourceGroupIDs {
			sourceGroupIDs[i] = groupIDMap[sourceGroupID]
		}
		rule.SourceGroupIDs = sourceGroupIDs

		rules[i] = rule
	}

	groupRulesString := filterRulesToFormatted(searchGroup, rules)
	fmt.Print(groupRulesString)

	return nil
}

// filterRulesToFormatted filters the rules to only include the ones that have searchGroup as either source or destination
// and formats them in a string that is ready to be printed to terminal
func filterRulesToFormatted(searchGroup string, ogr []OktaGroupRule) string {
	var filteredOgr []OktaGroupRule

	sourceMaxLength := 0

	// Only pick the rules that have searchGroup as either source or destination
	for _, o := range ogr {
		// is this shouldAdd stuff even needed?
		shoulAdd := true

		if strings.EqualFold(o.DestinationGroupID, searchGroup) {
			shoulAdd = true
		}

		var wantedSourceGroupValue string
		for _, sourceGroupID := range o.SourceGroupIDs {
			if strings.EqualFold(sourceGroupID, searchGroup) {
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
		if strings.EqualFold(o.DestinationGroupID, searchGroup) {
			for _, sourceGroupID := range o.SourceGroupIDs {
				sourceMaxLength = max(sourceMaxLength, len(sourceGroupID))
			}
			// add all
			filteredOgr = append(filteredOgr, o)
		}
		// wantedValue is one of the sourceGroups, drop the other sourceGroups
		if strings.EqualFold(wantedSourceGroupValue, searchGroup) {
			ogrNew := OktaGroupRule{
				DestinationGroupID: o.DestinationGroupID,
				SourceGroupIDs:     []string{wantedSourceGroupValue},
			}
			filteredOgr = append(filteredOgr, ogrNew)
		}
	}

	// separate slice for printing so we can get output alphabetically sorted
	var printSlice []string

	for _, o := range filteredOgr {
		for _, sourceGroupID := range o.SourceGroupIDs {
			printSlice = append(printSlice, fmt.Sprintf("%-*s -> %s", sourceMaxLength, sourceGroupID, o.DestinationGroupID))
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
