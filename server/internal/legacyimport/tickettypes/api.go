package tickettypes

type Lookup = ticketLookup

func Load(path string) ([]ticketLookup, error) {
	return loadLegacyTicketTypes(path)
}
