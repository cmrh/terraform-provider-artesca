package client

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

// xmlNode is a generic XML element, used to change a rule read from the
// server without losing elements the provider doesn't model.
type xmlNode struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Text    string     `xml:",chardata"`
	Nodes   []*xmlNode `xml:",any"`
}

func (n *xmlNode) child(name string) *xmlNode {
	for _, c := range n.Nodes {
		if c.XMLName.Local == name {
			return c
		}
	}
	return nil
}

// ensure returns the named child, appending it when missing.
func (n *xmlNode) ensure(name string) *xmlNode {
	if c := n.child(name); c != nil {
		return c
	}
	c := &xmlNode{XMLName: xml.Name{Local: name}}
	n.Nodes = append(n.Nodes, c)
	return c
}

func (n *xmlNode) remove(names ...string) {
	kept := n.Nodes[:0]
	for _, c := range n.Nodes {
		drop := false
		for _, name := range names {
			if c.XMLName.Local == name {
				drop = true
			}
		}
		if !drop {
			kept = append(kept, c)
		}
	}
	n.Nodes = kept
}

func (n *xmlNode) set(name, text string) {
	c := n.ensure(name)
	c.Text, c.Nodes = text, nil
}

// stripNamespaces drops the namespace the decoder copies from the response
// root onto every element, so the rule is written back without xmlns noise.
func (n *xmlNode) stripNamespaces() {
	n.XMLName.Space = ""
	attrs := n.Attrs[:0]
	for _, a := range n.Attrs {
		if a.Name.Space != "xmlns" && a.Name.Local != "xmlns" {
			attrs = append(attrs, a)
		}
	}
	n.Attrs = attrs
	for _, c := range n.Nodes {
		c.stripNamespaces()
	}
}

// innerXML encodes the element's children, the form lifecycleRule.Inner holds.
func (n *xmlNode) innerXML() (string, error) {
	n.stripNamespaces()
	var out []byte
	for _, c := range n.Nodes {
		b, err := xml.Marshal(c)
		if err != nil {
			return "", err
		}
		out = append(out, b...)
	}
	return string(out), nil
}

// setPrefix sets the rule's prefix where the rule keeps it: Filter/And/Prefix,
// Filter/Prefix, or the legacy top-level Prefix. A filter holding only a tag
// becomes an And of the prefix and the tag.
func setPrefix(rule *xmlNode, prefix string) {
	filter := rule.child("Filter")
	switch {
	case filter == nil && rule.child("Prefix") != nil:
		rule.set("Prefix", prefix)
	case filter == nil:
		rule.ensure("Filter").set("Prefix", prefix)
	case filter.child("And") != nil:
		filter.child("And").set("Prefix", prefix)
	case filter.child("Tag") != nil:
		if prefix == "" {
			return
		}
		tag := filter.child("Tag")
		filter.Nodes = []*xmlNode{{XMLName: xml.Name{Local: "And"}, Nodes: []*xmlNode{
			{XMLName: xml.Name{Local: "Prefix"}, Text: prefix}, tag,
		}}}
	default:
		filter.set("Prefix", prefix)
	}
}

// Updated returns rule r with the provider-modelled parts taken from next:
// status, prefix, and the expiration days or transition next sets. The rest
// of a rule read from the server (tag filters, other actions) is kept.
func (r LifecycleRule) Updated(next LifecycleRule) (LifecycleRule, error) {
	if r.raw == "" {
		return next, nil
	}
	var rule xmlNode
	if err := xml.Unmarshal([]byte("<Rule>"+r.raw+"</Rule>"), &rule); err != nil {
		return LifecycleRule{}, fmt.Errorf("parsing lifecycle rule %q: %w", r.ID, err)
	}

	rule.set("Status", next.Status)
	setPrefix(&rule, next.Prefix)
	if next.ExpirationDays > 0 {
		exp := rule.ensure("Expiration")
		exp.remove("Date", "ExpiredObjectDeleteMarker")
		exp.set("Days", strconv.Itoa(next.ExpirationDays))
	}
	if next.TransitionLocation != "" {
		tr := rule.ensure("Transition")
		tr.remove("Date")
		tr.set("Days", strconv.Itoa(next.TransitionDays))
		tr.set("StorageClass", next.TransitionLocation)
	}

	inner, err := rule.innerXML()
	if err != nil {
		return LifecycleRule{}, fmt.Errorf("encoding lifecycle rule %q: %w", r.ID, err)
	}
	next.raw = inner
	return next, nil
}
