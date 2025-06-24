package pkg

import (
	"context"
	"fmt"

	"golang.ngrok.com/ngrok/v2"
)

func NgrokForwarder(ctx context.Context, endpoint string) (ngrok.EndpointForwarder, error) {
	endpoint = fmt.Sprintf("tcp://%v", endpoint)
	a, err := ngrok.Forward(ctx, ngrok.WithUpstream(endpoint), ngrok.WithURL("tcp://"))
	if err != nil {
		return nil, err
	}

	return a, nil
}
