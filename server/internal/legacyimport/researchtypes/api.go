package researchtypes

type Item = legacyResearchType

func Load(path string) ([]legacyResearchType, error) {
	return loadLegacyResearchTypes(path)
}
