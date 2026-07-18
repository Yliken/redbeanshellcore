package core

// CloneNodeRecord 返回 NodeRecord 的深副本。
// Registry 实现使用它隔离输入、输出和内部存储的可变 map/slice。
func CloneNodeRecord(record *NodeRecord) *NodeRecord {
	if record == nil {
		return nil
	}
	clone := *record
	clone.Config.Auth = cloneStringMap(record.Config.Auth)
	clone.Config.Options = cloneStringMap(record.Config.Options)
	clone.Config.Tags = cloneStrings(record.Config.Tags)
	clone.Capabilities = cloneCapabilities(record.Capabilities)
	clone.Metadata = cloneStringMap(record.Metadata)
	return &clone
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func cloneStrings(input []string) []string {
	if input == nil {
		return nil
	}
	return append([]string(nil), input...)
}

func cloneCapabilities(input []Capability) []Capability {
	if input == nil {
		return nil
	}
	return append([]Capability(nil), input...)
}
