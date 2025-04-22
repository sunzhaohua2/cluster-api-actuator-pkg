package mapi

import (
	"context"

	machinev1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/framework"
	"github.com/openshift/cluster-api-actuator-pkg/pkg/machinesetapi"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)


type Creator struct{
    	client runtimeclient.Client
}

func NewCreator(client runtimeclient.Client) *Creator {
	return &Creator{client: client}
}

func (c *Creator) Create(ctx context.Context, params machinesetapi.CreateParams) (*machinev1.MachineSet, error) {
    // 统一的 MAPI 实现逻辑
    // 所有平台共用同一套代码
    machineSetParams := framework.BuildMachineSetParams(ctx, c.client, 1)
    machineSet, err := framework.CreateMachineSet(c.client, machineSetParams)
	if err != nil {
		return nil, err
	}
	//Expect(err).ToNot(HaveOccurred(), "MachineSet should be able to be created")
	// 转换为通用 MachineSet 类型
	return machineSet, nil
}
