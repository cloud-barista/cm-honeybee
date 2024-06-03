package common

import (
	agent "github.com/cloud-barista/cm-honeybee/agent/common"
	"github.com/cloud-barista/cm-honeybee/server/lib/config"
	"strings"
)

var agentRootURL = "http://AGENT_IP" + ":" + config.CMHoneybeeConfig.CMHoneybee.Agent.Port + "/" + strings.ToLower(agent.ShortModuleName)

type agentURL struct {
	Infra    string
	Software string
}

var AgentURL = agentURL{
	Infra:    agentRootURL + "/infra",
	Software: agentRootURL + "/software",
}
