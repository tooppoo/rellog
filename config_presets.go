package rellog

import "github.com/njreid/gokdl2/document"

func decodeUsePresetsConfig(rellogNode *document.Node) []string {
	var presetIDs []string
	for _, node := range rellogNode.Children {
		if nodeName(node) == "use-presets" && len(node.Arguments) == 1 {
			presetIDs = append(presetIDs, node.Arguments[0].ValueString())
		}
	}
	return presetIDs
}

func validateUsePresetsConfig(rellogNode *document.Node) []checkError {
	seen := map[string]bool{}
	for _, node := range rellogNode.Children {
		if nodeName(node) != "use-presets" {
			continue
		}
		if len(node.Arguments) != 1 {
			return []checkError{{"error[rellog.use-presets.argument_count]", "use-presets must have exactly one argument. Set one preset id as its argument."}}
		}
		if propertyName, ok := firstNodePropertyName(node); ok {
			return []checkError{{"error[rellog.use-presets.unknown_property]", "unknown property on use-presets: " + propertyName + ". Remove the property from use-presets."}}
		}
		if len(node.Children) > 0 {
			return []checkError{{"error[rellog.use-presets.unexpected_children]", "use-presets must not have child nodes. Remove its child nodes."}}
		}

		argument := node.Arguments[0]
		if _, ok := argument.Value.(string); !ok {
			return []checkError{{"error[rellog.use-presets.type]", "use-presets must be a string. Set its preset id as a quoted string."}}
		}
		presetID := argument.ValueString()
		if _, ok := builtinPresets[presetID]; !ok {
			return []checkError{{"error[rellog.use-presets.unknown]", "use-presets contains an unknown preset id. Use a supported preset id: rust."}}
		}
		if seen[presetID] {
			return []checkError{{"error[rellog.use-presets.duplicate]", "use-presets contains a duplicate preset id. Remove the duplicate use-presets node."}}
		}
		seen[presetID] = true
	}
	return nil
}
