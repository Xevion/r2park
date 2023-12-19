package main

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// FormToComponents converts the form requested into usable modal components.
func FormToComponents(form GetFormResult) []discordgo.MessageComponent {
	components := make([]discordgo.MessageComponent, 0, 3)

	for _, field := range form.fields {
		var component discordgo.TextInput

		switch field.id {
		case "vehicleApt":
			component = discordgo.TextInput{
				CustomID:  "vehicleApt",
				Label:     "Apartment Number",
				Style:     discordgo.TextInputShort,
				Required:  true,
				MinLength: 1,
				MaxLength: 5,
			}
		case "vehicleMake":
			component = discordgo.TextInput{
				CustomID:    "vehicleMake",
				Label:       "Make",
				Placeholder: "Honda",
				Style:       discordgo.TextInputShort,
				Required:    true,
				MinLength:   4,
				MaxLength:   15,
			}
		case "vehicleModel":
			component = discordgo.TextInput{
				CustomID:    "vehicleModel",
				Label:       "Model",
				Style:       discordgo.TextInputShort,
				Placeholder: "Accord",
				Required:    true,
				MinLength:   1,
				MaxLength:   16,
			}
		case "vehicleLicensePlate":
			component = discordgo.TextInput{
				CustomID:    "vehicleLicensePlate",
				Label:       "License Plate",
				Placeholder: "ABC123",
				Style:       discordgo.TextInputShort,
				Required:    true,
				MinLength:   1,
				MaxLength:   9,
			}
		case "vehicleLicensePlateConfirm":
			log.Debug("Ignored field \"vehicleLicensePlateConfirm\"")
		default:
			log.Warnf("unexpected field handled for \"%s\" (%v, %s)", form.propertyName, field.id, field.text)
			component = discordgo.TextInput{
				CustomID:  field.id,
				Label:     field.text,
				Style:     discordgo.TextInputShort,
				Required:  true,
				MinLength: 1,
				MaxLength: 100,
			}
		}

		// Each field is contained within its own row.
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				component,
			},
		})
	}

	return components
}
