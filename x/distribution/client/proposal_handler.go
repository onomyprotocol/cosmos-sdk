package client

import (
	"github.com/onomyprotocol/cosmos-sdk/x/distribution/client/cli"
	"github.com/onomyprotocol/cosmos-sdk/x/distribution/client/rest"
	govclient "github.com/onomyprotocol/cosmos-sdk/x/gov/client"
)

// ProposalHandler is the community spend proposal handler.
var (
	ProposalHandler = govclient.NewProposalHandler(cli.GetCmdSubmitProposal, rest.ProposalRESTHandler)
)
