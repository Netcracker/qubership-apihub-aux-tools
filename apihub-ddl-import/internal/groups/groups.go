// Package groups creates one DDL table group per workbook domain in the
// published APIHUB version.
package groups

import (
	"errors"
	"fmt"
	"net/http"

	"apihub-ddl-import/internal/apihub"
	"apihub-ddl-import/internal/logx"
	"apihub-ddl-import/internal/model"
)

// Client is the slice of the APIHUB client the groups step needs.
type Client interface {
	CreateDdlGroup(packageID, version string, body apihub.CreateDdlGroupReq) error
	UpdateDdlGroup(packageID, version, groupName string, body apihub.UpdateDdlGroupReq) error
}

// Outcome summarizes the groups step for the report.
type Outcome struct {
	APIAvailable bool
	Created      []string
	Updated      []string
	Failed       []string
	Warnings     []model.Warning
}

// duplicate group error code (backend exception/ErrorCodes.go: DdlTableGroupAlreadyExists).
const codeGroupAlreadyExists = "8403"

// Create builds domain → []ddlEntityId from the merge result and the published
// entities, then creates (or, on republish, updates) one group per domain.
// A 404/405 from the first call means the deployed backend has no /ddl/groups
// yet: the step degrades to a single warning.
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

	for _, domain := range res.Domains {
		tables := tablesByDomain[domain]
		if len(tables) == 0 {
			logx.Debugf("domain %s has no published entities, group skipped", domain)
			continue
		}
		desc := fmt.Sprintf(descTemplate, domain)
		err := c.CreateDdlGroup(packageID, version, apihub.CreateDdlGroupReq{
			GroupName:   domain,
			Description: desc,
			Tables:      tables,
		})
		if err == nil {
			out.Created = append(out.Created, domain)
			logx.Okf("group %s created (%d tables)", domain, len(tables))
			continue
		}
		var ae *apihub.APIError
		if errors.As(err, &ae) {
			if ae.HTTPStatus == http.StatusNotFound || ae.HTTPStatus == http.StatusMethodNotAllowed || ae.HTTPStatus == http.StatusMisdirectedRequest {
				// Some deployed backends answer unmounted routes with 421
				// "Requested unknown endpoint" instead of a plain 404/405 —
				// observed live against a backend build without /ddl/groups.
				out.APIAvailable = false
				out.Warnings = append(out.Warnings, model.Warning{
					Code: model.WGroupsApiUnavailable,
					Msg: fmt.Sprintf("the APIHUB backend does not expose POST /ddl/groups (status %d) — "+
						"table groups are skipped; the Group column of the exports is still filled by the tool", ae.HTTPStatus),
				})
				// Zero groups ever get created once this triggers, which the
				// pipeline now always treats as fatal — log at error level,
				// not warning, to match.
				logx.Errorf("DDL groups API unavailable (status %d), skipping the groups step", ae.HTTPStatus)
				return out
			}
			if ae.Code == codeGroupAlreadyExists {
				upd := apihub.UpdateDdlGroupReq{Description: &desc, Tables: &tables}
				if uerr := c.UpdateDdlGroup(packageID, version, domain, upd); uerr == nil {
					out.Updated = append(out.Updated, domain)
					logx.Okf("group %s already existed, membership replaced (%d tables)", domain, len(tables))
					continue
				} else {
					err = uerr
				}
			}
		}
		out.Failed = append(out.Failed, domain)
		out.Warnings = append(out.Warnings, model.Warning{
			Code: model.WGroupCreateFailed,
			Msg:  fmt.Sprintf("group %s not created: %v", domain, err),
		})
		logx.Errorf("group %s: %v", domain, err)
	}
	return out
}
