package provider

import (
	"context"

	"golang.ngrok.com/ngrok/v2"
)

func (i *Ngrok) NgrokForwarder(InstanceEP string, EndpointType string) (ngrok.EndpointForwarder, error) {
	ctx, cancel := context.WithCancel(context.Background())

	i.NgrokCtx = append(i.NgrokCtx, NgCtx{
		CtxCancel: cancel,
		Ctx:       ctx,
	})

	a, err := ngrok.Forward(ctx, ngrok.WithUpstream(InstanceEP), ngrok.WithURL(EndpointType))
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Stoping ngrok tunnel by CtxCancel()
func (i *Ngrok) NgrokStop(vmEndpoint string) {
	for _, v := range i.NgrokCtx {
		if vmEndpoint == v.VMendpoint {
			v.CtxCancel()
		}
	}
}
