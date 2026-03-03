package manufacturers

type Item = legacyManufacturer

func Load(path string) ([]legacyManufacturer, error) {
	return loadLegacyManufacturers(path)
}
