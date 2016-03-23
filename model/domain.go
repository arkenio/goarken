package model


type Domain struct {
	NodeKey string
	Name string
	Typ   string
	Value string
}


func (domain *Domain) Equals(other *Domain) bool {
	if domain == nil && other == nil {
		return true
	}

	return domain != nil && other != nil &&
		domain.Typ == other.Typ && domain.Value == other.Value
}
