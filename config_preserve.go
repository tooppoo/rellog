package rellog

import (
	"fmt"

	"github.com/njreid/gokdl2/document"
)

type preserveConfig struct {
	PreviewScripts []string
	RunScripts     []string
}

func decodePreserveConfig(rellogNode *document.Node) preserveConfig {
	var config preserveConfig
	for _, node := range rellogNode.Children {
		if nodeName(node) != "preserve" {
			continue
		}
		for _, phaseNode := range node.Children {
			var scripts *[]string
			switch nodeName(phaseNode) {
			case "preview":
				scripts = &config.PreviewScripts
			case "run":
				scripts = &config.RunScripts
			default:
				continue
			}
			for _, scriptNode := range phaseNode.Children {
				if nodeName(scriptNode) == "script" && len(scriptNode.Arguments) == 1 {
					*scripts = append(*scripts, scriptNode.Arguments[0].ValueString())
				}
			}
		}
		break
	}
	return config
}

func validatePreserveConfig(rellogNode *document.Node) []checkError {
	var preserveNodes []*document.Node
	for _, node := range rellogNode.Children {
		if nodeName(node) == "preserve" {
			preserveNodes = append(preserveNodes, node)
		}
	}
	if len(preserveNodes) > 1 {
		return []checkError{{"error[rellog.preserve.duplicate]", "preserve must appear at most once. Remove the duplicate preserve node."}}
	}
	if len(preserveNodes) == 0 {
		return nil
	}

	preserveNode := preserveNodes[0]
	if len(preserveNode.Arguments) > 0 {
		return []checkError{{"error[rellog.preserve.argument_count]", "preserve must not have arguments. Remove the arguments from preserve."}}
	}
	if propertyName, ok := firstNodePropertyName(preserveNode); ok {
		return []checkError{{"error[rellog.preserve.unknown_property]", "unknown property on preserve: " + propertyName + ". Remove the property from preserve."}}
	}
	phaseNodes := map[string][]*document.Node{
		"preview": nil,
		"run":     nil,
	}
	for _, node := range preserveNode.Children {
		name := nodeName(node)
		if _, ok := phaseNodes[name]; !ok {
			return []checkError{{
				"error[rellog.preserve.unknown_node]",
				fmt.Sprintf("unknown node: preserve.%s\n\nRemove unknown nodes from %s.", name, configFile()),
			}}
		}
		phaseNodes[name] = append(phaseNodes[name], node)
	}

	for _, phaseName := range []string{"preview", "run"} {
		nodes := phaseNodes[phaseName]
		if len(nodes) > 1 {
			return []checkError{{
				"error[rellog.preserve." + phaseName + ".duplicate]",
				"preserve." + phaseName + " must appear at most once. Remove the duplicate " + phaseName + " node.",
			}}
		}
		if len(nodes) == 1 {
			if errs := validateScriptPhase("preserve."+phaseName, nodes[0]); len(errs) > 0 {
				return errs
			}
		}
	}
	return nil
}
