package devices

type Item = legacyDevice

func Load(path string) ([]legacyDevice, error) {
	return loadLegacyDevices(path)
}
