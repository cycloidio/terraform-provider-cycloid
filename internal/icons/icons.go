package icons

import (
	"math/rand/v2"
)

var ValidIcons = []string{
	"folder_open",
	"mdi-cube-outline",
	"public",
	"extension",
	"science",
	"bug_report",
	"bolt",
	"call_merge",
	"commit",
	"mdi-source-branch",
	"traffic",
	"mdi-clipboard-check-outline",
	"mdi-progress-clock",
	"visibility",
	"vpn_key",
	"lightbulb",
	"favorite",
	"star",
	"auto_awesome",
	"mdi-controller-classic",
	"precision_manufacturing",
	"tour",
	"podcasts",
	"inventory",
	"save",
	"security",
	"mdi-lifebuoy",
	"mdi-ab-testing",
	"mdi-api",
	"mdi-console",
	"mdi-database",
	"mdi-vpn",
	"mdi-server",
	"mdi-server-security",
	"mdi-network-outline",
	"mdi-lan",
	"mdi-nas",
	"mdi-ansible",
	"mdi-aws",
	"mdi-microsoft-azure",
	"mdi-google-cloud",
	"mdi-kubernetes",
	"mdi-terraform",
}

var ValidColors = []string{
	"dev",
	"prod",
	"demo",
	"success",
	"default",
	"error",
	"staging",
	"preprod",
}

func RandomColor() string {
	return ValidColors[rand.IntN(len(ValidColors)-1)]
}

func RandomIcon() string {
	return ValidIcons[rand.IntN(len(ValidIcons)-1)]
}

func RandomColorAndIcon() (string, string) {
	return RandomColor(), RandomIcon()
}
