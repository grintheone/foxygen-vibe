package clients

type Item = legacyClient

func Load(path string) ([]legacyClient, error) {
	return loadLegacyClients(path)
}
