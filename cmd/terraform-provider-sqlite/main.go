package main

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/probablyclever/terraform-provider-sqlite/internal/provider"
)

func main() {
	if err := providerserver.Serve(context.Background(),
		provider.New,
		providerserver.ServeOpts{
			Address: "registry.terraform.io/probablyclever/sqlite",
		},
	); err != nil {
		log.Fatalf("failed to serve provider: %v", err)
	}
}
