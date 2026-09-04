package render

import "strings"

func Template(text, userName, companyName, role string) string {
	replacer := strings.NewReplacer(
		"{{user_name}}", userName,
		"{{company_name}}", companyName,
		"{{role}}", role,
	)
	return replacer.Replace(text)
}
