package cli

import (
	"context"

	"postcli/internal/tui/substackconfigureui"
)

func runSubstackConfigureWizard(ctx context.Context) error {
	return substackconfigureui.Run(substackconfigureui.Deps{
		Ctx:         ctx,
		Publication: SubstackPublication,
		Cookie:      SubstackCookie,
		SendEmail:   SubstackSendEmail,
		PersistEnv:  persistEnvAndShell,
	})
}
