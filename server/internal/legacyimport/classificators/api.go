package classificators

type Item = legacyClassificator

func Load(path string) ([]legacyClassificator, error) {
	return loadLegacyClassificators(path)
}
