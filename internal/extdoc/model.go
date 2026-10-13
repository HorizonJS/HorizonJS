package extdoc

import "strings"

// These models mirror the documented concepts in the original Extdoc tool,
// without its JAXB/XSLT intermediate representation.
type Description struct {
	Long, Short string
	HasShort    bool
}

type Param struct {
	Name, Type, Description string
	Optional                bool
}

type CustomTag struct{ Title, Value string }

type Member struct {
	Kind, Name, Type, Owner, ShortOwner, Href string
	Description                               Description
	Params                                    []Param
	ReturnType, ReturnDescription             string
	Static, Hide, Optional                    bool
	Custom                                    []CustomTag
}

type Class struct {
	Name, ShortName, Package, ParentName, Href, Description string
	DefinedIn                                               []string
	Singleton, Hide, HasConstructor, Component              bool
	ConstructorDescription                                  Description
	Params                                                  []Param
	Custom                                                  []CustomTag
	Members                                                 []*Member
	Ancestors, Subclasses                                   []*Class
}

type anchor struct {
	Offset int
	ID     string
}
type sourceFile struct {
	Name, Target, Content string
	Anchors               []anchor
}
type catalog struct {
	Classes []*Class
	ByName  map[string]*Class
	Members []*Member
	Files   []*sourceFile
	current *Class
}

func splitPackage(name string) (string, string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		return "", name
	}
	firstClass := len(parts) - 1
	for firstClass > 1 && startsUpper(parts[firstClass-1]) {
		firstClass--
	}
	return strings.Join(parts[:firstClass], "."), strings.Join(parts[firstClass:], ".")
}
