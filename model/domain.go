package model

type Domain struct {
	NodeKey string
	Name    string
	Typ     string
	Value   string
}

func (d *Domain) String() string {
	return d.Value + " at " + d.NodeKey
}

func (domain *Domain) Equals(other *Domain) bool {
	if domain == nil && other == nil {
		return true
	}

	return domain != nil && other != nil &&
		domain.Typ == other.Typ && domain.Value == other.Value
}
