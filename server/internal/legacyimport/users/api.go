package users

type Plan = importPlan
type User = legacyUser

func Load(path string) (importPlan, error) {
	return loadLegacyImportPlan(path)
}

func TrimLegacyPrefix(value string) string {
	return trimLegacyPrefix(value)
}
