package main

import (
	"k8s.io/client-go/informers"
	"k8s.io/klog"
	"okn/pkg/agent"
	"okn/pkg/agent/cniserver"
	_ "okn/pkg/agent/cniserver/ipam"
	nodecontroller "okn/pkg/agent/controller/node"
	"okn/pkg/k8s"
)

type OKNAgent struct {
	informerFactory  informers.SharedInformerFactory
	nodeController   *nodecontroller.NodeController
	agentInitializer *agent.AgentInitializer
	config           *AgentConfig
}

func newOKNAgent(config *AgentConfig) (*OKNAgent, error) {
	client, err := k8s.CreateClient(config.ClientConnection)
	if err != nil {
		return nil, err
	}
	informerFactory := informers.NewSharedInformerFactory(client, 60)

	nodeController := nodecontroller.NewNodeController(client, informerFactory)

	ifaceStore := agent.NewInterfaceStore()

	agentInitializer := agent.NewInitializer(config.OVSBridge, config.HostGateway, config.TunnelType, client, ifaceStore)

	return &OKNAgent{
		informerFactory:  informerFactory,
		nodeController:   nodeController,
		agentInitializer: agentInitializer,
		config:           config,
	}, nil
}

func (agent *OKNAgent) run() error {
	klog.Info("Starting OKN Agent")

	// Initialize agent and node network
	err := agent.agentInitializer.SetupNodeNetwork()
	if err != nil {
		return err
	}
	defer agent.agentInitializer.Cleanup()

	cniServer := cniserver.New(
		agent.config.CNISocket,
		agent.agentInitializer.GetNodeConfig(),
		agent.agentInitializer.GetOVSBridgeClient(),
		agent.agentInitializer.GetInterfaceStore())

	stopCh := make(chan struct{})

	go cniServer.Run(stopCh)

	go agent.nodeController.Run(stopCh)

	agent.informerFactory.Start(stopCh)

	<-stopCh
	return nil
}