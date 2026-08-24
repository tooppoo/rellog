package rellog

import (
	"fmt"

	"github.com/njreid/gokdl2/document"
)

type readyConfig struct {
	VerifyScripts []string
}

func decodeReadyConfig(rellogNode *document.Node) readyConfig {
	var config readyConfig
	for _, node := range rellogNode.Children {
		if nodeName(node) != "ready" {
			continue
		}
		for _, verifyNode := range node.Children {
			if nodeName(verifyNode) != "verify" {
				continue
			}
			for _, scriptNode := range verifyNode.Children {
				if nodeName(scriptNode) == "script" && len(scriptNode.Arguments) == 1 {
					config.VerifyScripts = append(config.VerifyScripts, scriptNode.Arguments[0].ValueString())
				}
			}
			break
		}
		break
	}
	return config
}

func validateReadyConfig(rellogNode *document.Node) []checkError {
	var readyNodes []*document.Node
	for _, node := range rellogNode.Children {
		if nodeName(node) == "ready" {
			readyNodes = append(readyNodes, node)
		}
	}
	if len(readyNodes) > 1 {
		return []checkError{{"error[rellog.ready.duplicate]", "ready must appear at most once. Remove the duplicate ready node."}}
	}
	if len(readyNodes) == 0 {
		return nil
	}

	readyNode := readyNodes[0]
	if len(readyNode.Arguments) > 0 {
		return []checkError{{"error[rellog.ready.argument_count]", "ready must not have arguments. Remove the arguments from ready."}}
	}
	if propertyName, ok := firstNodePropertyName(readyNode); ok {
		return []checkError{{"error[rellog.ready.unknown_property]", "unknown property on ready: " + propertyName + ". Remove the property from ready."}}
	}

	var verifyNodes []*document.Node
	for _, node := range readyNode.Children {
		if nodeName(node) != "verify" {
			return []checkError{{
				"error[rellog.ready.unknown_node]",
				fmt.Sprintf("unknown node: ready.%s\n\nRemove unknown nodes from %s.", nodeName(node), configFile()),
			}}
		}
		verifyNodes = append(verifyNodes, node)
	}
	if len(verifyNodes) > 1 {
		return []checkError{{"error[rellog.ready.verify.duplicate]", "ready.verify must appear at most once. Remove the duplicate verify node."}}
	}
	if len(verifyNodes) == 1 {
		return validateScriptPhase("ready.verify", verifyNodes[0])
	}
	return nil
}
