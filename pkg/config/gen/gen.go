package main

import (
	cfg "github.com/conductorone/baton-linear/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("linear", cfg.Config)
}
