package rellog

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

func checkConfigFile() *fileCheckResult {
	info, statErr := os.Stat(configFile())
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return &fileCheckResult{
				configFile(),
				[]checkError{{"error[config.missing] " + configFile(), "rellog configuration file does not exist."}},
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		msg := "rellog configuration file is not a regular file.\n\n" +
			"Expected a KDL file for config, but found a directory.\n" +
			"Remove the directory, then create a file:\n" +
			"  touch " + configFile()
		return &fileCheckResult{
			configFile(),
			[]checkError{{"error[config.not_file]", msg}},
		}
	}
	data, err := os.ReadFile(configFile())
	if err != nil {
		if os.IsPermission(err) {
			return &fileCheckResult{
				configFile(),
				[]checkError{{"error[config_file.permission]", "permission denied: " + configFile()}},
			}
		}
		return nil
	}
	content := string(data)
	if strings.HasPrefix(content, "/- kdl-version ") {
		firstLine := strings.SplitN(content, "\n", 2)[0]
		version := strings.TrimSpace(strings.TrimPrefix(firstLine, "/- kdl-version "))
		if version != "2" {
			return &fileCheckResult{
				configFile(),
				[]checkError{{"error[kdl-version.unsupported]", "rellog supports only KDL v2"}},
			}
		}
	}
	doc, parseErr := kdl.Parse(strings.NewReader(content))
	if parseErr != nil {
		msg := configFile() + ": " + parseErr.Error() + "\n\n" +
			"Failed to parse rellog configuration file.\n\n" +
			"Fix the KDL syntax error and run `rellog check` again."
		return &fileCheckResult{
			configFile(),
			[]checkError{{"error[config.parse_error]", msg}},
		}
	}
	if errs := validateRellogConfig(doc); len(errs) > 0 {
		return &fileCheckResult{configFile(), errs}
	}
	return nil
}

func nodeName(n *document.Node) string {
	if n.Name == nil {
		return ""
	}
	return n.Name.ValueString()
}

// kind id must match [A-Za-z][a-z0-9]*(?:-[a-z0-9]+)*
var validKindRe = regexp.MustCompile(`^[A-Za-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

var builtinKinds = map[string]bool{"empty": true}

var validTargetPolicies = map[string]bool{
	"deny-unknown":  true,
	"warn-unknown":  true,
	"allow-unknown": true,
}

type entryValidationConfig struct {
	allowedKinds map[string]bool
	kindTitles   map[string]string
	knownTargets map[string]bool
	targetPolicy string
}

func readEntryValidationConfig() (entryValidationConfig, error) {
	data, err := os.ReadFile(configFile())
	if err != nil {
		return entryValidationConfig{}, err
	}
	doc, err := kdl.Parse(strings.NewReader(string(data)))
	if err != nil {
		return entryValidationConfig{}, err
	}

	cfg := entryValidationConfig{
		allowedKinds: map[string]bool{},
		kindTitles:   map[string]string{},
		knownTargets: map[string]bool{},
		targetPolicy: "deny-unknown",
	}

	var entriesNode *document.Node
	for _, n := range doc.Nodes {
		if nodeName(n) != "rellog" {
			continue
		}
		for _, child := range n.Children {
			switch nodeName(child) {
			case "entries":
				entriesNode = child
			}
		}
		break
	}
	if entriesNode == nil {
		return cfg, nil
	}

	for _, n := range entriesNode.Children {
		switch nodeName(n) {
		case "target-policy":
			if len(n.Arguments) > 0 {
				cfg.targetPolicy = n.Arguments[0].ValueString()
			}
		case "kinds":
			for _, kindNode := range n.Children {
				if nodeName(kindNode) == "kind" && len(kindNode.Arguments) == 1 {
					id := kindNode.Arguments[0].ValueString()
					cfg.allowedKinds[id] = true
					if titleVal, ok := kindNode.Properties.Get("title"); ok {
						if title := strings.TrimSpace(titleVal.ValueString()); title != "" {
							cfg.kindTitles[id] = title
						}
					}
				}
			}
		case "targets":
			for _, targetNode := range n.Children {
				if nodeName(targetNode) == "target" && len(targetNode.Arguments) == 1 {
					cfg.knownTargets[targetNode.Arguments[0].ValueString()] = true
				}
			}
		}
	}

	return cfg, nil
}

func validateRellogConfig(doc *document.Document) []checkError {
	var rellogNode *document.Node
	rellogCount := 0
	for _, n := range doc.Nodes {
		if nodeName(n) == "rellog" {
			rellogCount++
			rellogNode = n
		}
	}
	if rellogCount != 1 {
		return []checkError{{"error[rellog.missing]", "configuration file must contain exactly one top-level rellog node."}}
	}

	allowedRootChildren := map[string]bool{
		"paths":   true,
		"entries": true,
		"consume": true,
	}
	for _, n := range rellogNode.Children {
		name := nodeName(n)
		if !allowedRootChildren[name] {
			return []checkError{{
				"error[rellog.unknown_node]",
				fmt.Sprintf("unknown node: %s\n\nRemove unknown nodes from %s.", name, configFile()),
			}}
		}
	}

	var pathsNode *document.Node
	for _, n := range rellogNode.Children {
		if nodeName(n) == "paths" {
			pathsNode = n
			break
		}
	}
	if pathsNode == nil {
		return []checkError{{"error[rellog.paths.missing]", "rellog.paths is required but not found"}}
	}
	if len(pathsNode.Arguments) > 0 {
		return []checkError{{"error[rellog.paths.missing]", "rellog.paths must be a block, but found a value"}}
	}

	type pathNodeInfo struct {
		present     bool
		hasChildren bool
		hasArgs     bool
		value       string
	}
	pathInfos := map[string]pathNodeInfo{}
	for _, n := range pathsNode.Children {
		info := pathNodeInfo{present: true}
		if len(n.Children) > 0 {
			info.hasChildren = true
		}
		if len(n.Arguments) > 0 {
			info.hasArgs = true
			info.value = n.Arguments[0].ValueString()
		}
		pathInfos[nodeName(n)] = info
	}

	pathValues := map[string]string{}
	var errs []checkError
	required := []string{"changelog", "entries", "release-notes"}
	for _, key := range required {
		info, ok := pathInfos[key]
		if !ok {
			errs = append(errs, checkError{
				"error[rellog.paths." + key + ".missing]",
				"rellog.paths." + key + " is required but not found",
			})
			continue
		}
		if info.hasChildren {
			errs = append(errs, checkError{
				"error[rellog.paths." + key + ".unexpected_children]",
				"rellog.paths." + key + " must not have child nodes.",
			})
			continue
		}
		if !info.hasArgs || strings.TrimFunc(info.value, unicode.IsSpace) == "" {
			errs = append(errs, checkError{
				"error[rellog.paths." + key + ".empty]",
				"rellog.paths." + key + " value cannot be empty.",
			})
			continue
		}
		pathValues[key] = info.value
	}
	if len(errs) > 0 {
		return errs
	}

	// Check for duplicate path values
	seen := map[string]string{} // path value → first key name
	for _, key := range required {
		val := pathValues[key]
		if firstKey, ok := seen[val]; ok {
			errs = append(errs, checkError{
				"error[rellog.paths." + key + ".conflict]",
				fmt.Sprintf("duplicate path: %q is used for both rellog.paths.%s and rellog.paths.%s", val, firstKey, key),
			})
		} else {
			seen[val] = key
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// Check for dot segments in path values (traversal prevention)
	for _, key := range required {
		val := pathValues[key]
		for _, segment := range strings.Split(val, "/") {
			if segment == "." || segment == ".." {
				errs = append(errs, checkError{
					"error[rellog.paths." + key + ".traversal]",
					fmt.Sprintf("%q is not allowed.\n\nconfiguration paths must be repository-root-relative paths and must not contain any dot segments.", val),
				})
				break
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// Check filesystem state for each path value
	if info, err := os.Stat(pathValues["changelog"]); err == nil && info.IsDir() {
		errs = append(errs, checkError{
			"error[config.rellog.paths.changelog.not-file]",
			"rellog.paths.changelog must be a file, but found a directory",
		})
	}
	if info, err := os.Stat(pathValues["entries"]); err == nil && !info.IsDir() {
		errs = append(errs, checkError{
			"error[config.rellog.paths.entries.not-dir]",
			"rellog.paths.entries must be a directory, but found a file",
		})
	}
	if info, err := os.Stat(pathValues["release-notes"]); err == nil && !info.IsDir() {
		errs = append(errs, checkError{
			"error[config.rellog.paths.release-notes.not-dir]",
			"rellog.paths.release-notes must be a directory, but found a file",
		})
	}
	if len(errs) > 0 {
		return errs
	}

	if errs := validateConsumeConfig(rellogNode); len(errs) > 0 {
		return errs
	}

	return validateEntriesConfig(rellogNode)
}

var validOnFailCreate = map[string]bool{
	"error":  true,
	"warn":   true,
	"ignore": true,
}

// readConsumeOnFailCreate returns the runtime value of consume.on-fail-create.
// Defaults to "error" if the node is absent.
func readConsumeOnFailCreate() (string, error) {
	data, err := os.ReadFile(configFile())
	if err != nil {
		return "", err
	}
	doc, err := kdl.Parse(strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	for _, n := range doc.Nodes {
		if nodeName(n) != "rellog" {
			continue
		}
		for _, child := range n.Children {
			if nodeName(child) != "consume" {
				continue
			}
			for _, gc := range child.Children {
				if nodeName(gc) == "on-fail-create" && len(gc.Arguments) > 0 {
					return gc.Arguments[0].ValueString(), nil
				}
			}
		}
		break
	}
	return "error", nil
}

func validateConsumeConfig(rellogNode *document.Node) []checkError {
	var consumeNodes []*document.Node
	for _, n := range rellogNode.Children {
		if nodeName(n) == "consume" {
			consumeNodes = append(consumeNodes, n)
		}
	}
	if len(consumeNodes) > 1 {
		return []checkError{{"error[rellog.consume.duplicate]", "consume must appear at most once."}}
	}
	if len(consumeNodes) == 0 {
		return nil
	}
	consumeNode := consumeNodes[0]

	if len(consumeNode.Arguments) > 0 {
		return []checkError{{"error[rellog.consume.argument_count]", "consume node must not have arguments."}}
	}

	for k := range consumeNode.Properties.Unordered() {
		return []checkError{{"error[rellog.consume.unknown_property]", "unknown property on consume: " + k}}
	}

	for _, child := range consumeNode.Children {
		if nodeName(child) != "on-fail-create" {
			return []checkError{{"error[rellog.consume.unexpected_children]", "consume may contain only on-fail-create."}}
		}
	}

	var onFailNodes []*document.Node
	for _, child := range consumeNode.Children {
		if nodeName(child) == "on-fail-create" {
			onFailNodes = append(onFailNodes, child)
		}
	}
	if len(onFailNodes) > 1 {
		return []checkError{{"error[rellog.consume.on-fail-create.duplicate]", "consume.on-fail-create must appear at most once."}}
	}
	if len(onFailNodes) == 0 {
		return nil
	}
	onFail := onFailNodes[0]

	if len(onFail.Arguments) == 0 {
		return []checkError{{"error[rellog.consume.on-fail-create.argument_count]", "consume.on-fail-create must have exactly one argument, but none was provided."}}
	}
	if len(onFail.Arguments) > 1 {
		args := make([]string, len(onFail.Arguments))
		for i, a := range onFail.Arguments {
			args[i] = `"` + a.ValueString() + `"`
		}
		return []checkError{{"error[rellog.consume.on-fail-create.argument_count]", "consume.on-fail-create must have exactly one argument.\nBut multiple arguments were provided: " + strings.Join(args, " ") + "."}}
	}

	for k := range onFail.Properties.Unordered() {
		return []checkError{{"error[rellog.consume.on-fail-create.unknown_property]", "unknown property on consume.on-fail-create: " + k}}
	}

	if len(onFail.Children) > 0 {
		return []checkError{{"error[rellog.consume.on-fail-create.unexpected_children]", "consume.on-fail-create must not have child nodes."}}
	}

	arg := onFail.Arguments[0]
	if _, ok := arg.Value.(string); !ok {
		return []checkError{{"error[rellog.consume.on-fail-create.type]", "consume.on-fail-create must be a string."}}
	}
	val := arg.ValueString()
	if !validOnFailCreate[val] {
		return []checkError{{"error[rellog.consume.on-fail-create.invalid]", fmt.Sprintf("consume.on-fail-create %q is not supported.\n\nAllowed values are: error, warn, ignore.", val)}}
	}

	return nil
}

func validateEntriesConfig(rellogNode *document.Node) []checkError {
	var entriesNode *document.Node
	for _, n := range rellogNode.Children {
		if nodeName(n) == "entries" {
			entriesNode = n
			break
		}
	}
	if entriesNode == nil {
		return []checkError{{"error[rellog.entries.missing]", "rellog.entries is required but not found."}}
	}

	// Validate target-policy if present
	for _, n := range entriesNode.Children {
		if nodeName(n) == "target-policy" {
			if len(n.Arguments) > 0 {
				val := n.Arguments[0].ValueString()
				if !validTargetPolicies[val] {
					return []checkError{{
						"error[rellog.entries.target-policy.invalid]",
						fmt.Sprintf("target-policy %q is not supported.\n\nAllowed values are: deny-unknown, warn-unknown, allow-unknown.", val),
					}}
				}
			}
			break
		}
	}

	var kindsNode *document.Node
	for _, n := range entriesNode.Children {
		if nodeName(n) == "kinds" {
			kindsNode = n
			break
		}
	}
	if kindsNode == nil {
		return []checkError{{"error[rellog.entries.kinds.missing]", "rellog.entries.kinds is required."}}
	}

	var values []string
	var perNodeErrs []checkError
	for _, n := range kindsNode.Children {
		if nodeName(n) != "kind" {
			continue
		}
		if len(n.Arguments) == 0 {
			perNodeErrs = append(perNodeErrs, checkError{
				"error[rellog.entries.kinds.kind.argument_count]",
				"kind node must have exactly one argument, but none was provided.",
			})
			continue
		}
		if len(n.Arguments) > 1 {
			args := make([]string, len(n.Arguments))
			for i, a := range n.Arguments {
				args[i] = `"` + a.ValueString() + `"`
			}
			perNodeErrs = append(perNodeErrs, checkError{
				"error[rellog.entries.kinds.kind.argument_count]",
				"kind node must have exactly one argument.\nBut multiple arguments were provided: " + strings.Join(args, " ") + ".",
			})
			continue
		}
		values = append(values, n.Arguments[0].ValueString())
	}
	if len(perNodeErrs) > 0 {
		return perNodeErrs
	}

	seen := map[string]bool{}
	var dupErrs []checkError
	for _, v := range values {
		if seen[v] {
			dupErrs = append(dupErrs, checkError{
				"error[rellog.entries.kinds.kind.id.duplicate]",
				fmt.Sprintf("The kind id %q is duplicated.", v),
			})
		}
		seen[v] = true
	}
	if len(dupErrs) > 0 {
		return dupErrs
	}

	var valErrs []checkError
	for _, v := range values {
		if builtinKinds[v] {
			valErrs = append(valErrs, checkError{
				"error[rellog.entries.kinds.kind.id.reserved]",
				fmt.Sprintf("%q is a reserved kind id and cannot be defined by the user.", v),
			})
		} else if !validKindRe.MatchString(v) {
			valErrs = append(valErrs, checkError{
				"error[rellog.entries.kinds.kind.id.invalid]",
				fmt.Sprintf("The kind id %q is invalid.\n\nkind id must match /[A-Za-z][a-z0-9]*(?:-[a-z0-9]+)*/", v),
			})
		}
	}
	return valErrs
}
