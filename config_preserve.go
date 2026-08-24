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
			if errs := validatePreservePhase(phaseName, nodes[0]); len(errs) > 0 {
				return errs
			}
		}
	}
	return nil
}

func validatePreservePhase(phaseName string, phaseNode *document.Node) []checkError {
	phasePath := "preserve." + phaseName
	if len(phaseNode.Arguments) > 0 {
		return []checkError{{"error[rellog." + phasePath + ".argument_count]", phasePath + " must not have arguments. Remove the arguments from " + phasePath + "."}}
	}
	if propertyName, ok := firstNodePropertyName(phaseNode); ok {
		return []checkError{{"error[rellog." + phasePath + ".unknown_property]", "unknown property on " + phasePath + ": " + propertyName + ". Remove the property from " + phasePath + "."}}
	}
	for _, scriptNode := range phaseNode.Children {
		if nodeName(scriptNode) != "script" {
			return []checkError{{
				"error[rellog." + phasePath + ".unknown_node]",
				fmt.Sprintf("unknown node: %s.%s\n\nRemove unknown nodes from %s.", phasePath, nodeName(scriptNode), configFile()),
			}}
		}
		if errs := validatePreserveScript(phasePath, scriptNode); len(errs) > 0 {
			return errs
		}
	}
	return nil
}

func validatePreserveScript(phasePath string, scriptNode *document.Node) []checkError {
	scriptPath := phasePath + ".script"
	if len(scriptNode.Arguments) != 1 {
		return []checkError{{"error[rellog." + scriptPath + ".argument_count]", scriptPath + " must have exactly one argument. Set one executable path as its argument."}}
	}
	if propertyName, ok := firstNodePropertyName(scriptNode); ok {
		return []checkError{{"error[rellog." + scriptPath + ".unknown_property]", "unknown property on " + scriptPath + ": " + propertyName + ". Remove the property from " + scriptPath + "."}}
	}
	if len(scriptNode.Children) > 0 {
		return []checkError{{"error[rellog." + scriptPath + ".unexpected_children]", scriptPath + " must not have child nodes. Remove its child nodes."}}
	}

	argument := scriptNode.Arguments[0]
	if _, ok := argument.Value.(string); !ok {
		return []checkError{{"error[rellog." + scriptPath + ".type]", scriptPath + " must be a string. Set its executable path as a quoted string."}}
	}
	if !isCanonicalScriptPath(argument.ValueString()) {
		return []checkError{{"error[rellog." + scriptPath + ".path]", scriptPath + ` must be a non-empty canonical project-root-relative path using / separators and no empty, "." or ".." segments.`}}
	}
	return nil
}

func firstNodePropertyName(node *document.Node) (string, bool) {
	var first string
	found := false
	for name := range node.Properties.Unordered() {
		if !found || name < first {
			first = name
			found = true
		}
	}
	return first, found
}
