package agreements

type Item = legacyAgreement

func Load(path string) ([]legacyAgreement, error) {
	return loadLegacyAgreements(path)
}
