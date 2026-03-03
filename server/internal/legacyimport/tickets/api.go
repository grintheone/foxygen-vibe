package tickets

type Item = legacyTicket
type Stats = ticketImportStats

func Load(path string) ([]legacyTicket, ticketImportStats, error) {
	return loadLegacyTickets(path)
}
