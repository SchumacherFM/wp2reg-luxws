package luxwsclient

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/hansmi/wp2reg-luxws/luxws"
)

func NewContentRoot(rawXML []byte, wantLocalName string) (*ContentRoot, error) {
	var cr ContentRoot
	if err := xmlUnmarshal(rawXML, &cr); err != nil {
		return nil, fmt.Errorf("failed to decode ContentRoot: %w", err)
	}
	if strings.ToLower(cr.XMLName.Local) == wantLocalName {
		return &cr, nil
	}
	return nil, luxws.ErrIgnore
}

// ContentRoot contains all items returned by a GET request to a LuxWS server.
type ContentRoot struct {
	XMLName xml.Name
	Items   ContentItems `xml:"item"`
}

var ErrContentItemNotFound = errors.New("content item not found")

// FindByName iterates through all items and finds the first with a given name.
// Returns nil if none is found.
func (r *ContentRoot) FindByName(cmpFn CompareFn) (*ContentItem, error) {
	itm := r.Items.findContentItemByName(cmpFn)
	if itm == nil {
		return nil, ErrContentItemNotFound
	}
	return itm, nil
}

// ContentItem is an individual entry on a content page.
type ContentItem struct {
	ID      string               `xml:"id,attr"`
	Name    string               `xml:"name"`
	Min     *string              `xml:"min"`
	Max     *string              `xml:"max"`
	Step    *string              `xml:"step"`
	Unit    *string              `xml:"unit"`
	Div     *string              `xml:"div"`
	Raw     *string              `xml:"raw"`
	Value   *string              `xml:"value"`
	Columns []string             `xml:"columns"`
	Headers []string             `xml:"headers"`
	Options []*ContentItemOption `xml:"option"`
	Items   ContentItems         `xml:"item"`
}

type ContentItems []*ContentItem

func (ci *ContentItem) EachNonNil(cb func(*ContentItem)) {
	if ci == nil {
		return
	}
	for _, it := range ci.Items {
		if it.Value != nil {
			cb(it)
		}
	}
}

type CompareFn func(*ContentItem) bool

func CmpName(name string) CompareFn {
	return func(itm *ContentItem) bool {
		return name == itm.Name
	}
}

func CmpNameAndItems(name string) CompareFn {
	return func(itm *ContentItem) bool {
		return name == itm.Name && len(itm.Items) > 0
	}
}

func (items ContentItems) findContentItemByName(cmpFn CompareFn) *ContentItem {
	for _, i := range items {
		if cmpFn(i) {
			return i
		}
		if i2 := i.Items.findContentItemByName(cmpFn); i2 != nil {
			return i2
		}
	}
	return nil
}

// ContentItemOption represents one option among others of a content item.
type ContentItemOption struct {
	Value string `xml:"value,attr"`
	Name  string `xml:",chardata"`
}
