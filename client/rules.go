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

	return printGroupRules(searchGroup, rules)
}

func printGroupRules(searchGroup string, ogr []OktaGroupRule) error {
	var ogr2 []OktaGroupRule

	sourceMaxLength := 0

	for _, o := range ogr {
		// is this shouldAdd stuff even needed?
		shoulAdd := true

		if strings.EqualFold(o.DestinationGroupID, searchGroup) {
			shoulAdd = true
		}

		var wantedSourceGroupValue string
		for _, sourceGroupID := range o.SourceGroupIDs {
			sourceMaxLength = max(sourceMaxLength, len(sourceGroupID))
			if strings.EqualFold(sourceGroupID, searchGroup) {
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
			// add all
			ogr2 = append(ogr2, o)
		}
		// wantedValue is one of the sourceGroups, drop the other sourceGroups
		if strings.EqualFold(wantedSourceGroupValue, searchGroup) {
			ogrNew := OktaGroupRule{
				DestinationGroupID: o.DestinationGroupID,
				SourceGroupIDs:     []string{wantedSourceGroupValue},
			}
			ogr2 = append(ogr2, ogrNew)
		}
	}

	ogr = ogr2

	// separate slice for printing so we can get output alphabetically sorted
	var printSlice []string

	for _, o := range ogr {
		for _, sourceGroupID := range o.SourceGroupIDs {
			printSlice = append(printSlice, fmt.Sprintf("%-*s -> %s", sourceMaxLength, sourceGroupID, o.DestinationGroupID))
		}
	}

	slices.Sort(printSlice)
	for _, s := range printSlice {
		fmt.Println(s)
	}

	return nil
}
