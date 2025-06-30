package provider

import (
	"context"
	"fmt"

	"golang.ngrok.com/ngrok/v2"
)

func (i *Ngrok) NgrokForwarder(vmEndpoint string, tunnelEndpoint *string) (ngrok.EndpointForwarder, error) {
	ctx, cancel := context.WithCancel(context.Background())

	i.NgrokCtx = append(i.NgrokCtx, NgCtx{
		CtxCancel: cancel,
		Ctx:       ctx,
	})

	ngURL := "tcp://"
	if tunnelEndpoint != nil {
		ngURL += *tunnelEndpoint
	}

	vmEndpoint = fmt.Sprintf("tcp://%v", vmEndpoint)
	a, err := ngrok.Forward(ctx, ngrok.WithUpstream(vmEndpoint), ngrok.WithURL(ngURL))
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
