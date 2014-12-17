package goarken

type Domain struct {
	typ   string
	value string
}

func (domain *Domain) Equals(other *Domain) bool {
	if domain == nil && other == nil {
		return true
	}

	return domain != nil && other != nil &&
		domain.typ == other.typ && domain.value == other.value
}
