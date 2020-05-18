package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var (
	pAnnotations = flag.String("annotations", "", "annotations file path")
)

func main() {
	flag.Parse()
	annotationsBody, err := ioutil.ReadFile(*pAnnotations)
	die(err)

	config := bytes.NewBuffer(nil)
	config.WriteString("pid_file = \"./pid_file\"\n")

	annotationLines := strings.Split(string(annotationsBody), "\n")
	annotations := map[string]string{}
	for _, annotation := range annotationLines {
		annotation = strings.TrimSpace(annotation)
		if annotation == "" {
			continue
		}
		index := strings.Index(annotation, "=")
		key := annotation[:index]
		value := MustString(strconv.Unquote(annotation[index+1:]))
		annotations[key] = value
	}

	vaultAddr := annotations["vault-agent.dstream.cloud/vault"]
	config.WriteString("vault {\n")
	config.WriteString(fmt.Sprintf("  address =\"%s\"\n", vaultAddr))
	config.WriteString("}\n\n")

	config.WriteString("auto_auth {\n")
	{
		method := annotations["auth.vault-agent.dstream.cloud/method"]
		config.WriteString(fmt.Sprintf("  method \"%s\" {\n", method))
		switch method {
		case "kubernetes":
			config.WriteString("    config = {\n")
			config.WriteString(fmt.Sprintf("      role = \"%s\"\n", annotations["auth.vault-agent.dstream.cloud/kubernetes-role"]))
			config.WriteString("    }\n")
		}
		config.WriteString("  }\n")
	}

	config.WriteString("  sink \"file\" {\n")
	config.WriteString("    config =  {\n")
	config.WriteString("      path = \"/tmp/token\"\n")
	config.WriteString("    }\n")
	config.WriteString("  }\n")
	config.WriteString("}\n\n")

	templates := map[string]map[string]string{}

	setTemplate := func(name, key, value string) {
		if _, ok := templates[name]; !ok {
			templates[name] = map[string]string{}
		}
		templates[name][key] = value
	}

	for key, value := range annotations {
		if strings.HasPrefix(key, "env.vault-agent.dstream.cloud/") {
			name := key[len("env.vault-agent.dstream.cloud/"):]
			die(os.Setenv(name, value))
			continue
		}

		if strings.HasPrefix(key, "source.template.vault-agent.dstream.cloud/") {
			name := key[len("contents.template.vault-agent.dstream.cloud/"):]
			source := fmt.Sprintf("/tmp/tpl_%s", name)
			die(ioutil.WriteFile(source, []byte(value), 0644))
			setTemplate(name, "source", strconv.Quote(source))
		}

		if strings.HasPrefix(key, "contents.template.vault-agent.dstream.cloud/") {
			name := key[len("contents.template.vault-agent.dstream.cloud/"):]
			setTemplate(name, "contents", "<<EOF\n"+value+"\nEOF\n")
		}

		if strings.HasPrefix(key, "command.template.vault-agent.dstream.cloud/") {
			name := key[len("command.template.vault-agent.dstream.cloud/"):]
			setTemplate(name, "command", strconv.Quote(value))
		}

		if strings.HasPrefix(key, "destination.template.vault-agent.dstream.cloud/") {
			name := key[len("destination.template.vault-agent.dstream.cloud/"):]
			setTemplate(name, "destination", strconv.Quote(value))
		}
	}

	for _, template := range templates {
		config.WriteString("template {\n")
		for k, v := range template {
			config.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
		}
		config.WriteString("}\n\n")
	}

	config.WriteString("listener \"tcp\" {\n")
	config.WriteString("  address = \"127.0.0.1:8200\"\n")
	config.WriteString("  tls_disable = true\n")
	config.WriteString("}\n\n")

	die(ioutil.WriteFile("/tmp/vault-agent-config.hcl", config.Bytes(), 0644))
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}

func MustString(v string, err error) string {
	die(err)
	return v
}
