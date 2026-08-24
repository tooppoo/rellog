package rellog

import (
	"fmt"

	"github.com/njreid/gokdl2/document"
)

func validateScriptPhase(phasePath string, phaseNode *document.Node) []checkError {
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
		if errs := validateScriptNode(phasePath, scriptNode); len(errs) > 0 {
			return errs
		}
	}
	return nil
}

func validateScriptNode(phasePath string, scriptNode *document.Node) []checkError {
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
