package externalusers

type Item = legacyExternalUser
type Stats = importStats

func Load(path string) ([]legacyExternalUser, error) {
	return loadLegacyExternalUsers(path)
}
