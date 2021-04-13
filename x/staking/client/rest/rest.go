package rest

import (
	"github.com/gorilla/mux"

	"github.com/onomyprotocol/cosmos-sdk/client"
	"github.com/onomyprotocol/cosmos-sdk/client/rest"
)

func RegisterHandlers(clientCtx client.Context, rtr *mux.Router) {
	r := rest.WithHTTPDeprecationHeaders(rtr)
	registerQueryRoutes(clientCtx, r)
	registerTxHandlers(clientCtx, r)
}
