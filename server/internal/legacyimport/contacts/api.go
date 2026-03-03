package contacts

type Item = legacyContact

func Load(path string) ([]legacyContact, error) {
	return loadLegacyContacts(path)
}
