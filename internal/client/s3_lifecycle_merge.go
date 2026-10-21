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

// LifecycleScope is the set of lifecycle actions a resource manages in a rule.
// Updated changes only those actions; the rule's other actions are kept.
type LifecycleScope int

const (
	// ScopeExpiration covers Expiration, NoncurrentVersionExpiration and
	// AbortIncompleteMultipartUpload.
	ScopeExpiration LifecycleScope = iota
	// ScopeTransition covers the rule's first Transition.
	ScopeTransition
	// ScopeNoncurrentTransition covers the rule's first NoncurrentVersionTransition.
	ScopeNoncurrentTransition
)

func textNode(name, text string) *xmlNode {
	return &xmlNode{XMLName: xml.Name{Local: name}, Text: text}
}

func tagNode(t BucketTag) *xmlNode {
	return &xmlNode{XMLName: xml.Name{Local: "Tag"}, Nodes: []*xmlNode{textNode("Key", t.Key), textNode("Value", t.Value)}}
}

// setFilter sets the rule's prefix and tags. A rule in the legacy form, with a
// top-level Prefix and no Filter, keeps that form while it has no tags. Filter
// conditions the provider doesn't model (such as object size) are kept, inside
// an And.
func setFilter(rule *xmlNode, prefix string, tags []BucketTag) {
	filter := rule.child("Filter")
	if filter == nil && rule.child("Prefix") != nil && len(tags) == 0 {
		rule.set("Prefix", prefix)
		return
	}
	rule.remove("Prefix")
	if filter == nil {
		filter = rule.ensure("Filter")
	}

	var extra []*xmlNode
	for _, c := range filter.Nodes {
		switch c.XMLName.Local {
		case "Prefix", "Tag":
		case "And":
			for _, a := range c.Nodes {
				if a.XMLName.Local != "Prefix" && a.XMLName.Local != "Tag" {
					extra = append(extra, a)
				}
			}
		default:
			extra = append(extra, c)
		}
	}

	switch {
	case len(extra) == 0 && len(tags) == 0:
		filter.Nodes = []*xmlNode{textNode("Prefix", prefix)}
	case len(extra) == 0 && len(tags) == 1 && prefix == "":
		filter.Nodes = []*xmlNode{tagNode(tags[0])}
	case len(tags) == 0 && prefix == "" && len(extra) == 1:
		filter.Nodes = extra
	default:
		and := &xmlNode{XMLName: xml.Name{Local: "And"}}
		if prefix != "" {
			and.Nodes = append(and.Nodes, textNode("Prefix", prefix))
		}
		for _, t := range tags {
			and.Nodes = append(and.Nodes, tagNode(t))
		}
		and.Nodes = append(and.Nodes, extra...)
		filter.Nodes = []*xmlNode{and}
	}
}

// setTransition sets a transition's timing (replacing the elements named in
// timing) and storage class, keeping its other children after them.
func setTransition(tr, when *xmlNode, storageClass string, timing ...string) {
	tr.remove(append(timing, "StorageClass")...)
	tr.Nodes = append([]*xmlNode{when, textNode("StorageClass", storageClass)}, tr.Nodes...)
}

// Updated returns rule r with status, filter and the actions in scope taken
// from next. The rest of a rule read from the server is kept.
func (r LifecycleRule) Updated(next LifecycleRule, scope LifecycleScope) (LifecycleRule, error) {
	if r.raw == "" {
		return next, nil
	}
	var rule xmlNode
	if err := xml.Unmarshal([]byte("<Rule>"+r.raw+"</Rule>"), &rule); err != nil {
		return LifecycleRule{}, fmt.Errorf("parsing lifecycle rule %q: %w", r.ID, err)
	}

	rule.set("Status", next.Status)
	setFilter(&rule, next.Prefix, next.Tags)

	switch scope {
	case ScopeExpiration:
		rule.remove("Expiration")
		switch {
		case next.ExpirationDays > 0:
			rule.ensure("Expiration").set("Days", strconv.Itoa(next.ExpirationDays))
		case next.ExpirationDate != "":
			rule.ensure("Expiration").set("Date", lifecycleDate(next.ExpirationDate))
		case next.ExpiredObjectDeleteMarker:
			rule.ensure("Expiration").set("ExpiredObjectDeleteMarker", "true")
		}
		if next.NoncurrentExpirationDays > 0 {
			rule.ensure("NoncurrentVersionExpiration").set("NoncurrentDays", strconv.Itoa(next.NoncurrentExpirationDays))
		} else {
			rule.remove("NoncurrentVersionExpiration")
		}
		if next.AbortMultipartDays > 0 {
			rule.ensure("AbortIncompleteMultipartUpload").set("DaysAfterInitiation", strconv.Itoa(next.AbortMultipartDays))
		} else {
			rule.remove("AbortIncompleteMultipartUpload")
		}
	case ScopeTransition:
		when := textNode("Days", strconv.Itoa(next.TransitionDays))
		if next.TransitionDate != "" {
			when = textNode("Date", lifecycleDate(next.TransitionDate))
		}
		setTransition(rule.ensure("Transition"), when, next.TransitionLocation, "Days", "Date")
	case ScopeNoncurrentTransition:
		setTransition(rule.ensure("NoncurrentVersionTransition"),
			textNode("NoncurrentDays", strconv.Itoa(next.NoncurrentTransitionDays)),
			next.NoncurrentTransitionLocation, "NoncurrentDays")
	}

	inner, err := rule.innerXML()
	if err != nil {
		return LifecycleRule{}, fmt.Errorf("encoding lifecycle rule %q: %w", r.ID, err)
	}
	next.raw = inner
	return next, nil
}
