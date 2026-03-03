package ticketreasons

type Item = legacyTicketReason

func Load(path string) ([]legacyTicketReason, error) {
	return loadLegacyTicketReasons(path)
}
