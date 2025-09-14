package luxwsclient

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/hansmi/wp2reg-luxws/luxws"
)

func findNavItemByName(name string, items []NavItem) *NavItem {
	for _, item := range items {
		if item.Name == name {
			return &item
		}

		if found := findNavItemByName(name, item.Items); found != nil {
			return found
		}
	}

	return nil
}

func NewNavRoot(rawXML []byte, wantLocalName string) (*NavRoot, error) {
	var cr NavRoot
	if err := xmlUnmarshal(rawXML, &cr); err != nil {
		return nil, fmt.Errorf("XML failed to decode NavRoot: %w", err)
	}
	if strings.ToLower(cr.XMLName.Local) == wantLocalName {
		return &cr, nil
	}
	return nil, luxws.ErrIgnore
}

func NewNavRootJSON(rawJSON []byte, wantLocalName string) (*NavRoot, error) {
	var cr NavigationJSON
	if err := json.Unmarshal(rawJSON, &cr); err != nil {
		return nil, fmt.Errorf("JSON failed to decode NavRoot: %w", err)
	}
	if strings.ToLower(cr.Type) == wantLocalName {
		var nr NavRoot
		nr.XMLName.Local = cr.Type
		nr.ID = cr.Id
		nr.Items = make([]NavItem, len(cr.Items))
		for i, item := range cr.Items {
			nr.Items[i] = NavItem{
				ID:   item.Id,
				Name: item.Name,
			}
			nr.Items[i].Items = make([]NavItem, len(item.Items))
			for j, item2 := range item.Items {
				nr.Items[i].Items[j] = NavItem{
					ID:   item2.Id,
					Name: item2.Name,
				}
				for k, item3 := range item2.Items {
					nr.Items[i].Items[j].Items = make([]NavItem, len(item3.Items))
					for l, item4 := range item3.Items {
						nr.Items[i].Items[j].Items[k].Items[l] = NavItem{
							ID:   item4.Id,
							Name: item4.Name,
						}
					}
				}
			}
		}
		return &nr, nil
	}
	return nil, luxws.ErrIgnore
}

// NavRoot represents the navigation structure of a LuxWS server.
type NavRoot struct {
	XMLName xml.Name
	ID      string    `xml:"id,attr"`
	Items   []NavItem `xml:"item"`
}

// FindByName iterates through all items and finds the first with a given name.
// Returns nil if none is found.
func (r *NavRoot) FindByName(name string) *NavItem {
	return findNavItemByName(name, r.Items)
}

// NavItem is an individual entry in the navigation structure.
type NavItem struct {
	ID    string    `xml:"id,attr"`
	Name  string    `xml:"name"`
	Items []NavItem `xml:"item"`
}
