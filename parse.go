package main

import "github.com/PuerkitoBio/goquery"

// CheckGuestCodeRequired checks whether the guest code is required to access the property.
// This is done by checking whether the guest code input field exists on the page.
// This function is pretty experimental as I am still unsure how the guest code is handled on a property by property basis.
func CheckGuestCodeRequired(doc *goquery.Document) bool {
	return doc.Find("div.form-group > input#guestCode").Length() > 0
}

func GetFields(doc *goquery.Document) ([]Field, []string) {
	hiddenInputs := make([]string, 0, 5)
	formFields := make([]Field, 0, 5)

	doc.Find("#property-name-form > input[type=hidden]").Each(func(i int, s *goquery.Selection) {
		hiddenInputs = append(hiddenInputs, s.AttrOr("id", ""))
	})

	doc.Find("#property-name-form > div.form-group").Each(func(i int, s *goquery.Selection) {
		input := s.Find("input.form-control").First()
		label := s.Find("label").First().Text()

		inputId, _ := input.Attr("id")

		formFields = append(formFields, Field{
			text: label,
			id:   inputId,
		})
	})

	return formFields, hiddenInputs
}
