package cli

import (
	"postcli/internal/config"
	"postcli/internal/schedule"
	"postcli/internal/store"
	"postcli/internal/substackapi"
	"postcli/internal/xapi"
)

// newAppPoster instantiates and configures a multi-channel poster.
func newAppPoster(st *store.Store) schedule.Poster {
	xClient := &xapi.Client{
		OAuth: xapi.OAuthConfig{
			ClientID:     ClientID(),
			ClientSecret: ClientSecret(),
			RedirectURI:  RedirectURI(),
		},
		TokenStore: st,
		TokenPath:  config.TokenPath(),
		DryRun:     DryRun(),
	}
	substackClient := &substackapi.Client{
		Cookie:      SubstackCookie(),
		Publication: SubstackPublication(),
		SendEmail:   SubstackSendEmail(),
		DryRun:      DryRun(),
	}
	return &schedule.AppPoster{
		X:        xClient,
		Substack: substackClient,
	}
}
