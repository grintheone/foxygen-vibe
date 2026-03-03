package ticketstatuses

type Lookup = ticketLookup

func Load(path string) ([]ticketLookup, error) {
	return loadLegacyTicketStatuses(path)
}
