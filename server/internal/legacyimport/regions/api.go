package regions

type Item = legacyRegion

func Load(path string) ([]legacyRegion, error) {
	return loadLegacyRegions(path)
}
