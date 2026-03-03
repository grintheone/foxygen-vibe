package attachments

type Item = legacyAttachment

func Load(path string) ([]legacyAttachment, error) {
	return loadLegacyAttachments(path)
}
