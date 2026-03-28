package reconciliation

import "github.com/marcosfpina/O.W.A.S.A.K.A/internal/models"

// ChangeType classifies a reconciliation change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "ASSET_ADDED"
	ChangeRemoved  ChangeType = "ASSET_REMOVED"
	ChangeModified ChangeType = "ASSET_MODIFIED"
)

// Change describes a single drift between two snapshots.
type Change struct {
	Type    ChangeType     `json:"type"`
	AssetID string         `json:"asset_id"`
	Fields  map[string]any `json:"fields,omitempty"` // old→new for modified fields
}

// Diff compares a previous snapshot against the current one and returns changes.
func Diff(prev, curr []models.Asset) []Change {
	prevMap := indexAssets(prev)
	currMap := indexAssets(curr)

	var changes []Change

	// Detect additions and modifications
	for id, ca := range currMap {
		pa, existed := prevMap[id]
		if !existed {
			changes = append(changes, Change{
				Type:    ChangeAdded,
				AssetID: id,
				Fields: map[string]any{
					"ip":       ca.IP,
					"hostname": ca.Hostname,
					"os":       ca.OS,
				},
			})
			continue
		}
		if fields := diffAsset(pa, ca); len(fields) > 0 {
			changes = append(changes, Change{
				Type:    ChangeModified,
				AssetID: id,
				Fields:  fields,
			})
		}
	}

	// Detect removals
	for id := range prevMap {
		if _, exists := currMap[id]; !exists {
			changes = append(changes, Change{
				Type:    ChangeRemoved,
				AssetID: id,
			})
		}
	}

	return changes
}

func indexAssets(assets []models.Asset) map[string]models.Asset {
	m := make(map[string]models.Asset, len(assets))
	for _, a := range assets {
		m[a.ID] = a
	}
	return m
}

func diffAsset(prev, curr models.Asset) map[string]any {
	fields := make(map[string]any)
	if prev.IP != curr.IP {
		fields["ip"] = map[string]string{"old": prev.IP, "new": curr.IP}
	}
	if prev.MAC != curr.MAC {
		fields["mac"] = map[string]string{"old": prev.MAC, "new": curr.MAC}
	}
	if prev.Hostname != curr.Hostname {
		fields["hostname"] = map[string]string{"old": prev.Hostname, "new": curr.Hostname}
	}
	if prev.OS != curr.OS {
		fields["os"] = map[string]string{"old": prev.OS, "new": curr.OS}
	}
	if !portsEqual(prev.Ports, curr.Ports) {
		fields["ports"] = map[string]any{"old": prev.Ports, "new": curr.Ports}
	}
	return fields
}

func portsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[int]struct{}, len(a))
	for _, p := range a {
		set[p] = struct{}{}
	}
	for _, p := range b {
		if _, ok := set[p]; !ok {
			return false
		}
	}
	return true
}
