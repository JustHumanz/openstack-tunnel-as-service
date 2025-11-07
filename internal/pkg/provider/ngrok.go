package provider

import (
	"context"
	"fmt"

	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
	"golang.ngrok.com/ngrok/v2"
)

func (i *Ngrok) NgrokForwarder(InstanceEP string, EndpointType string) (ngrok.EndpointForwarder, error) {
	ctx, cancel := context.WithCancel(context.Background())

	i.NgrokCtx = append(i.NgrokCtx, NgCtx{
		VMendpoint: InstanceEP,
		CtxCancel:  cancel,
		Ctx:        ctx,
	})

	a, err := ngrok.Forward(ctx, ngrok.WithUpstream(InstanceEP), ngrok.WithURL(EndpointType))
	if err != nil {
		cancel() // cleanup
		return nil, err
	}

	return a, nil
}

// Stoping ngrok tunnel by CtxCancel()
func (i *Ngrok) NgrokStop(vmEndpoint string) {
	for index, v := range i.NgrokCtx {
		if pkg.StripScheme(v.VMendpoint) == pkg.StripScheme(vmEndpoint) {
			fmt.Println(v)
			i.NgrokCtx[index].CtxCancel()
			i.NgrokCtx = append(i.NgrokCtx[:index], i.NgrokCtx[index+1:]...)
		}
	}
}
