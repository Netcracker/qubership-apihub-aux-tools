// Package groups creates one DDL table group per workbook domain in the
// published APIHUB version.
package groups

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"apihub-ddl-import/internal/apihub"
	"apihub-ddl-import/internal/logx"
	"apihub-ddl-import/internal/model"
)

// Client is the slice of the APIHUB client the groups step needs.
type Client interface {
	ListDdlGroups(packageID, version string) ([]apihub.DdlGroup, error)
	ListDdlGroupEntityIDs(packageID, version, groupName string) ([]string, error)
	CreateDdlGroup(packageID, version string, body apihub.CreateDdlGroupReq) error
	UpdateDdlGroup(packageID, version, groupName string, body apihub.UpdateDdlGroupReq) error
}

// Outcome summarizes the groups step for the report.
type Outcome struct {
	APIAvailable bool
	Created      []string
	Updated      []string
	Unchanged    []string
	Failed       []string
	Warnings     []model.Warning
}

// duplicate group error code (backend exception/ErrorCodes.go: DdlTableGroupAlreadyExists).
const codeGroupAlreadyExists = "8403"

// Create builds domain → []ddlEntityId from the merge result and the published
// entities, then writes one group per domain.
//
// Publication copies DDL table groups onto the new version from the previous
// revision, or from the latest revision of the previous version. The target
// version may therefore have none of the domains, some of them, or all of
// them. A copied group's tables are read back and compared with the domain.
// An exact match is left as it is. Any other set is replaced, then read
// back again, and the group fails if the result is still not that set.
// Missing groups are created, then given their tables.
// A 404, 405, or 421 from the list call means this backend has no /ddl/groups.
func Create(c Client, packageID, version string, res *model.MergeResult, entities []apihub.DdlEntity, descTemplate string) *Outcome {
	out := &Outcome{APIAvailable: true}

	idByKey := make(map[string]string, len(entities))
	for _, e := range entities {
		idByKey[model.NormKey(e.Name)] = e.DdlEntityId
	}
	tablesByDomain := map[string][]apihub.DdlGroupTable{}
	for key, domain := range res.DomainByTable {
		id, ok := idByKey[key]
		if !ok {
			continue
		}
		tablesByDomain[domain] = append(tablesByDomain[domain], apihub.DdlGroupTable{DdlEntityId: id})
	}

	existing, err := c.ListDdlGroups(packageID, version)
	if err != nil {
		if apiUnavailable(err) {
			out.APIAvailable = false
			var ae *apihub.APIError
			errors.As(err, &ae)
			out.Warnings = append(out.Warnings, model.Warning{
				Code: model.WGroupsApiUnavailable,
				Msg: fmt.Sprintf("the APIHUB backend does not expose GET /ddl/groups (status %d) — "+
					"table groups are skipped; the Group column of the exports is still filled by the tool", ae.HTTPStatus),
			})
			logx.Errorf("DDL groups API unavailable (status %d), skipping the groups step", ae.HTTPStatus)
			return out
		}
		out.Failed = append(out.Failed, "ddl-groups")
		out.Warnings = append(out.Warnings, model.Warning{
			Code: model.WGroupCreateFailed,
			Msg:  fmt.Sprintf("list DDL groups: %v", err),
		})
		logx.Errorf("list DDL groups: %v", err)
		return out
	}
	present := make(map[string]struct{}, len(existing))
	for _, group := range existing {
		present[group.GroupName] = struct{}{}
	}

	for _, domain := range res.Domains {
		tables := tablesByDomain[domain]
		if len(tables) == 0 {
			logx.Debugf("domain %s has no published entities, group skipped", domain)
			continue
		}
		desc := fmt.Sprintf(descTemplate, domain)
		if _, ok := present[domain]; ok {
			syncCopiedGroup(c, out, packageID, version, domain, desc, tables)
			continue
		}
		if err := c.CreateDdlGroup(packageID, version, apihub.CreateDdlGroupReq{
			GroupName:   domain,
			Description: desc,
		}); err != nil {
			// The list can miss a group that publication copied between the
			// list and this create.
			if !groupAlreadyExists(err) {
				recordGroupFailure(out, domain, err)
				continue
			}
			syncCopiedGroup(c, out, packageID, version, domain, desc, tables)
			continue
		}
		if uerr := c.UpdateDdlGroup(packageID, version, domain, apihub.UpdateDdlGroupReq{Tables: &tables}); uerr != nil {
			recordGroupFailure(out, domain, uerr)
			continue
		}
		out.Created = append(out.Created, domain)
		logx.Okf("group %s created (%d tables)", domain, len(tables))
	}
	return out
}

// syncCopiedGroup checks a group publication already placed on this version.
// The tables must be exactly the domain's entity ids: the same set, with
// nothing missing and nothing extra. A different set is replaced and checked
// again.
func syncCopiedGroup(c Client, out *Outcome, packageID, version, domain, desc string, tables []apihub.DdlGroupTable) {
	match, got, err := copiedGroupMatches(c, packageID, version, domain, tables)
	if err != nil {
		recordGroupFailure(out, domain, err)
		return
	}
	if match {
		out.Unchanged = append(out.Unchanged, domain)
		logx.Okf("group %s already has the expected %d tables", domain, len(tables))
		return
	}
	upd := apihub.UpdateDdlGroupReq{Description: &desc, Tables: &tables}
	if uerr := c.UpdateDdlGroup(packageID, version, domain, upd); uerr != nil {
		recordGroupFailure(out, domain, uerr)
		return
	}
	match, got, err = copiedGroupMatches(c, packageID, version, domain, tables)
	if err != nil {
		recordGroupFailure(out, domain, err)
		return
	}
	if !match {
		recordGroupFailure(out, domain, fmt.Errorf("membership is [%s], want [%s]", formatIDs(got), formatIDs(tableIDs(tables))))
		return
	}
	out.Updated = append(out.Updated, domain)
	logx.Okf("group %s membership replaced (%d tables)", domain, len(tables))
}

func copiedGroupMatches(c Client, packageID, version, domain string, tables []apihub.DdlGroupTable) (bool, []string, error) {
	got, err := c.ListDdlGroupEntityIDs(packageID, version, domain)
	if err != nil {
		return false, nil, err
	}
	return sameEntityIDs(got, tables), got, nil
}

func tableIDs(tables []apihub.DdlGroupTable) []string {
	ids := make([]string, len(tables))
	for i, table := range tables {
		ids[i] = table.DdlEntityId
	}
	return ids
}

func sameEntityIDs(got []string, tables []apihub.DdlGroupTable) bool {
	want := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		if table.DdlEntityId == "" {
			return false
		}
		if _, dup := want[table.DdlEntityId]; dup {
			return false
		}
		want[table.DdlEntityId] = struct{}{}
	}
	seen := make(map[string]struct{}, len(got))
	for _, id := range got {
		if id == "" {
			return false
		}
		if _, dup := seen[id]; dup {
			return false
		}
		seen[id] = struct{}{}
	}
	if len(seen) != len(want) {
		return false
	}
	for id := range want {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}

func formatIDs(ids []string) string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}

func apiUnavailable(err error) bool {
	var ae *apihub.APIError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.HTTPStatus {
	case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusMisdirectedRequest:
		return true
	default:
		return false
	}
}

func groupAlreadyExists(err error) bool {
	var ae *apihub.APIError
	return errors.As(err, &ae) && ae.Code == codeGroupAlreadyExists
}

func recordGroupFailure(out *Outcome, domain string, err error) {
	out.Failed = append(out.Failed, domain)
	out.Warnings = append(out.Warnings, model.Warning{
		Code: model.WGroupCreateFailed,
		Msg:  fmt.Sprintf("group %s not written: %v", domain, err),
	})
	logx.Errorf("group %s: %v", domain, err)
}
